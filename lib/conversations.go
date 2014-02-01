package lib

import (
	"fmt"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"log"
	"time"
)

var ENOTALLOWED = gp.APIerror{"You're not allowed to message this user!"}

func (api *API) TerminateConversation(convId gp.ConversationId) (err error) {
	err = api.db.TerminateConversation(convId)
	if err == nil {
		go api.cache.TerminateConversation(convId)
		go api.EndConversationEvent(convId)
	}
	return
}

func (api *API) generatePartners(id gp.UserId, count int, network gp.NetworkId) (partners []gp.User, err error) {
	return api.db.RandomPartners(id, count, network)
}

//MarkConversationSeen sets the "read" location to upTo for user id in conversation convId.
func (api *API) MarkConversationSeen(id gp.UserId, convId gp.ConversationId, upTo gp.MessageId) (err error) {
	err = api.db.MarkRead(id, convId, upTo)
	if err != nil {
		return
	}
	api.cache.MarkConversationSeen(id, convId, upTo)
	return
}

func (api *API) CreateConversation(initiator gp.UserId, participants []gp.User, live bool) (conversation gp.Conversation, err error) {
	var expiry *gp.Expiry
	if live {
		expiry = gp.NewExpiry(time.Duration(api.Config.Expiry) * time.Second)
	}
	conversation, err = api.db.CreateConversation(initiator, participants, expiry)
	if err == nil {
		go api.cache.AddConversation(conversation)
		go api.NewConversationEvent(conversation)
	}
	return
}

//CreateRandomConversation generates a new conversation for user id witn nParticipants participants.
func (api *API) CreateRandomConversation(id gp.UserId, nParticipants int, live bool) (conversation gp.Conversation, err error) {
	networks, err := api.GetUserNetworks(id)
	if err != nil {
		return
	}
	participants, err := api.generatePartners(id, nParticipants-1, networks[0].Id)
	if err != nil {
		return
	}
	user, err := api.GetUser(id)
	if err != nil {
		return
	}
	participants = append(participants, user)
	return api.CreateConversation(id, participants, live)
}

//CreateConversationWith generates a new conversation with a particular group of participants.
func (api *API) CreateConversationWith(initiator gp.UserId, with []gp.UserId, live bool) (conversation gp.Conversation, err error) {
	var participants []gp.User
	user, err := api.GetUser(initiator)
	if err != nil {
		return
	}
	participants = append(participants, user)
	for _, id := range with {
		//TODO: Handle error
		canContact, _ := api.CanContact(initiator, id)
		if canContact {
			user, err = api.GetUser(id)
			if err != nil {
				return
			}
			participants = append(participants, user)
		} else {
			err = &ENOTALLOWED
			return
		}
	}
	return api.CreateConversation(initiator, participants, live)
}

//CanContact returns true if the initiator is allowed to contact the recipient.
//TODO: actually do something.
func (api *API) CanContact(initiator gp.UserId, recipient gp.UserId) (contactable bool, err error){
	return true, nil
}

func (api *API) NewConversationEvent(conversation gp.Conversation) {
	chans := ConversationChannelKeys(conversation.Participants)
	go api.cache.PublishEvent("new-conversation", ConversationURI(conversation.Id), conversation, chans)
}

func (api *API) EndConversationEvent(conversation gp.ConversationId) {
	conv, err := api.getConversation(conversation)
	if err != nil {
		log.Println(err)
		return
	}
	chans := ConversationChannelKeys(conv.Participants)
	go api.cache.PublishEvent("ended-conversation", ConversationURI(conversation), conv, chans)
}

func (api *API) AwaitOneMessage(userId gp.UserId) (resp []byte) {
	c := api.GetMessageChan(userId)
	select {
	case resp = <-c:
		return
	case <-time.After(60 * time.Second):
		return []byte("{}")
	}
}

func (api *API) GetMessageChan(userId gp.UserId) (c chan []byte) {
	return api.cache.MessageChan(userId)
}

//TODO: use conf.ConversationPageSize
func (api *API) addAllConversations(userId gp.UserId) (err error) {
	conversations, err := api.db.GetConversations(userId, 0, 2000)
	for _, conv := range conversations {
		go api.cache.AddConversation(conv.Conversation)
	}
	return
}

//GetConversation retrieves a particular conversation including up to ConversationPageSize most recent messages
//TODO: Restrict access to correct userId
//TODO: defer actual operation to getConversation
func (api *API) GetConversation(userId gp.UserId, convId gp.ConversationId) (conversation gp.ConversationAndMessages, err error) {
	return api.getConversation(convId)
}

func (api *API) getConversation(convId gp.ConversationId) (conversation gp.ConversationAndMessages, err error) {
	return api.db.GetConversation(convId, api.Config.ConversationPageSize)
}

func (api *API) GetMessage(msgId gp.MessageId) (message gp.Message, err error) {
	message, err = api.cache.GetMessage(msgId)
	return message, err
}

func (api *API) updateConversation(id gp.ConversationId) (err error) {
	err = api.db.UpdateConversation(id)
	if err != nil {
		return err
	}
	participants := api.db.GetParticipants(id)
	go api.cache.UpdateConversationLists(participants, id)
	return nil
}

