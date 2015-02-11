package lib

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"code.google.com/p/go.crypto/bcrypt"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

//BadLogin = guess...
var BadLogin = gp.APIerror{Reason: "Bad username/password"}

//createToken generates a new gp.Token which expires in 24h. If something goes wrong,
//it issues a token which expires now
func createToken(userID gp.UserID) gp.Token {
	random, err := RandomString()
	if err != nil {
		log.Println(err)
		return (gp.Token{UserID: userID, Token: "foo", Expiry: time.Now().UTC()})
	}
	expiry := time.Now().AddDate(1, 0, 0).UTC().Round(time.Second)
	token := gp.Token{UserID: userID, Token: random, Expiry: expiry}
	return (token)
}

//ValidateToken returns true if this id:token pair is valid (or if LoginOverride) and false otherwise (or if there's a db error).
func (api *API) ValidateToken(id gp.UserID, token string) bool {
	//If the api.db is down, this will fail for everyone who doesn't have a api.cached
	//token, and so no new requests will be sent.
	//I'm calling that a "feature" for now.
	if api.Config.LoginOverride {
		return true
	} else if api.cache.TokenExists(id, token) {
		return true
	} else {
		return api.db.TokenExists(id, token)
	}
}

//ValidatePass returns the id of the user with this email:pass pair, or err if the comparison is not valid.
func (api *API) ValidatePass(email string, pass string) (id gp.UserID, err error) {
	passBytes := []byte(pass)
	hash, id, err := api.db.GetHash(email)
	if err != nil {
		return 0, err
	}
	err = bcrypt.CompareHashAndPassword(hash, passBytes)
	if err != nil {
		return 0, err
	}
	return id, nil
}

//CreateAndStoreToken issues an access token for this user.
func (api *API) createAndStoreToken(id gp.UserID) (gp.Token, error) {
	token := createToken(id)
	err := api.db.AddToken(token)
	api.cache.PutToken(token)
	if err != nil {
		return token, err
	}
	return token, nil
}

//AttemptLogin will (a) return BadLogin if your email:pass combination isn't correct; (b) return a non-nil verification status (if your account is not yet verified) and (c) if neither of the above, issue you a session token.
func (api *API) AttemptLogin(email, pass string) (token gp.Token, verification gp.Status, err error) {
	id, err := api.ValidatePass(email, pass)
	if err != nil {
		err = BadLogin
		return
	}
	verified, err := api.IsVerified(id)
	if err != nil {
		return
	}
	if !verified {
		verification = gp.NewStatus("unverified", email)
		return
	}
	token, err = api.createAndStoreToken(id)
	return
}

//ValidateEmail returns true if this email (a) looks vaguely well-formed and (b) belongs to a domain who is allowed to sign up.
func (api *API) ValidateEmail(email string) (validates bool, err error) {
	if !looksLikeEmail(email) {
		return false, nil
	}
	rules, err := api.db.GetRules()
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
func (api *API) RegisterUser(pass, email, first, last, invite string) (newUser gp.NewUser, err error) {
	email = normalizeEmail(email)
	userID, err := api.createUser(first, last, pass, email)
	if err != nil {
		return
	}
	_, err = api.assignNetworks(userID, email)
	if err != nil {
		return
	}
	exists, err := api.InviteExists(email, invite)
	log.Println(exists, err)
	newUser.ID = userID
	newUser.Status = "unverified"
	if err == nil && exists {
		err = api.db.Verify(userID)
		if err != nil {
			return
		}
		newUser.Status = "verified"
		err = api.AssignNetworksFromInvites(userID, email)
		if err != nil {
			return
		}
		err = api.AcceptAllInvites(email)
	} else {
		err = api.GenerateAndSendVerification(userID, first, email)
	}
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
	userID, err = api.db.RegisterUser(first, last, hash, email)
	if err != nil {
		return 0, err
	}
	return
}

//GenerateAndSendVerification generates a random string and sends it embedded in a link to the user.
//It's probably safe to give it user input -- \r\n is stripped out.
func (api *API) GenerateAndSendVerification(userID gp.UserID, user string, email string) (err error) {
	random, err := RandomString()
	if err != nil {
		return
	}
	err = api.db.SetVerificationToken(userID, random)
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
		return &ETOOWEAK
	}
	return nil
}
func (api *API) verificationURL(token string) (url string) {
	if api.Config.DevelopmentMode {
		url = "https://dev.gleepost.com/verification.html?token=" + token
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
	}
	url = fmt.Sprintf("https://gleepost.com/reset_password.html?user-id=%d&t=%s", id, token)
	return
}

