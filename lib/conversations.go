package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/db"
	"github.com/draaglom/GleepostAPI/lib/cache"
	"time"
)

func TerminateConversation(convId gp.ConversationId) (err error) {
	err = db.TerminateConversation(convId)
	if err == nil {
		go cache.TerminateConversation(convId)
	}
	return
}

func generatePartners(id gp.UserId, count int, network gp.NetworkId) (partners []gp.User, err error) {
	return db.RandomPartners(id, count, network)
}

func MarkConversationSeen(id gp.UserId, convId gp.ConversationId, upTo gp.MessageId) (conversation gp.ConversationAndMessages, err error) {
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

func CreateConversation(id gp.UserId, nParticipants int, live bool) (conversation gp.Conversation, err error) {
	networks, err := GetUserNetworks(id)
	if err != nil {
		return
	}
	participants, err := generatePartners(id, nParticipants-1, networks[0].Id)
	if err != nil {
		return
	}
	user, err := GetUser(id)
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

func AwaitOneMessage(userId gp.UserId) (resp []byte) {
	c := GetMessageChan(userId)
	select {
	case resp = <-c:
		return
	case <-time.After(60 * time.Second):
		return []byte("{}")
	}
}

func GetMessageChan(userId gp.UserId) (c chan []byte) {
	return cache.MessageChan(userId)
}

func addAllConversations(userId gp.UserId) (err error) {
	conf := gp.GetConfig()
	conversations, err := db.GetConversations(userId, 0, conf.ConversationPageSize)
	for _, conv := range conversations {
		go cache.AddConversation(conv.Conversation)
	}
	return
}

func GetConversation(userId gp.UserId, convId gp.ConversationId) (conversation gp.ConversationAndMessages, err error) {
	//cache.GetConversation
	return db.GetConversation(convId)
}

func GetMessage(msgId gp.MessageId) (message gp.Message, err error) {
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

func AddMessage(convId gp.ConversationId, userId gp.UserId, text string) (messageId gp.MessageId, err error) {
	messageId, err = db.AddMessage(convId, userId, text)
	if err != nil {
		return
	}
	user, err := GetUser(userId)
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

func GetFullConversation(convId gp.ConversationId, start int64) (conv gp.ConversationAndMessages, err error) {
	conv.Id = convId
	conv.LastActivity, err = ConversationLastActivity(convId)
	if err != nil {
		return
	}
	conv.Participants = GetParticipants(convId)
	conv.Messages, err = GetMessages(convId, start, "start")
	return
}

func ConversationLastActivity(convId gp.ConversationId) (t time.Time, err error) {
	return db.ConversationActivity(convId)
}

func GetParticipants(convId gp.ConversationId) []gp.User {
	participants, err := cache.GetParticipants(convId)
	if err != nil {
		participants = db.GetParticipants(convId)
		go cache.SetConversationParticipants(convId, participants)
	}
	return participants
}

func GetMessages(convId gp.ConversationId, index int64, sel string) (messages []gp.Message, err error) {
	conf := gp.GetConfig()
	messages, err = cache.GetMessages(convId, index, sel, conf.MessagePageSize)
	if err != nil {
		messages, err = db.GetMessages(convId, index, sel, conf.MessagePageSize)
		go cache.AddAllMessages(convId)
		return
	}
	return
}

func GetConversations(userId gp.UserId, start int64) (conversations []gp.ConversationSmall, err error) {
	conf := gp.GetConfig()
	conversations, err = cache.GetConversations(userId, start, conf.ConversationPageSize)
	if err != nil {
		conversations, err = db.GetConversations(userId, start, conf.ConversationPageSize)
		go addAllConversations(userId)
	} else {
		//This is here because cache.GetConversations doesn't get the expiry itself...
		for i, c := range(conversations) {
			exp, err := Expiry(c.Id)
			if err == nil {
				conversations[i].Expiry = &exp
			}
		}
	}
	return
}

func GetLastMessage(id gp.ConversationId) (message gp.Message, err error) {
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

func Expiry(convId gp.ConversationId) (expiry gp.Expiry, err error) {
	expiry, err = cache.ConversationExpiry(convId)
	if err != nil {
		expiry, err = db.ConversationExpiry(convId)
		if err == nil {
			cache.SetConversationExpiry(convId, expiry)
		}
	}
	return
}

func DeleteExpiry(convId gp.ConversationId) (err error) {
	err = db.DeleteConversationExpiry(convId)
	if err == nil {
		go cache.DelConversationExpiry(convId)
	}
	return
}

