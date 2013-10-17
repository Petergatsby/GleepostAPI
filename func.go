package main

import (
	"code.google.com/p/go.crypto/bcrypt"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"time"
	"github.com/garyburd/redigo/redis"
	"log"
)

/********************************************************************
Top-level functions
********************************************************************/

func createToken(userId UserId) Token {
	hash := sha256.New()
	random := make([]byte, 10)
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
	conf := GetConfig()
	if conf.LoginOverride {
		return (true)
	} else if redisTokenExists(id, token) {
		return (true)
	} else {
		var expiry string
		err := tokenSelectStmt.QueryRow(id, token).Scan(&expiry)
		if err != nil {
			return (false)
		} else {
			t, _ := time.Parse(MysqlTime, expiry)
			if t.After(time.Now()) {
				return (true)
			}
			return (false)
		}
	}
}

func validatePass(user string, pass string) (id UserId, err error) {
	hash := make([]byte, 256)
	passBytes := []byte(pass)
	err = passStmt.QueryRow(user).Scan(&id, &hash)
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
	_, err := tokenInsertStmt.Exec(token.UserId, token.Token, token.Expiry)
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
		go redisSetCommentCount(id, count)
	}
	return count
}

func createComment(postId PostId, userId UserId, text string) (commId CommentId, err error) {
	commId, err = dbCreateComment(postId, userId, text)
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
			messages, err = dbGetMessages(convId, start)
			go redisAddAllMessages(convId)
		}
	} else {
		messages, err = dbGetMessages(convId, start)
	}
	return
}

func getMessagesAfter(convId ConversationId, after int64) (messages []Message, err error) {
	messages, err = redisGetMessagesAfter(convId, after)
	if err != nil {
		messages, err = dbGetMessagesAfter(convId, after)
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
	log.Printf("Adding message to db: %d, %d %s", convId, userId, text)
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
	conn := pool.Get()
	defer conn.Close()
	psc := redis.PubSubConn{Conn: conn}
	psc.Subscribe(userId)
	defer psc.Unsubscribe(userId)
	for {
		switch n := psc.Receive().(type) {
		case redis.Message:
			return n.Data
		case redis.Subscription:
			fmt.Printf("%s: %s %d\n", n.Channel, n.Kind, n.Count)
		}
	}
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
	return dbCreateConversation(id, nParticipants)
}

func validateEmail(email string) bool {
	if !looksLikeEmail(email) {
		return (false)
	} else {
		return dbValidateEmail(email)
	}
}

func registerUser(user string, pass string, email string) (UserId, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		return 0, err
	}
	return dbRegisterUser(user, hash, email)
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
