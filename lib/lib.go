package lib

import (
	"code.google.com/p/go.crypto/bcrypt"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/draaglom/GleepostAPI/lib/cache"
	"github.com/draaglom/GleepostAPI/lib/db"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/mail"
	"github.com/draaglom/GleepostAPI/lib/push"
	"io"
	"regexp"
	"strings"
	"time"
	"log"
)

type API struct {
	cache  *cache.Cache
	db     *db.DB
	fb     *FB
	mail   *mail.Mailer
	Config gp.Config
	push   *push.Pusher
}

func New(conf gp.Config) (api *API) {
	api = new(API)
	api.cache = cache.New(conf.Redis)
	api.db = db.New(conf.Mysql)
	api.Config = conf
	api.fb = &FB{config: conf.Facebook}
	api.mail = mail.New(conf.Email)
	api.push = push.New(conf)
	return
}

var ETOOWEAK = gp.APIerror{"Password too weak!"}
var EBADREC = gp.APIerror{"Bad password recovery token."}

const INVITE_CAMPAIGN_IOS = "http://ad.apps.fm/2sQSPmGhIyIaKGZ01wtHD_E7og6fuV2oOMeOQdRqrE1xKZaHtwHb8iGWO0i4C3przjNn5v5h3werrSfj3HdREnrOdTW3xhZTjoAE5juerBQ8UiWF6mcRlxGSVB6OqmJv"
const INVITE_CAMPAIGN_ANDROID = "http://ad.apps.fm/WOIqfW3iWi3krjT_Y-U5uq5px440Px0vtrw1ww5B54zsDQMwj9gVfW3tCxpkeXdizYtt678Ci7Y3djqLAxIATdBAW28aYabvxh6AeQ1YLF8"

/********************************************************************
Top-level functions
********************************************************************/

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
func createToken(userId gp.UserId) gp.Token {
	random, err := RandomString()
	if err != nil {
		return (gp.Token{userId, "foo", time.Now().UTC()})
	} else {
		expiry := time.Now().Add(time.Duration(168) * time.Hour).UTC().Round(time.Second)
		token := gp.Token{userId, random, expiry}
		return (token)
	}
}

func looksLikeEmail(email string) bool {
	rx := "<?\\S+@\\S+?>?"
	regex, _ := regexp.Compile(rx)
	if !regex.MatchString(email) {
		return (false)
	} else {
		return (true)
	}
}

func checkPassStrength(pass string) (err error) {
	if len(pass) < 5 {
		return &ETOOWEAK
	}
	return nil
}

