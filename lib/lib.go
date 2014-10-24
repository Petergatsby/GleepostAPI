package lib

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	"code.google.com/p/go.crypto/bcrypt"
	"github.com/draaglom/GleepostAPI/lib/cache"
	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/db"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/mail"
	"github.com/draaglom/GleepostAPI/lib/push"
	"github.com/peterbourgon/g2s"
)

//API contains all the configuration and sub-modules the Gleepost API requires to function.
type API struct {
	cache  *cache.Cache
	db     *db.DB
	fb     *FB
	mail   *mail.Mailer
	Config conf.Config
	push   *push.Pusher
	statsd g2s.Statter
}

//New creates an API from a gp.Config
func New(conf conf.Config) (api *API) {
	api = new(API)
	api.cache = cache.New(conf.Redis)
	api.db = db.New(conf.Mysql)
	api.Config = conf
	api.fb = &FB{config: conf.Facebook}
	api.mail = mail.New(conf.Email)
	api.push = push.New(conf)
	statsd, err := g2s.Dial("udp", api.Config.Statsd)
	api.statsd = statsd
	if err != nil {
		log.Println(err)
	}
	go api.process(transcodeQueue)
	return
}

//Time reports the time for this stat to statsd. (use it with defer)
func (api *API) Time(start time.Time, bucket string) {
	duration := time.Since(start)
	var ns string
	if api.Config.DevelopmentMode {
		ns = "dev."
	} else {
		ns = "prod."
	}
	bucket = ns + bucket
	api.statsd.Timing(1.0, bucket, duration)
}

//You'll get this when your password is too week (ie, less than 5 chars at the moment)
var ETOOWEAK = gp.APIerror{Reason: "Password too weak!"}

//EBADREC means you tried to recover your password with an invalid or missing password reset token.
var EBADREC = gp.APIerror{Reason: "Bad password recovery token."}

const inviteCampaignIOS = "http://ad.apps.fm/2sQSPmGhIyIaKGZ01wtHD_E7og6fuV2oOMeOQdRqrE1xKZaHtwHb8iGWO0i4C3przjNn5v5h3werrSfj3HdREnrOdTW3xhZTjoAE5juerBQ8UiWF6mcRlxGSVB6OqmJv"
const inviteCampaignAndroid = "http://ad.apps.fm/WOIqfW3iWi3krjT_Y-U5uq5px440Px0vtrw1ww5B54zsDQMwj9gVfW3tCxpkeXdizYtt678Ci7Y3djqLAxIATdBAW28aYabvxh6AeQ1YLF8"

/********************************************************************
Top-level functions
********************************************************************/

//RandomString generates a long, random string (currently hex encoded, for some unknown reason.)
//TODO: base64 url-encode instead.
func RandomString() (random string, err error) {
	hash := sha256.New()
	randombuf := make([]byte, 32) //Number pulled out of my... ahem.
	_, err = io.ReadFull(rand.Reader, randombuf)
	if err != nil {
		return
	}
	hash.Write(randombuf)
	random = hex.EncodeToString(hash.Sum(nil))
	return
}

