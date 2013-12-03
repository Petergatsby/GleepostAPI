package lib

import (
	"code.google.com/p/go.crypto/bcrypt"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/draaglom/GleepostAPI/lib/db"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/cache"
	"io"
	"regexp"
	"strings"
	"time"
)

var ETOOWEAK = gp.APIerror{"Password too weak!"}

/********************************************************************
Top-level functions
********************************************************************/

//createToken generates a new gp.Token which expires in 24h. If something goes wrong,
//it issues a token which expires now

func randomString() (random string, err error) {
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

//createtoken might do with returning an error
//why would it break though
func createToken(userId gp.UserId) gp.Token {
	random, err := randomString()
	if err != nil {
		return (gp.Token{userId, "foo", time.Now().UTC()})
	} else {
		expiry := time.Now().Add(time.Duration(24) * time.Hour).UTC().Round(time.Second)
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

func ValidateToken(id gp.UserId, token string) bool {
	//If the db is down, this will fail for everyone who doesn't have a cached
	//token, and so no new requests will be sent.
	//I'm calling that a "feature" for now.
	conf := gp.GetConfig()
	if conf.LoginOverride {
		return (true)
	} else if cache.TokenExists(id, token) {
		return (true)
	} else {
		return db.TokenExists(id, token)
	}
}

func ValidatePass(user string, pass string) (id gp.UserId, err error) {
	passBytes := []byte(pass)
	hash, id, err := db.GetHash(user, pass)
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

func CreateAndStoreToken(id gp.UserId) (gp.Token, error) {
	token := createToken(id)
	err := db.AddToken(token)
	cache.PutToken(token)
	if err != nil {
		return token, err
	} else {
		return token, nil
	}
}

func GetUser(id gp.UserId) (user gp.User, err error) {
	/* Hits the cache then the db
	only I'm not 100% confident yet with what
	happens when you attempt to get a redis key
	that doesn't exist in redigo! */
	user, err = cache.GetUser(id)
	if err != nil {
		user, err = db.GetUser(id)
		if err == nil {
			cache.SetUser(user)
		}
	}
	return
}

func GetUserNetworks(id gp.UserId) (nets []gp.Network, err error) {
	nets, err = cache.GetUserNetwork(id)
	if err != nil {
		nets, err = db.GetUserNetworks(id)
		if err != nil {
			return
		}
		if len(nets) == 0 {
			return nets, gp.APIerror{"User has no networks!"}
		}
		cache.SetUserNetwork(id, nets[0])
	}
	return
}

func GetProfile(id gp.UserId) (user gp.Profile, err error) {
	user, err = db.GetProfile(id)
	if err != nil {
		return
	}
	nets, err := GetUserNetworks(user.Id)
	if err != nil {
		return
	}
	user.Network = nets[0]
	return
}

func ValidateEmail(email string) (validates bool, err error) {
	if !looksLikeEmail(email) {
		return false, nil
	} else {
		rules, err := db.GetRules()
		if err != nil {
			return false, err
		}
		return testEmail(email, rules), nil
	}
}

func testEmail(email string, rules []gp.Rule) bool {
	for _, rule := range rules {
		if rule.Type == "email" && strings.HasSuffix(email, rule.Value) {
			return true
		}
	}
	return false
}

func RegisterUser(user string, pass string, email string) (userId gp.UserId, err error) {
	userId, err = createUser(user, pass, email)
	if err != nil {
		return
	}
	err = generateAndSendVerification(userId, user, email)
	return
}

func createUser(user string, pass string, email string) (userId gp.UserId, err error) {
	err = checkPassStrength(pass)
	if err != nil {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		return 0, err
	}
	userId, err = db.RegisterUser(user, hash, email)
	if err != nil {
		return 0, err
	}
	_, err = assignNetworks(userId, email)
	if err != nil {
		return 0, err
	}
	return
}

//TODO: this might end up using user input directly in an email. Sanitize!
func generateAndSendVerification(userId gp.UserId, user string, email string) (err error) {
	random, err := randomString()
	if err != nil {
		return
	}
	err = db.SetVerificationToken(userId, random)
	if err != nil {
		return
	}
	err = issueVerificationEmail(email, user, random)
	return
}

func GetContacts(user gp.UserId) (contacts []gp.Contact, err error) {
	return db.GetContacts(user)
}

func AddContact(adder gp.UserId, addee gp.UserId) (contact gp.Contact, err error) {
	user, err := GetUser(addee)
	if err != nil {
		return
	}
	exists, err := ContactRequestExists(addee, adder)
	if err != nil {
		return
	}
	if exists {
		return AcceptContact(adder, addee)
	}
	err = db.AddContact(adder, addee)
	if err == nil {
		go createNotification("added_you", adder, addee, false, 0)
	}
	contact.User = user
	contact.YouConfirmed = true
	contact.TheyConfirmed = false
	return
}

func ContactRequestExists(adder gp.UserId, addee gp.UserId) (exists bool, err error) {
	return db.ContactRequestExists(adder, addee)
}

func AcceptContact(user gp.UserId, toAccept gp.UserId) (contact gp.Contact, err error) {
	err = db.UpdateContact(user, toAccept)
	if err != nil {
		return
	}
	contact.User, err = GetUser(toAccept)
	if err != nil {
		return
	}
	contact.YouConfirmed = true
	contact.TheyConfirmed = true
	go createNotification("accepted_you", user, toAccept, false, 0)
	go UnExpireBetween([]gp.UserId{user, toAccept})
	return
}

func AddDevice(user gp.UserId, deviceType string, deviceId string) (device gp.Device, err error) {
	err = db.AddDevice(user, deviceType, deviceId)
	if err != nil {
		return
	}
	device.User = user
	device.Type = deviceType
	device.Id = deviceId
	return
}

func GetDevices(user gp.UserId) (devices []gp.Device, err error) {
	return db.GetDevices(user)
}

func DeleteDevice(user gp.UserId, deviceId string) (err error) {
	return db.DeleteDevice(user, deviceId)
}

func setNetwork(userId gp.UserId, netId gp.NetworkId) (err error) {
	return db.SetNetwork(userId, netId)
}

func SetProfileImage(id gp.UserId, url string) (err error) {
	err = db.SetProfileImage(id, url)
	if err == nil {
		go cache.SetProfileImage(id, url)
	}
	return
}

func SetBusyStatus(id gp.UserId, busy bool) (err error) {
	err = db.SetBusyStatus(id, busy)
	if err == nil {
		go cache.SetBusyStatus(id, busy)
	}
	return
}

func BusyStatus(id gp.UserId) (busy bool, err error) {
	busy, err = db.BusyStatus(id)
	return
}

func userPing(id gp.UserId) {
	cache.UserPing(id)
}

func userIsOnline(id gp.UserId) bool {
	return cache.UserIsOnline(id)
}

func GetUserNotifications(id gp.UserId) (notifications []interface{}, err error) {
	return db.GetUserNotifications(id)
}

func MarkNotificationsSeen(id gp.UserId, upTo gp.NotificationId) (err error) {
	return db.MarkNotificationsSeen(id, upTo)
}

func createNotification(ntype string, by gp.UserId, recipient gp.UserId, isPN bool, post gp.PostId) (err error) {
	_, err = db.CreateNotification(ntype, by, recipient, isPN, post)
	if err == nil {
		go notificationPush(recipient)
	}
	return
}

func assignNetworks(user gp.UserId, email string) (networks int, err error) {
	conf := gp.GetConfig()
	if conf.RegisterOverride {
		setNetwork(user, 1338) //Highlands and Islands :D
	} else {
		rules, e := db.GetRules()
		if e != nil {
			return 0, e
		}
		for _, rule := range rules {
			if rule.Type == "email" && strings.HasSuffix(email, rule.Value) {
				e := setNetwork(user, rule.NetworkID)
				if e != nil {
					return networks, e
				}
				networks++
			}
		}
	}
	return
}

func verificationUrl(token string) (url string) {
	url = "https://gleepost.com/verification.html?token=" + token
	return
}

//TODO: send an actual link
func issueVerificationEmail(email string, name string, token string) (err error) {
	url := verificationUrl(token)
	html := "<html><body><a href=" + url + ">Verify your account here</a></body></html>"
	err = sendHTML(email, name+", verify your Gleepost account!", html)
	return
}

func GetEmail(id gp.UserId) (email string, err error) {
	return db.GetEmail(id)
}

//Verify will verify an account associated with a given verification token, or return an error if no such token exists.
//Additionally, if the token has been issued as part of the facebook login process, Verify will first attempt to match the verified email with an existing gleepost account, and verify that, linking the gleepost account to the facebook id.
//If no such account exists, Verify will create a new gleepost account for that facebook user and verify it.
func Verify(token string) (err error) {
	id, err := db.VerificationTokenExists(token)
	if err == nil {
		err = db.Verify(id)
		return
	}
	fbid, err := FBVerify(token)
	if err != nil {
		return
	}
	email, err := FBGetEmail(fbid)
	if err != nil {
		return
	}
	userId, err := UserWithEmail(email)
	if err != nil {
		name, e := FBName(fbid)
		if e != nil {
			return e
		}
		random, e := randomString()
		if e != nil {
			return e
		}
		id, e := createUser(name, random, email)
		if err != nil {
			return e
		}
		err = db.Verify(id)
		return
	}
	err = UserSetFB(userId, fbid)
	if err == nil {
		err = db.Verify(userId)
	}
	return
}

func UserWithEmail(email string) (id gp.UserId, err error) {
	return db.UserWithEmail(email)
}
