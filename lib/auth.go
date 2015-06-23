package lib

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"code.google.com/p/go.crypto/bcrypt"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/psc"
	"github.com/garyburd/redigo/redis"
)

var (
	//BadLogin = guess...
	BadLogin = gp.APIerror{Reason: "Bad username/password"}
	//BadPassword = you are trying to change your password but haven't given the correct old password.
	BadPassword = gp.APIerror{Reason: "The password you have provided is incorrect"}
	//MissingParamFirst = your first name wasn't long enough
	MissingParamFirst = gp.APIerror{Reason: "Missing parameter: first"}
	//MissingParamLast = your last name wasn't long enough
	MissingParamLast = gp.APIerror{Reason: "Missing parameter: last"}
	//MissingParamPass = your password wasn't long enough
	MissingParamPass = gp.APIerror{Reason: "Missing parameter: pass"}
	//MissingParamEmail = your email wasn't long enough
	MissingParamEmail = gp.APIerror{Reason: "Missing parameter: email"}
	//InvalidEmail = Your email isn't in our approved list
	InvalidEmail = gp.APIerror{Reason: "Invalid Email"}
	//UserAlreadyExists appens when creating an account with a dupe email address.
	UserAlreadyExists = gp.APIerror{Reason: "Username or email address already taken"}
	//NoSuchUser happens when you do an action which specifies a non-existent user.
	NoSuchUser = gp.APIerror{Reason: "That user does not exist."}
)

//Authenticator handles user authentication.
type Authenticator struct {
	sc   *psc.StatementCache
	pool *redis.Pool
}

//tokenCached returns true if this id:token pair exists.
func (auth *Authenticator) tokenCached(id gp.UserID, token string) bool {
	conn := auth.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:token:%s", id, token)
	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false
	}
	return exists
}

//createToken generates a new gp.Token which expires in 24h. If something goes wrong,
//it issues a token which expires now
func createToken(userID gp.UserID) gp.Token {
	random, err := randomString()
	if err != nil {
		log.Println(err)
		return gp.Token{UserID: userID, Token: "foo", Expiry: time.Now().UTC()}
	}
	expiry := time.Now().AddDate(1, 0, 0).UTC().Round(time.Second)
	token := gp.Token{UserID: userID, Token: random, Expiry: expiry}
	return token
}

//ValidateToken returns true if this id:token pair is valid and false otherwise (or if there's a db error).
func (auth *Authenticator) ValidateToken(id gp.UserID, token string) bool {
	//If the api.db is down, this will fail for everyone who doesn't have a api.cached
	//token, and so no new requests will be sent.
	//I'm calling that a "feature" for now.
	if auth.tokenCached(id, token) {
		return true
	}
	return auth.tokenExists(id, token)
}

//TokenExists returns true if this user:token pair exists, false otherwise (or in the case of error)
func (auth *Authenticator) tokenExists(id gp.UserID, token string) bool {
	var expiry string
	s, err := auth.sc.Prepare("SELECT expiry FROM tokens WHERE user_id = ? AND token = ?")
	if err != nil {
		return false
	}
	err = s.QueryRow(id, token).Scan(&expiry)
	if err != nil {
		return false
	}
	t, _ := time.Parse(mysqlTime, expiry)
	if t.After(time.Now()) {
		return true
	}
	return false
}

//ValidatePass returns the id of the user with this email:pass pair, or err if the comparison is not valid.
func (auth *Authenticator) validatePass(email string, pass string) (id gp.UserID, err error) {
	passBytes := []byte(pass)
	hash, id, err := auth.getHash(email)
	if err != nil {
		return 0, err
	}
	err = bcrypt.CompareHashAndPassword(hash, passBytes)
	if err != nil {
		return 0, err
	}
	return id, nil
}