//AddMessage creates a new message from userId in conversation convId.
func (api *API) AddMessage(convId gp.ConversationId, userId gp.UserId, text string) (messageId gp.MessageId, err error) {
	messageId, err = api.db.AddMessage(convId, userId, text)
	if err != nil {
		return
	}
	user, err := api.GetUser(userId)
	if err != nil {
		return
	}
	msg := gp.Message{gp.MessageId(messageId), user, text, time.Now().UTC()}
	participants := api.db.GetParticipants(convId)
	go api.cache.Publish(msg, participants, convId)
	chans := ConversationChannelKeys(participants)
	go api.cache.PublishEvent("message", ConversationURI(convId), msg, chans)
	go api.cache.AddMessage(msg, convId)
	go api.updateConversation(convId)
	go api.messagePush(msg, convId)
	return
}

func ConversationURI(convId gp.ConversationId) (uri string) {
	return fmt.Sprintf("/conversations/%d", convId)
}

func ConversationChannelKeys(participants []gp.User) (keys []string) {
	for _, u := range participants {
		keys = append(keys, fmt.Sprintf("c:%d", u.Id))
	}
	return keys
}

func (api *API) GetFullConversation(convId gp.ConversationId, start int64, count int) (conv gp.ConversationAndMessages, err error) {
	conv.Id = convId
	conv.LastActivity, err = api.ConversationLastActivity(convId)
	if err != nil {
		return
	}
	conv.Participants = api.GetParticipants(convId)
	conv.Read, err = api.readStatus(convId)
	if err != nil {
		return
	}
	conv.Messages, err = api.GetMessages(convId, start, "start", count)
	return
}

//readStatus returns the point all participants have read until in a conversation, omitting any participants who have read nothing.
//TODO: Use cache
func (api *API) readStatus(convId gp.ConversationId) (read []gp.Read, err error) {
	return api.db.GetReadStatus(convId)
}

func (api *API) ConversationLastActivity(convId gp.ConversationId) (t time.Time, err error) {
	return api.db.ConversationActivity(convId)
}

func (api *API) GetParticipants(convId gp.ConversationId) []gp.User {
	participants, err := api.cache.GetParticipants(convId)
	if err != nil {
		participants = api.db.GetParticipants(convId)
		go api.cache.SetConversationParticipants(convId, participants)
	}
	return participants
}

//todo: pass in message count
func (api *API) GetMessages(convId gp.ConversationId, index int64, sel string, count int) (messages []gp.Message, err error) {
	messages, err = api.cache.GetMessages(convId, index, sel, count)
	if err != nil {
		messages, err = api.db.GetMessages(convId, index, sel, count)
		go api.FillMessageCache(convId)
		return
	}
	return
}

func (api *API) FillMessageCache(convId gp.ConversationId) (err error) {
	messages, err := api.db.GetMessages(convId, 0, "start", api.Config.Redis.MessageCache)
	if err != nil {
		log.Println(err)
		return (err)
	}
	go api.cache.AddMessages(convId, messages)
	return
}

func (api *API) GetConversations(userId gp.UserId, start int64, count int) (conversations []gp.ConversationSmall, err error) {
	conversations, err = api.cache.GetConversations(userId, start, count)
	if err != nil {
		conversations, err = api.db.GetConversations(userId, start, count)
		go api.addAllConversations(userId)
	} else {
		//This is here because api.cache.GetConversations doesn't get the expiry itself...
		for i, c := range conversations {
			exp, err := api.Expiry(c.Id)
			if err == nil {
				conversations[i].Expiry = &exp
			}
		}
	}
	return
}

func (api *API) GetLastMessage(id gp.ConversationId) (message gp.Message, err error) {
	message, err = api.cache.GetLastMessage(id)
	if err != nil {
		message, err = api.db.GetLastMessage(id)
		go api.FillMessageCache(id)
		if err != nil {
			return
		}
	}
	return
}

func (api *API) Expiry(convId gp.ConversationId) (expiry gp.Expiry, err error) {
	expiry, err = api.cache.ConversationExpiry(convId)
	if err != nil {
		expiry, err = api.db.ConversationExpiry(convId)
		if err == nil {
			api.cache.SetConversationExpiry(convId, expiry)
		}
	}
	return
}

func (api *API) DeleteExpiry(convId gp.ConversationId) (err error) {
	err = api.db.DeleteConversationExpiry(convId)
	if err == nil {
		go api.cache.DelConversationExpiry(convId)
	}
	return
}

//UnExpireBetweenUsers should fetch all of users[0] conversations, find the ones which contain
//exactly the same participants as users and delete its expiry(if it exists).
func (api *API) UnExpireBetween(users []gp.UserId) (err error) {
	if len(users) < 2 {
		return gp.APIerror{">1 user required?"}
	}
	conversations, err := api.db.GetConversations(users[0], 0, 99999)
	if err != nil {
		return
	}
	for _, c := range conversations {
		n := 0
		if len(users) == len(c.Participants) {
			for _, p := range c.Participants {
				for _, u := range users {
					if u == p.Id {
						n++
					}
				}
			}
		}
		if n == len(users) {
			err = api.DeleteExpiry(c.Id)
			if err != nil {
				return
			}
		}
	}
	return
}
