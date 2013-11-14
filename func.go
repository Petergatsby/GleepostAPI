package main

import (
	"code.google.com/p/go.crypto/bcrypt"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"mime/multipart"
	"regexp"
	"strings"
	"time"
)

/********************************************************************
Top-level functions
********************************************************************/

//createToken generates a new Token which expires in 24h. If something goes wrong,
//it issues a token which expires now

//createtoken might do with returning an error
//why would it break though
func createToken(userId UserId) Token {
	hash := sha256.New()
	random := make([]byte, 32) //Number pulled out of my... ahem.
	_, err := io.ReadFull(rand.Reader, random)
	if err == nil {
		hash.Write(random)
		digest := hex.EncodeToString(hash.Sum(nil))
		expiry := time.Now().Add(time.Duration(24) * time.Hour).UTC().Round(time.Second)
		token := Token{userId, digest, expiry}
		return (token)
	} else {
		return (Token{userId, "foo", time.Now().UTC()})
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

func getLastMessage(id ConversationId) (message Message, err error) {
	message, err = redisGetLastMessage(id)
	if err != nil {
		message, err = dbGetLastMessage(id)
		go redisAddAllMessages(id)
		if err != nil {
			return
		}
	}
	return
}

func validateToken(id UserId, token string) bool {
	//If the db is down, this will fail for everyone who doesn't have a cached
	//token, and so no new requests will be sent.
	//I'm calling that a "feature" for now.
	conf := GetConfig()
	if conf.LoginOverride {
		return (true)
	} else if redisTokenExists(id, token) {
		return (true)
	} else {
		return dbTokenExists(id, token)
	}
}

func validatePass(user string, pass string) (id UserId, err error) {
	hash := make([]byte, 256)
	passBytes := []byte(pass)
	s := stmt["passSelect"]
	err = s.QueryRow(user).Scan(&id, &hash)
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

func createAndStoreToken(id UserId) (Token, error) {
	token := createToken(id)
	err := dbAddToken(token)
	redisPutToken(token)
	if err != nil {
		return token, err
	} else {
		return token, nil
	}
}

func getUser(id UserId) (user User, err error) {
	/* Hits the cache then the db
	only I'm not 100% confident yet with what
	happens when you attempt to get a redis key
	that doesn't exist in redigo! */
	user, err = redisGetUser(id)
	if err != nil {
		user, err = dbGetUser(id)
		if err == nil {
			redisSetUser(user)
		}
	}
	return
}

func getCommentCount(id PostId) (count int) {
	count, err := redisGetCommentCount(id)
	if err != nil {
		count = dbGetCommentCount(id)
	}
	return count
}

func createComment(postId PostId, userId UserId, text string) (commId CommentId, err error) {
	post, err := getPost(postId)
	if err != nil {
		return
	}
	commId, err = dbCreateComment(postId, userId, text)
	if err == nil {
		user, e := getUser(userId)
		if e != nil {
			return commId, e
		}
		comment := Comment{Id: commId, Post: postId, By: user, Time: time.Now().UTC(), Text: text}
		go createNotification("commented", userId, post.By.Id, true, postId)
		go redisAddComment(postId, comment)
	}
	return commId, err
}

func getUserNetworks(id UserId) (nets []Network, err error) {
	nets, err = redisGetUserNetwork(id)
	if err != nil {
		nets, err = dbGetUserNetworks(id)
		if err != nil {
			return
		}
		if len(nets) == 0 {
			return nets, APIerror{"User has no networks!"}
		}
		redisSetUserNetwork(id, nets[0])
	}
	return
}

func getParticipants(convId ConversationId) []User {
	participants, err := redisGetConversationParticipants(convId)
	if err != nil {
		participants = dbGetParticipants(convId)
		go redisSetConversationParticipants(convId, participants)
	}
	return participants
}

func getMessages(convId ConversationId, index int64, sel string) (messages []Message, err error) {
	conf := GetConfig()
	messages, err = redisGetMessages(convId, index, sel, conf.MessagePageSize)
	if err != nil {
		messages, err = dbGetMessages(convId, index, sel, conf.MessagePageSize)
		go redisAddAllMessages(convId)
		return
	}
	return
}

func getConversations(userId UserId, start int64) (conversations []ConversationSmall, err error) {
	conf := GetConfig()
	conversations, err = redisGetConversations(userId, start)
	if err != nil {
		conversations, err = dbGetConversations(userId, start, conf.ConversationPageSize)
		go addAllConversations(userId)
	}
	return
}

func addAllConversations(userId UserId) (err error) {
	conf := GetConfig()
	conversations, err := dbGetConversations(userId, 0, conf.ConversationPageSize)
	for _, conv := range conversations {
		go redisAddConversation(conv.Conversation)
	}
	return
}

func getConversation(userId UserId, convId ConversationId) (conversation ConversationAndMessages, err error) {
	//redisGetConversation
	return dbGetConversation(convId)
}

func getMessage(msgId MessageId) (message Message, err error) {
	message, err = redisGetMessage(msgId)
	return message, err
}

func updateConversation(id ConversationId) (err error) {
	err = dbUpdateConversation(id)
	if err != nil {
		return err
	}
	go redisUpdateConversation(id)
	return nil
}

func addMessage(convId ConversationId, userId UserId, text string) (messageId MessageId, err error) {
	messageId, err = dbAddMessage(convId, userId, text)
	if err != nil {
		return
	}
	user, err := getUser(userId)
	if err != nil {
		return
	}
	msgSmall := Message{MessageId(messageId), user, text, time.Now().UTC(), false}
	msg := RedisMessage{msgSmall, convId}
	go redisPublish(msg)
	go redisAddMessage(msgSmall, convId)
	go updateConversation(convId)
	return
}

func getFullConversation(convId ConversationId, start int64) (conv ConversationAndMessages, err error) {
	conv.Id = convId
	conv.LastActivity, err = ConversationLastActivity(convId)
	if err != nil {
		return
	}
	conv.Participants = getParticipants(convId)
	conv.Messages, err = getMessages(convId, start, "start")
	return
}

func ConversationLastActivity(convId ConversationId) (t time.Time, err error) {
	return dbConversationActivity(convId)
}

func getPostImages(postId PostId) (images []string) {
	images, _ = dbGetPostImages(postId)
	return
}

func addPostImage(postId PostId, url string) (err error) {
	return dbAddPostImage(postId, url)
}

func getProfile(id UserId) (user Profile, err error) {
	user, err = dbGetProfile(id)
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

func awaitOneMessage(userId UserId) (resp []byte) {
	c := getMessageChan(userId)
	select {
	case resp = <-c:
		return
	case <-time.After(60 * time.Second):
		return []byte("{}")
	}
}

func getMessageChan(userId UserId) (c chan []byte) {
	return redisMessageChan(userId)
}

func addPost(userId UserId, text string) (postId PostId, err error) {
	networks, err := getUserNetworks(userId)
	if err != nil {
		return
	}
	postId, err = dbAddPost(userId, text, networks[0].Id)
	if err == nil {
		go redisAddNewPost(userId, text, postId)
	}
	return
}

func getPosts(netId NetworkId, index int64, sel string) (posts []PostSmall, err error) {
	conf := GetConfig()
	posts, err = redisGetNetworkPosts(netId, index, sel)
	if err != nil {
		posts, err = dbGetPosts(netId, index, conf.PostPageSize, sel)
		go redisAddAllPosts(netId)
	}
	return
}

func getComments(id PostId, start int64) (comments []Comment, err error) {
	conf := GetConfig()
	if start+int64(conf.CommentPageSize) <= int64(conf.CommentCache) {
		comments, err = redisGetComments(id, start)
		if err != nil {
			comments, err = dbGetComments(id, start, conf.CommentPageSize)
			go redisAddAllComments(id)
		}
	} else {
		comments, err = dbGetComments(id, start, conf.CommentPageSize)
	}
	return
}

func createConversation(id UserId, nParticipants int, live bool) (conversation Conversation, err error) {
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
	conversation, err = dbCreateConversation(id, participants, live)
	if err == nil {
		go redisAddConversation(conversation)
	}
	return
}

func validateEmail(email string) (validates bool, err error) {
	if !looksLikeEmail(email) {
		return false, nil
	} else {
		rules, err := dbGetRules()
		if err != nil {
			return false, err
		}
		return testEmail(email, rules), nil
	}
}

func testEmail(email string, rules []Rule) bool {
	for _, rule := range rules {
		if rule.Type == "email" && strings.HasSuffix(email, rule.Value) {
			return true
		}
	}
	return false
}

func registerUser(user string, pass string, email string) (userId UserId, err error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		return 0, err
	}
	userId, err = dbRegisterUser(user, hash, email)
	if err != nil {
		return 0, err
	}
	_, err = assignNetworks(userId, email)
	return
}

func getContacts(user UserId) (contacts []Contact, err error) {
	return dbGetContacts(user)
}

func addContact(adder UserId, addee UserId) (user User, err error) {
	user, err = getUser(addee)
	if err != nil {
		return
	} else {
		err = dbAddContact(adder, addee)
		if err == nil {
			go createNotification("added_you", adder, addee, false, 0)
		}
		return
	}
}

func acceptContact(user UserId, toAccept UserId) (contact Contact, err error) {
	err = dbUpdateContact(user, toAccept)
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

func addDevice(user UserId, deviceType string, deviceId string) (device Device, err error) {
	err = dbAddDevice(user, deviceType, deviceId)
	if err != nil {
		return
	}
	device.User = user
	device.Type = deviceType
	device.Id = deviceId
	return
}

func getDevices(user UserId) (devices []Device, err error) {
	return dbGetDevices(user)
}

func generatePartners(id UserId, count int, network NetworkId) (partners []User, err error) {
	return dbRandomPartners(id, count, network)
}

func markConversationSeen(id UserId, convId ConversationId, upTo MessageId) (conversation ConversationAndMessages, err error) {
	err = dbMarkRead(id, convId, upTo)
	if err != nil {
		return
	}
	err = redisMarkConversationSeen(id, convId, upTo)
	if err != nil {
		go redisAddAllMessages(convId)
	}
	conversation, err = dbGetConversation(convId)
	return
}

func setNetwork(userId UserId, netId NetworkId) (err error) {
	return dbSetNetwork(userId, netId)
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
	conf := GetConfig()
	var auth aws.Auth
	auth.AccessKey, auth.SecretKey = conf.AWS.KeyId, conf.AWS.SecretKey
	s = s3.New(auth, aws.EUWest)
	return
}

func storeFile(id UserId, file multipart.File, header *multipart.FileHeader) (url string, err error) {
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
		return "", APIerror{"Unsupported file type"}
	}
	if err != nil {
		return "", APIerror{err.Error()}
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

func userAddUpload(id UserId, url string) (err error) {
	return dbAddUpload(id, url)
}

func userUploadExists(id UserId, url string) (exists bool, err error) {
	return dbUploadExists(id, url)
}

func setProfileImage(id UserId, url string) (err error) {
	err = dbSetProfileImage(id, url)
	if err == nil {
		go redisSetProfileImage(id, url)
	}
	return
}

func setBusyStatus(id UserId, busy bool) (err error) {
	err = dbSetBusyStatus(id, busy)
	if err == nil {
		go redisSetBusyStatus(id, busy)
	}
	return
}

func userPing(id UserId) {
	redisUserPing(id)
}

func userIsOnline(id UserId) bool {
	return redisUserIsOnline(id)
}

func getUserNotifications(id UserId) (notifications []interface{}, err error) {
	return dbGetUserNotifications(id)
}

func markNotificationsSeen(upTo NotificationId) (err error) {
	return dbMarkNotificationsSeen(upTo)
}

func createNotification(ntype string, by UserId, recipient UserId, isPN bool, post PostId) (err error) {
	_, err = dbCreateNotification(ntype, by, recipient, isPN, post)
	return
}

func assignNetworks(user UserId, email string) (networks int, err error) {
	conf := GetConfig()
	if conf.RegisterOverride {
		setNetwork(user, 1338) //Highlands and Islands :D
	} else {
		rules, e := dbGetRules()
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

func getPost(postId PostId) (post Post, err error) {
	return dbGetPost(postId)
}

func getPostFull(postId PostId) (post PostFull, err error) {
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

func addLike(user UserId, postId PostId) (err error) {
	//TODO: add like to redis
	post, err := getPost(postId)
	if err != nil {
		return
	} else {
		err = dbCreateLike(user, postId)
		if err != nil {
			return
		} else {
			createNotification("liked", user, post.By.Id, true, postId)
		}
	}
	return
}

func delLike(user UserId, post PostId) (err error) {
	return dbRemoveLike(user, post)
}

func getLikes(post PostId) (likes []LikeFull, err error) {
	l, err := dbGetLikes(post)
	if err != nil {
		return
	}
	for _, like := range l {
		lf := LikeFull{}
		lf.User, err = getUser(like.UserID)
		if err != nil {
			return
		}
		lf.Time = like.Time
		likes = append(likes, lf)
	}
	return
}

func hasLiked(user UserId, post PostId) (liked bool, err error) {
	return dbHasLiked(user, post)
}

func likeCount(post PostId) (count int, err error) {
	return dbLikeCount(post)
}

func conversationExpiry(convId ConversationId) (expiry Expiry, err error) {
	return dbConversationExpiry(convId)
}
