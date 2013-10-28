package main

import (
	"code.google.com/p/go.crypto/bcrypt"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"regexp"
	"time"
	"mime/multipart"
	"strings"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"io/ioutil"
)

/********************************************************************
Top-level functions
********************************************************************/

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
		redisSetUser(user)
	}
	return user, err
}

func getCommentCount(id PostId) (count int) {
	count, err := redisGetCommentCount(id)
	if err != nil {
		count = dbGetCommentCount(id)
	}
	return count
}

func createComment(postId PostId, userId UserId, text string) (commId CommentId, err error) {
	commId, err = dbCreateComment(postId, userId, text)
	if err == nil {
		user, e := getUser(userId)
		if e != nil {
			return commId, e
		}
		comment := Comment{Id: commId, Post: postId, By: user, Time: time.Now().UTC(), Text: text}
		go redisAddComment(postId, comment)
	}
	return commId, err
}

func getUserNetworks(id UserId) (nets []Network) {
	nets, err := redisGetUserNetwork(id)
	if err != nil {
		nets = dbGetUserNetworks(id)
		redisSetUserNetwork(id, nets[0])
	}
	return (nets)
}

func getParticipants(convId ConversationId) []User {
	participants, err := redisGetConversationParticipants(convId)
	if err != nil {
		participants = dbGetParticipants(convId)
		go redisSetConversationParticipants(convId, participants)
	}
	return participants
}

func getMessages(convId ConversationId, start int64) (messages []Message, err error) {
	conf := GetConfig()
	if start+int64(conf.MessagePageSize) <= int64(conf.MessageCache) {
		messages, err = redisGetMessages(convId, start)
		if err != nil {
			messages, err = dbGetMessages(convId, start, false)
			go redisAddAllMessages(convId)
		}
	} else {
		messages, err = dbGetMessages(convId, start, false)
	}
	return
}

func getMessagesAfter(convId ConversationId, after int64) (messages []Message, err error) {
	messages, err = redisGetMessagesAfter(convId, after)
	if err != nil {
		messages, err = dbGetMessages(convId, after, true)
		go redisAddAllMessages(convId)
	}
	return
}

func getConversations(user_id UserId, start int64) (conversations []ConversationSmall, err error) {
	conversations, err = redisGetConversations(user_id, start)
	if err != nil {
		conversations, err = dbGetConversations(user_id, start)
		if err == nil {
			for _, conv := range conversations {
				go redisAddConversation(conv)
			}
		}
		return
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
	conv.Conversation.Id = convId
	conv.Conversation.Participants = getParticipants(convId)
	conv.Messages, err = getMessages(convId, start)
	return
}

func getPostImages(postId PostId) (images []string) {
	images, _ = dbGetPostImages(postId)
	return
}

func getProfile(id UserId) (user Profile, err error) {
	user, err = dbGetProfile(id)
	return
}

func awaitOneMessage(userId UserId) []byte {
	return redisAwaitOneMessage(userId)
}

func addPost(userId UserId, text string) (postId PostId, err error) {
	postId, err = dbAddPost(userId, text)
	go redisAddNewPost(userId, text, postId)
	return
}

func getPosts(netId NetworkId, start int64) (posts []PostSmall, err error) {
	conf := GetConfig()
	if start+int64(conf.PostPageSize) <= int64(conf.PostCache) {
		posts, err = redisGetNetworkPosts(netId, start)
		if err != nil {
			posts, err = dbGetPosts(netId, start, conf.PostPageSize)
			go redisAddAllPosts(netId)
		}
	} else {
		posts, err = dbGetPosts(netId, start, conf.PostPageSize)
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

func createConversation(id UserId, nParticipants int) (conversation Conversation, err error) {
	participants, err := generatePartners(id, nParticipants-1)
	if err != nil {
		return
	}
	user, err := getUser(id)
	if err != nil {
		return
	}
	participants = append(participants, user)
	return dbCreateConversation(id, participants)
}

func validateEmail(email string) bool {
	if !looksLikeEmail(email) {
		return (false)
	} else {
		return dbValidateEmail(email)
	}
}

func registerUser(user string, pass string, email string) (userId UserId, err error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		return 0, err
	}
	userId, err = dbRegisterUser(user, hash, email)
	conf := GetConfig()
	if conf.RegisterOverride {
		setNetwork(userId, 1338) //Highlands and Islands :D
	}
	return userId, err
}

func getContacts(user UserId) (contacts []Contact, err error) {
	return dbGetContacts(user)
}

func addContact(adder UserId, addee UserId) (user User, err error) {
	// Todo : actually add contact
	user, err = getUser(addee)
	if err != nil {
		return
	} else {
		err = dbAddContact(adder, addee)
		return
	}
}

func acceptContact(user UserId, toAccept UserId) (contact Contact, err error) {
	err = dbUpdateContact(user, toAccept)
	if err != nil {
		contact.User, err = getUser(toAccept)
		if err != nil {
			return
		}
		contact.YouConfirmed = true
		contact.TheyConfirmed = true
	}
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

func generatePartners(id UserId, count int) (partners []User, err error) {
	return dbRandomPartners(id, count)
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
