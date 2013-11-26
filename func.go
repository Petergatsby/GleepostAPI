package main

import (
	"code.google.com/p/go.crypto/bcrypt"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/draaglom/GleepostAPI/db"
	"github.com/draaglom/GleepostAPI/gp"
	"github.com/draaglom/GleepostAPI/cache"
	"io"
	"io/ioutil"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"mime/multipart"
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

func getLastMessage(id gp.ConversationId) (message gp.Message, err error) {
	message, err = cache.GetLastMessage(id)
	if err != nil {
		message, err = db.GetLastMessage(id)
		go cache.AddAllMessages(id)
		if err != nil {
			return
		}
	}
	return
}

func validateToken(id gp.UserId, token string) bool {
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

func validatePass(user string, pass string) (id gp.UserId, err error) {
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

func createAndStoreToken(id gp.UserId) (gp.Token, error) {
	token := createToken(id)
	err := db.AddToken(token)
	cache.PutToken(token)
	if err != nil {
		return token, err
	} else {
		return token, nil
	}
}

func getUser(id gp.UserId) (user gp.User, err error) {
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

func getCommentCount(id gp.PostId) (count int) {
	count, err := cache.GetCommentCount(id)
	if err != nil {
		count = db.GetCommentCount(id)
	}
	return count
}

func createComment(postId gp.PostId, userId gp.UserId, text string) (commId gp.CommentId, err error) {
	post, err := getPost(postId)
	if err != nil {
		return
	}
	commId, err = db.CreateComment(postId, userId, text)
	if err == nil {
		user, e := getUser(userId)
		if e != nil {
			return commId, e
		}
		comment := gp.Comment{Id: commId, Post: postId, By: user, Time: time.Now().UTC(), Text: text}
		go createNotification("commented", userId, post.By.Id, true, postId)
		go cache.AddComment(postId, comment)
	}
	return commId, err
}

func getUserNetworks(id gp.UserId) (nets []gp.Network, err error) {
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

func getParticipants(convId gp.ConversationId) []gp.User {
	participants, err := cache.GetParticipants(convId)
	if err != nil {
		participants = db.GetParticipants(convId)
		go cache.SetConversationParticipants(convId, participants)
	}
	return participants
}

func getMessages(convId gp.ConversationId, index int64, sel string) (messages []gp.Message, err error) {
	conf := gp.GetConfig()
	messages, err = cache.GetMessages(convId, index, sel, conf.MessagePageSize)
	if err != nil {
		messages, err = db.GetMessages(convId, index, sel, conf.MessagePageSize)
		go cache.AddAllMessages(convId)
		return
	}
	return
}

func getConversations(userId gp.UserId, start int64) (conversations []gp.ConversationSmall, err error) {
	conf := gp.GetConfig()
	conversations, err = cache.GetConversations(userId, start, conf.ConversationPageSize)
	if err != nil {
		conversations, err = db.GetConversations(userId, start, conf.ConversationPageSize)
		go addAllConversations(userId)
	}
	return
}

func addAllConversations(userId gp.UserId) (err error) {
	conf := gp.GetConfig()
	conversations, err := db.GetConversations(userId, 0, conf.ConversationPageSize)
	for _, conv := range conversations {
		go cache.AddConversation(conv.Conversation)
	}
	return
}

func getConversation(userId gp.UserId, convId gp.ConversationId) (conversation gp.ConversationAndMessages, err error) {
	//cache.GetConversation
	return db.GetConversation(convId)
}

func getMessage(msgId gp.MessageId) (message gp.Message, err error) {
	message, err = cache.GetMessage(msgId)
	return message, err
}

func updateConversation(id gp.ConversationId) (err error) {
	err = db.UpdateConversation(id)
	if err != nil {
		return err
	}
	go cache.UpdateConversation(id)
	return nil
}

func addMessage(convId gp.ConversationId, userId gp.UserId, text string) (messageId gp.MessageId, err error) {
	messageId, err = db.AddMessage(convId, userId, text)
	if err != nil {
		return
	}
	user, err := getUser(userId)
	if err != nil {
		return
	}
	msg := gp.Message{gp.MessageId(messageId), user, text, time.Now().UTC(), false}
	go cache.Publish(msg, convId)
	go cache.AddMessage(msg, convId)
	go updateConversation(convId)
	go messagePush(msg, convId)
	return
}

func getFullConversation(convId gp.ConversationId, start int64) (conv gp.ConversationAndMessages, err error) {
	conv.Id = convId
	conv.LastActivity, err = ConversationLastActivity(convId)
	if err != nil {
		return
	}
	conv.Participants = getParticipants(convId)
	conv.Messages, err = getMessages(convId, start, "start")
	return
}

func ConversationLastActivity(convId gp.ConversationId) (t time.Time, err error) {
	return db.ConversationActivity(convId)
}

func getPostImages(postId gp.PostId) (images []string) {
	images, _ = db.GetPostImages(postId)
	return
}

func addPostImage(postId gp.PostId, url string) (err error) {
	return db.AddPostImage(postId, url)
}

func getProfile(id gp.UserId) (user gp.Profile, err error) {
	user, err = db.GetProfile(id)
	if err != nil {
		return
	}
	nets, err := getUserNetworks(user.Id)
	if err != nil {
		return
	}
	user.Network = nets[0]
	return
}

func awaitOneMessage(userId gp.UserId) (resp []byte) {
	c := getMessageChan(userId)
	select {
	case resp = <-c:
		return
	case <-time.After(60 * time.Second):
		return []byte("{}")
	}
}

func getMessageChan(userId gp.UserId) (c chan []byte) {
	return cache.MessageChan(userId)
}

func addPost(userId gp.UserId, text string) (postId gp.PostId, err error) {
	networks, err := getUserNetworks(userId)
	if err != nil {
		return
	}
	postId, err = db.AddPost(userId, text, networks[0].Id)
	if err == nil {
		go cache.AddNewPost(userId, text, postId, networks[0].Id)
	}
	return
}

func getPosts(netId gp.NetworkId, index int64, sel string) (posts []gp.PostSmall, err error) {
	conf := gp.GetConfig()
	posts, err = cache.GetPosts(netId, index, conf.PostPageSize, sel)
	if err != nil {
		posts, err = db.GetPosts(netId, index, conf.PostPageSize, sel)
		go cache.AddAllPosts(netId)
	}
	return
}

func getComments(id gp.PostId, start int64) (comments []gp.Comment, err error) {
	conf := gp.GetConfig()
	if start+int64(conf.CommentPageSize) <= int64(conf.CommentCache) {
		comments, err = cache.GetComments(id, start)
		if err != nil {
			comments, err = db.GetComments(id, start, conf.CommentPageSize)
			go cache.AddAllComments(id)
		}
	} else {
		comments, err = db.GetComments(id, start, conf.CommentPageSize)
	}
	return
}

func createConversation(id gp.UserId, nParticipants int, live bool) (conversation gp.Conversation, err error) {
	networks, err := getUserNetworks(id)
	if err != nil {
		return
	}
	participants, err := generatePartners(id, nParticipants-1, networks[0].Id)
	if err != nil {
		return
	}
	user, err := getUser(id)
	if err != nil {
		return
	}
	participants = append(participants, user)
	conversation, err = db.CreateConversation(id, participants, live)
	if err == nil {
		go cache.AddConversation(conversation)
	}
	return
}

func validateEmail(email string) (validates bool, err error) {
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

func registerUser(user string, pass string, email string) (userId gp.UserId, err error) {
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

func getContacts(user gp.UserId) (contacts []gp.Contact, err error) {
	return db.GetContacts(user)
}

func addContact(adder gp.UserId, addee gp.UserId) (user gp.User, err error) {
	user, err = getUser(addee)
	if err != nil {
		return
	} else {
		err = db.AddContact(adder, addee)
		if err == nil {
			go createNotification("added_you", adder, addee, false, 0)
		}
		return
	}
}

func acceptContact(user gp.UserId, toAccept gp.UserId) (contact gp.Contact, err error) {
	err = db.UpdateContact(user, toAccept)
	if err != nil {
		return
	}
	contact.User, err = getUser(toAccept)
	if err != nil {
		return
	}
	contact.YouConfirmed = true
	contact.TheyConfirmed = true
	go createNotification("accepted_you", user, toAccept, false, 0)
	return
}

func addDevice(user gp.UserId, deviceType string, deviceId string) (device gp.Device, err error) {
	err = db.AddDevice(user, deviceType, deviceId)
	if err != nil {
		return
	}
	device.User = user
	device.Type = deviceType
	device.Id = deviceId
	return
}

func getDevices(user gp.UserId) (devices []gp.Device, err error) {
	return db.GetDevices(user)
}

func deleteDevice(user gp.UserId, deviceId string) (err error) {
	return db.DeleteDevice(user, deviceId)
}

func generatePartners(id gp.UserId, count int, network gp.NetworkId) (partners []gp.User, err error) {
	return db.RandomPartners(id, count, network)
}

func markConversationSeen(id gp.UserId, convId gp.ConversationId, upTo gp.MessageId) (conversation gp.ConversationAndMessages, err error) {
	err = db.MarkRead(id, convId, upTo)
	if err != nil {
		return
	}
	err = cache.MarkConversationSeen(id, convId, upTo)
	if err != nil {
		go cache.AddAllMessages(convId)
	}
	conversation, err = db.GetConversation(convId)
	return
}

func setNetwork(userId gp.UserId, netId gp.NetworkId) (err error) {
	return db.SetNetwork(userId, netId)
}

func randomFilename(extension string) (string, error) {
	hash := sha256.New()
	random := make([]byte, 32) //Number pulled out of my... ahem.
	_, err := io.ReadFull(rand.Reader, random)
	if err == nil {
		hash.Write(random)
		digest := hex.EncodeToString(hash.Sum(nil))
		return digest + extension, nil
	} else {
		return "", err
	}
}

func getS3() (s *s3.S3) {
	conf := gp.GetConfig()
	var auth aws.Auth
	auth.AccessKey, auth.SecretKey = conf.AWS.KeyId, conf.AWS.SecretKey
	s = s3.New(auth, aws.EUWest)
	return
}

func storeFile(id gp.UserId, file multipart.File, header *multipart.FileHeader) (url string, err error) {
	var filename string
	var contenttype string
	switch {
	case strings.HasSuffix(header.Filename, ".jpg"):
		filename, err = randomFilename(".jpg")
		contenttype = "image/jpeg"
	case strings.HasSuffix(header.Filename, ".jpeg"):
		filename, err = randomFilename(".jpg")
		contenttype = "image/jpeg"
	case strings.HasSuffix(header.Filename, ".png"):
		filename, err = randomFilename(".png")
		contenttype = "image/png"
	default:
		return "", gp.APIerror{"Unsupported file type"}
	}
	if err != nil {
		return "", gp.APIerror{err.Error()}
	}
	//store on s3
	s := getS3()
	bucket := s.Bucket("gpimg")
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	err = bucket.Put(filename, data, contenttype, s3.PublicRead)
	url = bucket.URL(filename)
	if err != nil {
		return "", err
	}
	err = userAddUpload(id, url)
	return url, err
}

func userAddUpload(id gp.UserId, url string) (err error) {
	return db.AddUpload(id, url)
}

func userUploadExists(id gp.UserId, url string) (exists bool, err error) {
	return db.UploadExists(id, url)
}

func setProfileImage(id gp.UserId, url string) (err error) {
	err = db.SetProfileImage(id, url)
	if err == nil {
		go cache.SetProfileImage(id, url)
	}
	return
}

func setBusyStatus(id gp.UserId, busy bool) (err error) {
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

func getUserNotifications(id gp.UserId) (notifications []interface{}, err error) {
	return db.GetUserNotifications(id)
}

func markNotificationsSeen(id gp.UserId, upTo gp.NotificationId) (err error) {
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

func getPost(postId gp.PostId) (post gp.Post, err error) {
	return db.GetPost(postId)
}

func getPostFull(postId gp.PostId) (post gp.PostFull, err error) {
	post.Post, err = getPost(postId)
	if err != nil {
		return
	}
	post.Comments, err = getComments(postId, 0)
	if err != nil {
		return
	}
	post.Likes, err = getLikes(postId)
	return
}

func addLike(user gp.UserId, postId gp.PostId) (err error) {
	//TODO: add like to redis
	post, err := getPost(postId)
	if err != nil {
		return
	} else {
		err = db.CreateLike(user, postId)
		if err != nil {
			return
		} else {
			createNotification("liked", user, post.By.Id, true, postId)
		}
	}
	return
}

func delLike(user gp.UserId, post gp.PostId) (err error) {
	return db.RemoveLike(user, post)
}

func getLikes(post gp.PostId) (likes []gp.LikeFull, err error) {
	l, err := db.GetLikes(post)
	if err != nil {
		return
	}
	for _, like := range l {
		lf := gp.LikeFull{}
		lf.User, err = getUser(like.UserID)
		if err != nil {
			return
		}
		lf.Time = like.Time
		likes = append(likes, lf)
	}
	return
}

func hasLiked(user gp.UserId, post gp.PostId) (liked bool, err error) {
	return db.HasLiked(user, post)
}

func likeCount(post gp.PostId) (count int, err error) {
	return db.LikeCount(post)
}

func conversationExpiry(convId gp.ConversationId) (expiry gp.Expiry, err error) {
	return db.ConversationExpiry(convId)
}

func verificationUrl(token string) (url string) {
	url = "https://gleepost.com/verification.html?token=" + token
	return
}

//TODO: send an actual link
func issueVerificationEmail(email string, name string, token string) (err error) {
	err = send(email, name+", verify your Gleepost account!", verificationUrl(token))
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

func terminateConversation(convId gp.ConversationId) (err error) {
	return db.TerminateConversation(convId)
}