//createToken generates a new gp.Token which expires in 24h. If something goes wrong,
//it issues a token which expires now
//createtoken might do with returning an error
//why would it break though
func createToken(userID gp.UserID) gp.Token {
	random, err := RandomString()
	if err != nil {
		return (gp.Token{UserID: userID, Token: "foo", Expiry: time.Now().UTC()})
	}
	expiry := time.Now().AddDate(1, 0, 0).UTC().Round(time.Second)
	token := gp.Token{UserID: userID, Token: random, Expiry: expiry}
	return (token)
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

//ValidateToken returns true if this id:token pair is valid (or if LoginOverride) and false otherwise (or if there's a db error).
func (api *API) ValidateToken(id gp.UserID, token string) bool {
	//If the api.db is down, this will fail for everyone who doesn't have a api.cached
	//token, and so no new requests will be sent.
	//I'm calling that a "feature" for now.
	if api.Config.LoginOverride {
		return (true)
	} else if api.cache.TokenExists(id, token) {
		return (true)
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
func (api *API) CreateAndStoreToken(id gp.UserID) (gp.Token, error) {
	token := createToken(id)
	err := api.db.AddToken(token)
	api.cache.PutToken(token)
	if err != nil {
		return token, err
	}
	return token, nil
}

//GetUser returns the User with this ID. It hits the cache first, so some details may be out of date.
func (api *API) GetUser(id gp.UserID) (user gp.User, err error) {
	/* Hits the api.cache then the api.db
	only I'm not 100% confident yet with what
	happens when you attempt to get a redis key
	that doesn't exist in redigo! */
	user, err = api.cache.GetUser(id)
	if err != nil {
		user, err = api.db.GetUser(id)
		if err == nil {
			api.cache.SetUser(user)
		}
	}
	return
}

//UserGetProfile returns the Profile (extended info) for the user with this ID.
func (api *API) UserGetProfile(userID, otherID gp.UserID) (user gp.Profile, err error) {
	if userID == otherID {
		return api.getProfile(userID, otherID)
	}
	shared, e := api.HaveSharedNetwork(userID, otherID)
	switch {
	case e != nil:
		fallthrough
	case !shared:
		err = &ENOTALLOWED
	default:
		return api.getProfile(userID, otherID)
	}
	return
}

//SubjectiveRSVPCount shows the number of events otherID has attended, from the perspective of the `perspective` user (ie, not counting those events perspective can't see...)
func (api *API) SubjectiveRSVPCount(perspective gp.UserID, otherID gp.UserID) (count int, err error) {
	return api.db.SubjectiveRSVPCount(perspective, otherID)
}

//getProfile returns the Profile (extended info) for the user with this ID.
func (api *API) getProfile(perspective, otherID gp.UserID) (user gp.Profile, err error) {
	user, err = api.db.GetProfile(otherID)
	if err != nil {
		return
	}
	nets, err := api.GetUserNetworks(user.ID)
	if err != nil {
		return
	}
	rsvps, err := api.SubjectiveRSVPCount(perspective, otherID)
	if err != nil {
		return
	}
	user.RSVPCount = rsvps
	groupCount, err := api.db.SubjectiveMembershipCount(perspective, otherID)
	if err != nil {
		return
	}
	user.GroupCount = groupCount
	postCount, err := api.db.UserPostCount(perspective, otherID)
	if err != nil {
		return
	}
	user.PostCount = postCount
	user.Network = nets[0]
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

//RegisterUser accepts a username, password, email address, firstname and lastname. It will return an error if user or email aren't unique, or if pass is too short.
//If the optional "invite" is set and corresponds to email, it will skip the verification step.
func (api *API) RegisterUser(user, pass, email, first, last, invite string) (newUser gp.NewUser, err error) {
	email = normalizeEmail(email)
	userID, err := api.createUser(user, pass, email)
	if err != nil {
		return
	}
	_, err = api.assignNetworks(userID, email)
	if err != nil {
		return
	}
	err = api.SetUserName(userID, first, last)
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

func (api *API) createUser(user string, pass string, email string) (userID gp.UserID, err error) {
	err = checkPassStrength(pass)
	if err != nil {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		return 0, err
	}
	userID, err = api.db.RegisterUser(user, hash, email)
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

//GetContacts returns all contacts (incl. those who have not yet accepted) for this user.
func (api *API) GetContacts(user gp.UserID) (contacts []gp.Contact, err error) {
	return api.db.GetContacts(user)
}

//AreContacts returns true if a and b are (confirmed) contacts.
//TODO: Implement a proper db-level version
func (api *API) AreContacts(a, b gp.UserID) (areContacts bool, err error) {
	contacts, err := api.GetContacts(a)
	if err != nil {
		return
	}
	for _, c := range contacts {
		if c.ID == b && c.YouConfirmed && c.TheyConfirmed {
			return true, nil
		}
	}
	return false, nil
}

//UserHasPosted returns true if user has ever created a post from the perspective of perspective.
//TODO: Implement a direct version
func (api *API) UserHasPosted(user gp.UserID, perspective gp.UserID) (posted bool, err error) {
	posts, err := api.GetUserPosts(user, perspective, gp.OSTART, 0, 1, "")
	if err != nil {
		return
	}
	if len(posts) > 0 {
		return true, nil
	}
	return false, nil
}

//AddContact sends a contact request from adder to addee.
func (api *API) AddContact(adder gp.UserID, addee gp.UserID) (contact gp.Contact, err error) {
	user, err := api.GetUser(addee)
	if err != nil {
		return
	}
	exists, err := api.ContactRequestExists(addee, adder)
	if err != nil {
		return
	}
	if exists {
		return api.AcceptContact(adder, addee)
	}
	err = api.db.AddContact(adder, addee)
	if err == nil {
		go api.createNotification("added_you", adder, addee, 0)
	}
	contact.User = user
	contact.YouConfirmed = true
	contact.TheyConfirmed = false
	return
}

//ContactRequestExists returns true if adder has previously added addee (whether they have accepted or not).
func (api *API) ContactRequestExists(adder gp.UserID, addee gp.UserID) (exists bool, err error) {
	return api.db.ContactRequestExists(adder, addee)
}

//AcceptContact marks this request as accepted - these users are now contacts.
func (api *API) AcceptContact(user gp.UserID, toAccept gp.UserID) (contact gp.Contact, err error) {
	err = api.db.UpdateContact(user, toAccept)
	if err != nil {
		return
	}
	contact.User, err = api.GetUser(toAccept)
	if err != nil {
		return
	}
	contact.YouConfirmed = true
	contact.TheyConfirmed = true
	go api.createNotification("accepted_you", user, toAccept, 0)
	go api.UnExpireBetween([]gp.UserID{user, toAccept})
	return
}

//SetProfileImage updates this user's profile image to the new url
func (api *API) SetProfileImage(id gp.UserID, url string) (err error) {
	err = api.db.SetProfileImage(id, url)
	if err == nil {
		go api.cache.SetProfileImage(id, url)
	}
	return
}

//SetBusyStatus records whether you are busy or not.
func (api *API) SetBusyStatus(id gp.UserID, busy bool) (err error) {
	err = api.db.SetBusyStatus(id, busy)
	if err == nil {
		go api.cache.SetBusyStatus(id, busy)
	}
	return
}

//BusyStatus returns true if this user is busy.
func (api *API) BusyStatus(id gp.UserID) (busy bool, err error) {
	busy, err = api.db.BusyStatus(id)
	return
}

func (api *API) userPing(id gp.UserID) {
	api.cache.UserPing(id, api.Config.OnlineTimeout)
}

func (api *API) userIsOnline(id gp.UserID) bool {
	return api.cache.UserIsOnline(id)
}

//GetUserNotifications returns all unseen notifications for this user, and the seen ones as well if includeSeen is true.
func (api *API) GetUserNotifications(id gp.UserID, includeSeen bool) (notifications []interface{}, err error) {
	return api.db.GetUserNotifications(id, includeSeen)
}

//MarkNotificationsSeen marks all notifications up to upTo seen for this user.
func (api *API) MarkNotificationsSeen(id gp.UserID, upTo gp.NotificationID) (err error) {
	return api.db.MarkNotificationsSeen(id, upTo)
}

//createNotification creates a new gleepost notification. location is the id of the object where the notification happened - a post id if the notification is "liked" or "commented", or a network id if the notification type is "added_group". Otherwise, the location will be ignored.
func (api *API) createNotification(ntype string, by gp.UserID, recipient gp.UserID, location uint64) (err error) {
	notification, err := api.db.CreateNotification(ntype, by, recipient, location)
	if err == nil {
		api.Push(notification, recipient)
		go api.cache.PublishEvent("notification", "/notifications", notification, []string{NotificationChannelKey(recipient)})
	}
	return
}

//NotificationChannelKey returns the channel used for this user's notifications.
func NotificationChannelKey(id gp.UserID) (channel string) {
	return fmt.Sprintf("n:%d", id)
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

func (api *API) inviteURL(token, email string) string {
	if api.Config.DevelopmentMode {
		return fmt.Sprintf("https://dev.gleepost.com/?invite=%s&email=%s", token, email)
	}
	return fmt.Sprintf("https://gleepost.com/?invite=%s&email=%s", token, email)
}

func (api *API) issueInviteEmail(email string, from gp.User, group gp.Group, token string) (err error) {
	url := api.inviteURL(token, email)
	subject := fmt.Sprintf("%s has invited you to the private group \"%s\" on Gleepost.", from.Name, group.Name)
	html := "<html><body>" +
		"Don't miss out on their events - <a href=" + url + ">Click here to accept the invitation.</a><br>" +
		"On your phone? <a href=\"" + inviteCampaignIOS + "\">install the app on your iPhone here</a>" +
		" or <a href=\"" + inviteCampaignAndroid + "\">click here to get the Android app.</a>" +
		"</body></html>"
	err = api.mail.SendHTML(email, subject, html)
	return
}

//GetEmail returns this user's email address.
func (api *API) GetEmail(id gp.UserID) (email string, err error) {
	return api.db.GetEmail(id)
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

//SetUserName updates this user's name.
func (api *API) SetUserName(id gp.UserID, firstName, lastName string) (err error) {
	return api.db.SetUserName(id, firstName, lastName)
}

//UserChangeTagline sets this user's tagline (obviously enough...)
func (api *API) UserChangeTagline(userID gp.UserID, tagline string) (err error) {
	return api.db.UserChangeTagline(userID, tagline)
}

//UserWithEmail returns the userID this email is associated with, or err if there isn't one.
func (api *API) UserWithEmail(email string) (id gp.UserID, err error) {
	return api.db.UserWithEmail(email)
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

//GetLiveConversations returns all the live conversations (there should only be 3 or less) for this user.
//(A live conversation is one which has not ended and has an expiry in the future)
func (api *API) GetLiveConversations(userID gp.UserID) (conversations []gp.ConversationSmall, err error) {
	return api.db.GetLiveConversations(userID)
}

//DeviceFeedback is called in response to APNS feedback; it records that a device token was no longer valid at this time and deletes it if it hasn't been re-registered since.
func (api *API) DeviceFeedback(deviceID string, timestamp uint32) (err error) {
	t := time.Unix(int64(timestamp), 0)
	return api.db.Feedback(deviceID, t)
}

//IsAdmin returns true if tis user is a member of the Admin network specified in the config.
func (api *API) IsAdmin(user gp.UserID) (admin bool) {
	in, err := api.UserInNetwork(user, gp.NetworkID(api.Config.Admins))
	if err == nil && in {
		return true
	}
	return false
}

//CreateUserSpecial manually creates a user with these details, bypassing validation etc
func (api *API) CreateUserSpecial(first, last, email, pass string, verified bool, primaryNetwork gp.NetworkID) (userID gp.UserID, err error) {
	user, err := RandomString()
	if err != nil {
		return
	}
	userID, err = api.createUser(user, pass, email)
	if err != nil {
		return
	}
	err = api.SetUserName(userID, first, last)
	if err != nil {
		return
	}
	if verified {
		err = api.db.Verify(userID)
		if err != nil {
			return
		}
	}
	err = api.setNetwork(userID, primaryNetwork)
	return
}
