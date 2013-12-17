package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"time"
	"fmt"
	"log"
)

func (api *API)TerminateConversation(convId gp.ConversationId) (err error) {
	err = api.db.TerminateConversation(convId)
	if err == nil {
		go api.cache.TerminateConversation(convId)
	}
	return
}

func (api *API)generatePartners(id gp.UserId, count int, network gp.NetworkId) (partners []gp.User, err error) {
	return api.db.RandomPartners(id, count, network)
}

func (api *API)MarkConversationSeen(id gp.UserId, convId gp.ConversationId, upTo gp.MessageId) (conversation gp.ConversationAndMessages, err error) {
	err = api.db.MarkRead(id, convId, upTo)
	if err != nil {
		return
	}
	err = api.cache.MarkConversationSeen(id, convId, upTo)
	if err != nil {
		go api.FillMessageCache(convId)
	}
	conversation, err = api.db.GetConversation(convId)
	return
}

func (api *API)CreateConversation(id gp.UserId, nParticipants int, live bool) (conversation gp.Conversation, err error) {
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
	conversation, err = api.db.CreateConversation(id, participants, live)
	if err == nil {
		go api.cache.AddConversation(conversation)
		go api.NewConversationEvent(conversation)
	}
	return
}

func (api *API)NewConversationEvent(conversation gp.Conversation) {
		chans := ConversationChannelKeys(conversation.Participants)
		go api.cache.PublishEvent("new-conversation", ConversationURI(conversation.Id), conversation, chans)
}

func (api *API)AwaitOneMessage(userId gp.UserId) (resp []byte) {
	c := api.GetMessageChan(userId)
	select {
	case resp = <-c:
		return
	case <-time.After(60 * time.Second):
		return []byte("{}")
	}
}

func (api *API)GetMessageChan(userId gp.UserId) (c chan []byte) {
	return api.cache.MessageChan(userId)
}

//TODO: pass in count from outside
func (api *API)addAllConversations(userId gp.UserId) (err error) {
	conf := gp.GetConfig()
	conversations, err := api.db.GetConversations(userId, 0, conf.ConversationPageSize)
	for _, conv := range conversations {
		go api.cache.AddConversation(conv.Conversation)
	}
	return
}

func (api *API)GetConversation(userId gp.UserId, convId gp.ConversationId) (conversation gp.ConversationAndMessages, err error) {
	//cache.GetConversation
	return api.db.GetConversation(convId)
}

func (api *API)GetMessage(msgId gp.MessageId) (message gp.Message, err error) {
	message, err = api.cache.GetMessage(msgId)
	return message, err
}

func (api *API)updateConversation(id gp.ConversationId) (err error) {
	err = api.db.UpdateConversation(id)
	if err != nil {
		return err
	}
	participants := api.db.GetParticipants(id)
	go api.cache.UpdateConversationLists(participants, id)
	return nil
}

func (api *API)AddMessage(convId gp.ConversationId, userId gp.UserId, text string) (messageId gp.MessageId, err error) {
	messageId, err = api.db.AddMessage(convId, userId, text)
	if err != nil {
		return
	}
	user, err := api.GetUser(userId)
	if err != nil {
		return
	}
	msg := gp.Message{gp.MessageId(messageId), user, text, time.Now().UTC(), false}
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

func (api *API)GetFullConversation(convId gp.ConversationId, start int64) (conv gp.ConversationAndMessages, err error) {
	conv.Id = convId
	conv.LastActivity, err = api.ConversationLastActivity(convId)
	if err != nil {
		return
	}
	conv.Participants = api.GetParticipants(convId)
	conv.Messages, err = api.GetMessages(convId, start, "start")
	return
}

func (api *API)ConversationLastActivity(convId gp.ConversationId) (t time.Time, err error) {
	return api.db.ConversationActivity(convId)
}

func (api *API)GetParticipants(convId gp.ConversationId) []gp.User {
	participants, err := api.cache.GetParticipants(convId)
	if err != nil {
		participants = api.db.GetParticipants(convId)
		go api.cache.SetConversationParticipants(convId, participants)
	}
	return participants
}

//todo: pass in message count
func (api *API)GetMessages(convId gp.ConversationId, index int64, sel string) (messages []gp.Message, err error) {
	conf := gp.GetConfig()
	messages, err = api.cache.GetMessages(convId, index, sel, conf.MessagePageSize)
	if err != nil {
		messages, err = api.db.GetMessages(convId, index, sel, conf.MessagePageSize)
		go api.FillMessageCache(convId)
		return
	}
	return
}

func (api *API)FillMessageCache(convId gp.ConversationId) (err error) {
	conf := gp.GetConfig()
	messages, err := api.db.GetMessages(convId, 0, "start", conf.MessageCache)
	if err != nil {
		log.Println(err)
		return(err)
	}
	go api.cache.AddMessages(convId, messages)
	return
}

func (api *API)GetConversations(userId gp.UserId, start int64, count int) (conversations []gp.ConversationSmall, err error) {
	conversations, err = api.cache.GetConversations(userId, start, count)
	if err != nil {
		conversations, err = api.db.GetConversations(userId, start, count)
		go api.addAllConversations(userId)
	} else {
		//This is here because api.cache.GetConversations doesn't get the expiry itself...
		for i, c := range(conversations) {
			exp, err := api.Expiry(c.Id)
			if err == nil {
				conversations[i].Expiry = &exp
			}
		}
	}
	return
}

func (api *API)GetLastMessage(id gp.ConversationId) (message gp.Message, err error) {
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

func (api *API)Expiry(convId gp.ConversationId) (expiry gp.Expiry, err error) {
	expiry, err = api.cache.ConversationExpiry(convId)
	if err != nil {
		expiry, err = api.db.ConversationExpiry(convId)
		if err == nil {
			api.cache.SetConversationExpiry(convId, expiry)
		}
	}
	return
}

func (api *API)DeleteExpiry(convId gp.ConversationId) (err error) {
	err = api.db.DeleteConversationExpiry(convId)
	if err == nil {
		go api.cache.DelConversationExpiry(convId)
	}
	return
}

//UnExpireBetweenUsers should fetch all of users[0] conversations, find the ones which contain
//exactly the same participants as users and delete its expiry(if it exists).
func (api *API)UnExpireBetween(users []gp.UserId) (err error) {
	if len(users) < 2 {
		return gp.APIerror{">1 user required?"}
	}
	conversations, err := api.db.GetConversations(users[0], 0, 99999)
	if err != nil {
		return
	}
	for _, c := range(conversations) {
		n := 0
		if len(users) == len(c.Participants) {
			for _, p := range(c.Participants) {
				for _, u := range(users) {
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