//TODO: send an actual link
func (api *API) issueVerificationEmail(email string, name string, token string) (err error) {
	url := api.verificationURL(token)
	html := "<html><body><a href=\"" + url + "\">Verify your account online here.</a></body></html>"
	err = api.mail.SendHTML(email, name+", verify your Gleepost account!", html)
	return
}

func (api *API) issueRecoveryEmail(email string, user gp.User, token string) (err error) {
	url := api.recoveryURL(user.ID, token)
	html := "<html><body><a href=\"" + url + "\">Click here to recover your password.</a></body></html>"
	err = api.mail.SendHTML(email, user.Name+", recover your Gleepost password!", html)
	return
}

//Verify will verify an account associated with a given verification token, or return an error if no such token exists.
//Additionally, if the token has been issued as part of the facebook login process, Verify will first attempt to match the verified email with an existing gleepost account, and verify that, linking the gleepost account to the facebook id.
//If no such account exists, Verify will create a new gleepost account for that facebook user and verify it.
//In addition, Verify adds the user to any networks they've been invited to.
func (api *API) Verify(token string) (err error) {
	id, err := api.db.VerificationTokenExists(token)
	if err == nil {
		log.Println("Verification token exists (normal-mode)")
		err = api.db.Verify(id)
		if err == nil {
			log.Println("User has verified successfully")
			var email string
			email, err = api.GetEmail(id)
			if err != nil {
				return
			}
			err = api.AssignNetworksFromInvites(id, email)
			if err != nil {
				log.Println("Something went wrong with assigning to invited networks:", err)
				return
			}
			err = api.AcceptAllInvites(email)
		}
		return
	}
	fbid, err := api.FBVerify(token)
	if err != nil {
		log.Println("Error verifying (facebook)", err)
		return
	}
	email, err := api.FBGetEmail(fbid)
	if err != nil {
		log.Println("Couldn't get this facebook account's email:", err)
		return
	}
	userID, err := api.UserWithEmail(email)
	if err != nil {
		log.Println("There isn't a user with this facebook email")
		userID, err = api.CreateUserFromFB(fbid, email)
		if err != nil {
			return
		}
	}
	err = api.UserSetFB(userID, fbid)
	if err == nil {
		err = api.db.Verify(userID)
		if err == nil {
			log.Println("Verifying worked. Now setting networks from invites...")
			err = api.AssignNetworksFromInvites(userID, email)
			if err != nil {
				log.Println("Something went wrong while setting networks from invites:", err)
				return
			}
			err = api.AcceptAllInvites(email)
		}
	}
	return
}

//ChangePass updates a user's password, or gives a bcrypt error if the oldPass isn't valid.
func (api *API) ChangePass(userID gp.UserID, oldPass string, newPass string) (err error) {
	passBytes := []byte(oldPass)
	hash, err := api.db.GetHashByID(userID)
	if err != nil {
		return
	}
	err = bcrypt.CompareHashAndPassword(hash, passBytes)
	if err != nil {
		return
	}
	hash, err = bcrypt.GenerateFromPassword([]byte(newPass), 10)
	if err != nil {
		return
	}
	err = api.db.PassUpdate(userID, hash)
	return
}

//RequestReset sends a random reset token to this email address. If it doesn't correspond to an existing user, returns an error.
func (api *API) RequestReset(email string) (err error) {
	userID, err := api.UserWithEmail(email)
	if err != nil {
		return
	}
	user, err := api.GetUser(userID)
	if err != nil {
		return
	}
	token, err := RandomString()
	if err != nil {
		return
	}
	err = api.db.AddPasswordRecovery(userID, token)
	if err != nil {
		return
	}
	err = api.issueRecoveryEmail(email, user, token)
	return
}

//ResetPass takes a reset token and a password and (if the reset token is valid) updates the password.
func (api *API) ResetPass(userID gp.UserID, token string, newPass string) (err error) {
	exists, err := api.db.CheckPasswordRecovery(userID, token)
	if err != nil {
		return
	}
	if !exists {
		err = EBADREC
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), 10)
	if err != nil {
		return
	}
	err = api.db.PassUpdate(userID, hash)
	return
}

//IsVerified returns true if this user has verified their email, and probably err if this user doesn't exist?
func (api *API) IsVerified(userID gp.UserID) (verified bool, err error) {
	return api.db.IsVerified(userID)
}