func (api *API) ValidateToken(id gp.UserId, token string) bool {
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

func (api *API) ValidatePass(email string, pass string) (id gp.UserId, err error) {
	passBytes := []byte(pass)
	hash, id, err := api.db.GetHash(email)
	if err != nil {
		return 0, err
	} else {
		err := bcrypt.CompareHashAndPassword(hash, passBytes)
		if err != nil {
			return 0, err
		} else {
			return id, nil
		}
	}
}

func (api *API) CreateAndStoreToken(id gp.UserId) (gp.Token, error) {
	token := createToken(id)
	err := api.db.AddToken(token)
	api.cache.PutToken(token)
	if err != nil {
		return token, err
	} else {
		return token, nil
	}
}

func (api *API) GetUser(id gp.UserId) (user gp.User, err error) {
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

func (api *API) GetProfile(id gp.UserId) (user gp.Profile, err error) {
	user, err = api.db.GetProfile(id)
	if err != nil {
		return
	}
	nets, err := api.GetUserNetworks(user.Id)
	if err != nil {
		return
	}
	user.Network = nets[0]
	return
}

func (api *API) ValidateEmail(email string) (validates bool, err error) {
	if !looksLikeEmail(email) {
		return false, nil
	} else {
		rules, err := api.db.GetRules()
		if err != nil {
			return false, err
		}
		return api.testEmail(email, rules), nil
	}
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
	userId, err := api.createUser(user, pass, email)
	if err != nil {
		return
	}
	_, err = api.assignNetworks(userId, email)
	if err != nil {
		return
	}
	err = api.SetUserName(userId, first, last)
	if err != nil {
		return
	}
	exists, err := api.InviteExists(email, invite)
	log.Println(exists, err)
	newUser.Id = userId
	newUser.Status = "unverified"
	if err == nil && exists {
		err = api.db.Verify(userId)
		if err != nil {
			return
		}
		newUser.Status = "verified"
		err = api.AssignNetworksFromInvites(userId, email)
		if err != nil {
			return
		}
		err = api.AcceptAllInvites(email)
	} else {
		err = api.GenerateAndSendVerification(userId, first, email)
	}
	return
}

func (api *API) createUser(user string, pass string, email string) (userId gp.UserId, err error) {
	err = checkPassStrength(pass)
	if err != nil {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		return 0, err
	}
	userId, err = api.db.RegisterUser(user, hash, email)
	if err != nil {
		return 0, err
	}
	return
}

//TODO: this might end up using user input directly in an email. Sanitize!
func (api *API) GenerateAndSendVerification(userId gp.UserId, user string, email string) (err error) {
	random, err := RandomString()
	if err != nil {
		return
	}
	err = api.db.SetVerificationToken(userId, random)
	if err != nil {
		return
	}
	err = api.issueVerificationEmail(email, user, random)
	return
}

func (api *API) GetContacts(user gp.UserId) (contacts []gp.Contact, err error) {
	return api.db.GetContacts(user)
}

//AreContacts returns true if a and b are (confirmed) contacts.
//TODO: Implement a proper db-level version
func (api *API) AreContacts(a, b gp.UserId) (areContacts bool, err error) {
	contacts, err := api.GetContacts(a)
	if err != nil {
		return
	}
	for _, c := range contacts {
		if c.Id == b && c.YouConfirmed && c.TheyConfirmed {
			return true, nil
		}
	}
	return false, nil
}

//UserHasPosted returns true if user has ever created a post from the perspective of perspective.
//TODO: Implement a direct version
func (api *API) UserHasPosted(user gp.UserId, perspective gp.UserId) (posted bool, err error) {
	posts, err := api.GetUserPosts(user, perspective, gp.OSTART, 0, 1, "")
	if err != nil {
		return
	}
	if len(posts) > 0 {
		return true, nil
	}
	return false, nil
}

func (api *API) AddContact(adder gp.UserId, addee gp.UserId) (contact gp.Contact, err error) {
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

func (api *API) ContactRequestExists(adder gp.UserId, addee gp.UserId) (exists bool, err error) {
	return api.db.ContactRequestExists(adder, addee)
}

func (api *API) AcceptContact(user gp.UserId, toAccept gp.UserId) (contact gp.Contact, err error) {
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
	go api.UnExpireBetween([]gp.UserId{user, toAccept})
	return
}

func (api *API) AddDevice(user gp.UserId, deviceType string, deviceId string) (device gp.Device, err error) {
	err = api.db.AddDevice(user, deviceType, deviceId)
	if err != nil {
		return
	}
	device.User = user
	device.Type = deviceType
	device.Id = deviceId
	return
}

func (api *API) GetDevices(user gp.UserId) (devices []gp.Device, err error) {
	return api.db.GetDevices(user)
}

func (api *API) DeleteDevice(user gp.UserId, deviceId string) (err error) {
	return api.db.DeleteDevice(user, deviceId)
}

func (api *API) SetProfileImage(id gp.UserId, url string) (err error) {
	err = api.db.SetProfileImage(id, url)
	if err == nil {
		go api.cache.SetProfileImage(id, url)
	}
	return
}

func (api *API) SetBusyStatus(id gp.UserId, busy bool) (err error) {
	err = api.db.SetBusyStatus(id, busy)
	if err == nil {
		go api.cache.SetBusyStatus(id, busy)
	}
	return
}

func (api *API) BusyStatus(id gp.UserId) (busy bool, err error) {
	busy, err = api.db.BusyStatus(id)
	return
}

func (api *API) userPing(id gp.UserId) {
	api.cache.UserPing(id, api.Config.OnlineTimeout)
}

func (api *API) userIsOnline(id gp.UserId) bool {
	return api.cache.UserIsOnline(id)
}

func (api *API) GetUserNotifications(id gp.UserId) (notifications []interface{}, err error) {
	return api.db.GetUserNotifications(id)
}

func (api *API) MarkNotificationsSeen(id gp.UserId, upTo gp.NotificationId) (err error) {
	return api.db.MarkNotificationsSeen(id, upTo)
}

//createNotification creates a new gleepost notification. location is the id of the object where the notification happened - a post id if the notification is "liked" or "commented", or a network id if the notification type is "added_group". Otherwise, the location will be ignored.
func (api *API) createNotification(ntype string, by gp.UserId, recipient gp.UserId, location uint64) (err error) {
	notification, err := api.db.CreateNotification(ntype, by, recipient, location)
	if err == nil {
		api.Push(notification, recipient)
		go api.cache.PublishEvent("notification", "/notifications", notification, []string{NotificationChannelKey(recipient)})
	}
	return
}

func NotificationChannelKey(id gp.UserId) (channel string) {
	return fmt.Sprintf("n:%d", id)
}

func (api *API) verificationUrl(token string) (url string) {
	if api.Config.DevelopmentMode {
		url = "https://dev.gleepost.com/verification.html?token=" + token
	} else {
		url = "https://gleepost.com/verification.html?token=" + token
	}
	return
}

func (api *API) appVerificationUrl(token string) (url string) {
	return "gleepost://verify/" + token
}

func (api *API) recoveryUrl(id gp.UserId, token string) (url string) {
	if api.Config.DevelopmentMode {
		url = fmt.Sprintf("https://dev.gleepost.com/reset_password.html?user-id=%d&t=%s", id, token)
	} else {
		url = fmt.Sprintf("https://gleepost.com/reset_password.html?user-id=%d&t=%s", id, token)
	}
	return
}

//TODO: send an actual link
func (api *API) issueVerificationEmail(email string, name string, token string) (err error) {
	url := api.verificationUrl(token)
	html := "<html><body><a href=\"" + url + "\">Verify your account online here.</a></body></html>"
	err = api.mail.SendHTML(email, name+", verify your Gleepost account!", html)
	return
}

func (api *API) issueRecoveryEmail(email string, user gp.User, token string) (err error) {
	url := api.recoveryUrl(user.Id, token)
	html := "<html><body><a href=\"" + url + "\">Click here to recover your password.</a></body></html>"
	err = api.mail.SendHTML(email, user.Name+", recover your Gleepost password!", html)
	return
}

func (api *API) inviteUrl(token, email string) string {
	if api.Config.DevelopmentMode {
		return fmt.Sprintf("https://dev.gleepost.com/?invite=%s&email=%s", token, email)
	} else {
		return fmt.Sprintf("https://gleepost.com/?invite=%s&email=%s", token, email)
	}
}

func (api *API) issueInviteEmail(email string, from gp.User, group gp.Group, token string) (err error) {
	url := api.inviteUrl(token, email)
	subject := fmt.Sprintf("%s has invited you to the private group \"%s\" on Gleepost.", from.Name, group.Name)
	html := "<html><body>" +
	"Don't miss out on their events - <a href=" + url + ">Click here to accept the invitation.</a><br>" +
	"On your phone? <a href=\"" + INVITE_CAMPAIGN_IOS + "\">install the app on your iPhone here</a>" +
	" or <a href=\"" + INVITE_CAMPAIGN_ANDROID + "\">click here to get the Android app.</a>" +
	"</body></html>"
	err = api.mail.SendHTML(email, subject, html)
	return
}

func (api *API) GetEmail(id gp.UserId) (email string, err error) {
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
	userId, err := api.UserWithEmail(email)
	if err != nil {
		log.Println("There isn't a user with this facebook email")
		userId, err = api.CreateUserFromFB(fbid, email)
		if err != nil {
			return
		}
	}
	err = api.UserSetFB(userId, fbid)
	if err == nil {
		err = api.db.Verify(userId)
		if err == nil {
			log.Println("Verifying worked. Now setting networks from invites...")
			err = api.AssignNetworksFromInvites(userId, email)
			if err != nil {
				log.Println("Something went wrong while setting networks from invites:", err)
				return
			}
			err = api.AcceptAllInvites(email)
		}
	}
	return
}

func (api *API) SetUserName(id gp.UserId, firstName, lastName string) (err error) {
	return api.db.SetUserName(id, firstName, lastName)
}

func (api *API) UserWithEmail(email string) (id gp.UserId, err error) {
	return api.db.UserWithEmail(email)
}

func (api *API) ChangePass(userId gp.UserId, oldPass string, newPass string) (err error) {
	passBytes := []byte(oldPass)
	hash, err := api.db.GetHashById(userId)
	if err != nil {
		return
	} else {
		err = bcrypt.CompareHashAndPassword(hash, passBytes)
		if err != nil {
			return
		}
		hash, err = bcrypt.GenerateFromPassword([]byte(newPass), 10)
		if err != nil {
			return
		}
		err = api.db.PassUpdate(userId, hash)
		return
	}

}

func (api *API) RequestReset(email string) (err error) {
	userId, err := api.UserWithEmail(email)
	if err != nil {
		return
	}
	user, err := api.GetUser(userId)
	if err != nil {
		return
	}
	token, err := RandomString()
	if err != nil {
		return
	}
	err = api.db.AddPasswordRecovery(userId, token)
	if err != nil {
		return
	}
	err = api.issueRecoveryEmail(email, user, token)
	return
}

func (api *API) ResetPass(userId gp.UserId, token string, newPass string) (err error) {
	exists, err := api.db.CheckPasswordRecovery(userId, token)
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
	err = api.db.PassUpdate(userId, hash)
	return
}

func (api *API) IsVerified(userId gp.UserId) (verified bool, err error) {
	return api.db.IsVerified(userId)
}

func (api *API) GetLiveConversations(userId gp.UserId) (conversations []gp.ConversationSmall, err error) {
	return api.db.GetLiveConversations(userId)
}

func (api *API) DeviceFeedback(deviceId string, timestamp uint32) (err error) {
	t := time.Unix(int64(timestamp), 0)
	return api.db.Feedback(deviceId, t)
}

func (api *API) IsAdmin(user gp.UserId) (admin bool) {
	in, err := api.UserInNetwork(user, gp.NetworkId(api.Config.Admins))
	if err == nil && in {
		return true
	}
	return false
}

//CreateUserSpecial manually creates a user with these details, bypassing validation etc
func (api *API) CreateUserSpecial(first, last, email, pass string, verified bool, primaryNetwork gp.NetworkId) (err error) {
	user, err := RandomString()
	if err != nil {
		return
	}
	userId, err := api.createUser(user, pass, email)
	if err != nil {
		return
	}
	err = api.SetUserName(userId, first, last)
	if err != nil {
		return
	}
	if verified {
		err = api.db.Verify(userId)
		if err != nil {
			return
		}
	}
	err = api.setNetwork(userId, primaryNetwork)
	return
}