//GetHash returns this user's password hash (by username).
func (auth *Authenticator) getHash(user string) (hash []byte, id gp.UserID, err error) {
	s, err := auth.sc.Prepare("SELECT id, password FROM users WHERE email = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(user).Scan(&id, &hash)
	return
}

//CreateAndStoreToken issues an access token for this user.
func (auth *Authenticator) createAndStoreToken(id gp.UserID) (gp.Token, error) {
	token := createToken(id)
	err := auth.addToken(token)
	go auth.cacheToken(token)
	if err != nil {
		return token, err
	}
	return token, nil
}

//cacheToken records this token in the cache until it expires.
func (auth *Authenticator) cacheToken(token gp.Token) {
	conn := auth.pool.Get()
	defer conn.Close()
	expiry := int(token.Expiry.Sub(time.Now()).Seconds())
	key := fmt.Sprintf("users:%d:token:%s", token.UserID, token.Token)
	conn.Send("SETEX", key, expiry, token.Expiry)
	conn.Flush()
}

//AddToken records this session token in the database.
func (auth *Authenticator) addToken(token gp.Token) (err error) {
	s, err := auth.sc.Prepare("INSERT INTO tokens (user_id, token, expiry) VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(token.UserID, token.Token, token.Expiry)
	return
}

//AttemptLogin will (a) return BadLogin if your email:pass combination isn't correct; (b) return a non-nil verification status (if your account is not yet verified) and (c) if neither of the above, issue you a session token.
func (api *API) AttemptLogin(email, pass string) (token gp.Token, verification gp.Status, err error) {
	id, err := api.Auth.validatePass(email, pass)
	if err != nil {
		err = BadLogin
		return
	}
	verified, err := api.isVerified(id)
	if err != nil {
		return
	}
	if !verified {
		verification = gp.NewStatus("unverified", email)
		return
	}
	token, err = api.Auth.createAndStoreToken(id)
	return
}

//AttemptRegister tries to register this user.
func (api *API) AttemptRegister(email, pass, first, last, invite string) (created gp.NewUser, err error) {
	switch {
	case len(first) < 2:
		err = MissingParamFirst
		return
	case len(last) < 1:
		err = MissingParamLast
		return
	case len(pass) == 0:
		err = MissingParamPass
		return
	case len(email) == 0:
		err = MissingParamEmail
		return
	}
	validates, err := api.validateEmail(email)
	if err != nil {
		return
	}
	if !validates {
		err = InvalidEmail
		return
	}
	return api.registerUser(pass, email, first, last, invite)
}

//ValidateEmail returns true if this email (a) looks vaguely well-formed and (b) belongs to a domain who is allowed to sign up.
func (api *API) validateEmail(email string) (validates bool, err error) {
	if !looksLikeEmail(email) {
		return false, nil
	}
	rules, err := api.getRules()
	if err != nil {
		return false, err
	}
	return api.testEmail(email, rules), nil
}

func (api *API) testEmail(email string, rules []gp.Rule) bool {
	for _, rule := range rules {
		if rule.Type == "email" && strings.HasSuffix(email, rule.Value) {
			return true
		}
	}
	return false
}

//RegisterUser accepts a password, email address, firstname and lastname. It will return an error if email isn't unique, or if pass is too short.
//If the optional "invite" is set and corresponds to email, it will skip the verification step.
func (api *API) registerUser(pass, email, first, last, invite string) (newUser gp.NewUser, err error) {
	email = normalizeEmail(email)
	userID, err := api.createUser(first, last, pass, email)
	if err != nil {
		return
	}
	_, err = api.assignNetworks(userID, email)
	if err != nil {
		return
	}
	exists, err := api.inviteExists(email, invite)
	newUser.ID = userID
	newUser.Status = "unverified"
	if err == nil && exists {
		err = api.verify(userID)
		if err != nil {
			return
		}
		newUser.Status = "verified"
		err = api.acceptAllInvites(userID, email)
	} else {
		err = api.generateAndSendVerification(userID, first, email)
	}
	go api.setUserType(userID)
	return
}

func (api *API) createUser(first, last string, pass string, email string) (userID gp.UserID, err error) {
	err = checkPassStrength(pass)
	if err != nil {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		return 0, err
	}
	userID, err = api._registerUser(first, last, hash, email)
	if err != nil {
		return 0, err
	}
	api.esIndexUser(userID)
	return
}

//AttemptResendVerification tries to send a new verification email to this address, or returns NoSuchUser if that email isn't one we know about. NB: this allows account enumeration, I guess...
func (api *API) AttemptResendVerification(email string) error {
	userID, err := api.userWithEmail(email)
	switch {
	case err != nil: //No user with this email
		fbid, err := api.fBUserWithEmail(email)
		if err == nil {
			api.FBissueVerification(fbid)
			return nil
		}
		if err == NoSuchUser {
			err = NoSuchUser
		}
		return err
	default:
		user, err := api.users.byID(userID)
		if err != nil {
			return err
		}
		api.generateAndSendVerification(userID, user.Name, email)
		return nil
	}
}

//GenerateAndSendVerification generates a random string and sends it embedded in a link to the user.
//It's probably safe to give it user input -- \r\n is stripped out.
func (api *API) generateAndSendVerification(userID gp.UserID, user string, email string) (err error) {
	random, err := randomString()
	if err != nil {
		return
	}
	err = api.setVerificationToken(userID, random)
	if err != nil {
		return
	}
	user = strings.Replace(user, "\r", "", -1)
	user = strings.Replace(user, "\n", "", -1)
	err = api.issueVerificationEmail(email, user, random)
	return
}
func normalizeEmail(email string) string {
	splitOnPlus := strings.Split(email, "+")
	if len(splitOnPlus) > 1 {
		splitOnAt := strings.Split(email, "@")
		if len(splitOnAt) > 1 {
			return splitOnPlus[0] + "@" + splitOnAt[1]
		}
		//Shouldn't happen if used in conjunction with looksLikeEmail
		return email
	}
	return email
}

func looksLikeEmail(email string) bool {
	rx := "<?\\S+@\\S+?>?"
	regex, _ := regexp.Compile(rx)
	if !regex.MatchString(email) {
		return (false)
	}
	return (true)
}

func checkPassStrength(pass string) (err error) {
	if len(pass) < 5 {
		return ETOOWEAK
	}
	return nil
}

func (api *API) verificationURL(token string) (url string) {
	if api.Config.DevelopmentMode {
		url = "https://dev.gleepost.com/verification.html?token=" + token
		return
	}
	url = "https://gleepost.com/verification.html?token=" + token
	return
}

func (api *API) appVerificationURL(token string) (url string) {
	return "gleepost://verify/" + token
}

func (api *API) recoveryURL(id gp.UserID, token string) (url string) {
	if api.Config.DevelopmentMode {
		url = fmt.Sprintf("https://dev.gleepost.com/reset_password.html?user-id=%d&t=%s", id, token)
		return
	}
	url = fmt.Sprintf("https://gleepost.com/reset_password.html?user-id=%d&t=%s", id, token)
	return
}

//TODO: send an actual link
func (api *API) issueVerificationEmail(email string, name string, token string) (err error) {
	url := api.verificationURL(token)
	html := "<html><body><a href=\"" + url + "\">Verify your account online here.</a></body></html>"
	err = api.Mail.SendHTML(email, name+", verify your Gleepost account!", html)
	return
}

func (api *API) issueRecoveryEmail(email string, user gp.User, token string) (err error) {
	url := api.recoveryURL(user.ID, token)
	html := "<html><body><a href=\"" + url + "\">Click here to recover your password.</a></body></html>"
	err = api.Mail.SendHTML(email, user.Name+", recover your Gleepost password!", html)
	return
}

//Verify will verify an account associated with a given verification token, or return an error if no such token exists.
//Additionally, if the token has been issued as part of the facebook login process, Verify will first attempt to match the verified email with an existing gleepost account, and verify that, linking the gleepost account to the facebook id.
//If no such account exists, Verify will create a new gleepost account for that facebook user and verify it.
//In addition, Verify adds the user to any networks they've been invited to.
func (api *API) Verify(token string) (err error) {
	id, err := api.verificationTokenExists(token)
	if err == nil {
		err = api.verify(id)
		if err == nil {
			var email string
			email, err = api.getEmail(id)
			if err != nil {
				log.Println("Error getting user email:", err)
				return
			}
			err = api.acceptAllInvites(id, email)
		}
		if err != nil {
			log.Println("Error with verification/accepting invites:", err)
		}
		return
	}
	fbid, err := api.fBVerificationExists(token)
	if err != nil {
		if err != NoSuchVerificationToken {
			log.Println("Error verifying (facebook)", err)
		}
		return
	}
	email, err := api.fBGetEmail(fbid)
	if err != nil {
		log.Println("Couldn't get this facebook account's email:", err)
		return
	}
	userID, err := api.userWithEmail(email)
	if err != nil {
		log.Println("There isn't a user with this facebook email")
		userID, err = api.createUserFromFB(fbid, email)
		if err != nil {
			return
		}
	}
	err = api.userSetFB(userID, fbid)
	if err == nil {
		err = api.verify(userID)
		if err == nil {
			log.Println("Verifying worked. Now setting networks from invites...")
			err = api.acceptAllInvites(userID, email)
		}
	}
	return
}

//ChangePass updates a user's password, or gives a bcrypt error if the oldPass isn't valid.
func (api *API) ChangePass(userID gp.UserID, oldPass, newPass string) (err error) {
	passBytes := []byte(oldPass)
	hash, err := api.getHashByID(userID)
	if err != nil {
		return
	}
	err = bcrypt.CompareHashAndPassword(hash, passBytes)
	if err != nil {
		err = BadPassword
		return
	}
	err = checkPassStrength(newPass)
	if err != nil {
		return
	}
	hash, err = bcrypt.GenerateFromPassword([]byte(newPass), 10)
	if err != nil {
		return
	}
	err = api.passUpdate(userID, hash)
	return
}

//RequestReset sends a random reset token to this email address. If it doesn't correspond to an existing user, returns an error.
func (api *API) RequestReset(email string) (err error) {
	userID, err := api.userWithEmail(email)
	if err != nil {
		return
	}
	user, err := api.users.byID(userID)
	if err != nil {
		return
	}
	token, err := randomString()
	if err != nil {
		return
	}
	err = api.addPasswordRecovery(userID, token)
	if err != nil {
		return
	}
	err = api.issueRecoveryEmail(email, user, token)
	return
}

//ResetPass takes a reset token and a password and (if the reset token is valid) updates the password.
func (api *API) ResetPass(userID gp.UserID, token string, newPass string) (err error) {
	exists, err := api.checkPasswordRecovery(userID, token)
	if err != nil {
		return
	}
	if !exists {
		err = EBADREC
		return
	}
	err = checkPassStrength(newPass)
	if err != nil {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), 10)
	if err != nil {
		return
	}
	err = api.passUpdate(userID, hash)
	if err != nil {
		return
	}
	err = api.deletePasswordRecovery(userID, token)
	return
}

//GetHashByID returns this user's password hash.
func (api *API) getHashByID(id gp.UserID) (hash []byte, err error) {
	s, err := api.sc.Prepare("SELECT password FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&hash)
	return
}

//PassUpdate replaces this user's password hash with a new one.
func (api *API) passUpdate(id gp.UserID, newHash []byte) (err error) {
	s, err := api.sc.Prepare("UPDATE users SET password = ? WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(newHash, id)
	return
}

//SetVerificationToken records a (hopefully random!) verification token for this user.
func (api *API) setVerificationToken(id gp.UserID, token string) (err error) {
	s, err := api.sc.Prepare("REPLACE INTO `verification` (user_id, token) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(id, token)
	return
}

//VerificationTokenExists returns the user who this verification token belongs to, or an error if there isn't one.
func (api *API) verificationTokenExists(token string) (id gp.UserID, err error) {
	s, err := api.sc.Prepare("SELECT user_id FROM verification WHERE token = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(token).Scan(&id)
	return
}

//Verify marks a user as verified.
func (api *API) verify(id gp.UserID) (err error) {
	s, err := api.sc.Prepare("UPDATE users SET verified = 1 WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(id)
	return
}

//IsVerified returns true if this user is verified.
func (api *API) isVerified(user gp.UserID) (verified bool, err error) {
	s, err := api.sc.Prepare("SELECT verified FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(user).Scan(&verified)
	return
}

//AddPasswordRecovery records a password recovery token for this user.
func (api *API) addPasswordRecovery(userID gp.UserID, token string) (err error) {
	s, err := api.sc.Prepare("REPLACE INTO password_recovery (token, user) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(token, userID)
	return
}

//CheckPasswordRecovery returns true if this password recovery user:token pair exists.
func (api *API) checkPasswordRecovery(userID gp.UserID, token string) (exists bool, err error) {
	s, err := api.sc.Prepare("SELECT count(*) FROM password_recovery WHERE user = ? and token = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(userID, token).Scan(&exists)
	return
}

//DeletePasswordRecovery removes this password recovery token so it can't be used again.
func (api *API) deletePasswordRecovery(userID gp.UserID, token string) (err error) {
	s, err := api.sc.Prepare("DELETE FROM password_recovery WHERE user = ? and token = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(userID, token)
	return
}
