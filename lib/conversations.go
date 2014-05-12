package lib

import (
	"fmt"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

var ENOTALLOWED = gp.APIerror{Reason: "You're not allowed to do that!"}

func (api *API) terminateConversation(convID gp.ConversationId) (err error) {
	log.Println("Terminating conversation:", convID)
	err = api.db.TerminateConversation(convID)
	if err == nil {
		go api.cache.TerminateConversation(convID)
		go api.EndConversationEvent(convID)
	}
	return
}

//UserEndConversation finishes a live conversation, or returns ENOTALLOWED if the user isn't allowed to.
func (api *API) UserEndConversation(userID gp.UserID, convID gp.ConversationId) (err error) {
	if api.UserCanViewConversation(userID, convID) {
		return api.terminateConversation(convID)
	}
	return &ENOTALLOWED
}

func (api *API) generatePartners(id gp.UserID, count int, network gp.NetworkId) (partners []gp.User, err error) {
	return api.db.RandomPartners(id, count, network)
}

//MarkConversationSeen sets the "read" location to upTo for user id in conversation convId.
func (api *API) MarkConversationSeen(id gp.UserID, convID gp.ConversationId, upTo gp.MessageId) (err error) {
	err = api.db.MarkRead(id, convID, upTo)
	if err != nil {
		return
	}
	api.cache.MarkConversationSeen(id, convID, upTo)
	return
}

func (api *API) CreateConversation(initiator gp.UserID, participants []gp.User, live bool) (conversation gp.Conversation, err error) {
	var expiry *gp.Expiry
	if live {
		expiry = gp.NewExpiry(time.Duration(api.Config.Expiry) * time.Second)
	}
	conversation, err = api.db.CreateConversation(initiator, participants, expiry)
	if err == nil {
		go api.cache.AddConversation(conversation)
		go api.NewConversationEvent(conversation)
		initiator, err := api.GetUser(initiator)
		if err == nil {
			for _, u := range participants {
				go api.newConversationPush(initiator, u.Id, conversation.Id)
			}
		} else {
			log.Println("Problem getting user:", err)
		}
	}
	return
}

//CreateRandomConversation generates a new conversation for user id witn nParticipants participants.
func (api *API) CreateRandomConversation(id gp.UserID, nParticipants int, live bool) (conversation gp.Conversation, err error) {
	log.Println("Terminating old conversations")
	conversations, err := api.db.ConversationsToTerminate(id)
	if err == nil {
		for _, c := range conversations {
			e := api.terminateConversation(c)
			if e != nil {
				log.Println(e)
			}
		}
	}
	log.Println("Creating a random conversation")
	log.Println("Getting networks")
	networks, err := api.GetUserNetworks(id)
	if err != nil {
		return
	}
	log.Println("Getting partner(s)")
	participants, err := api.generatePartners(id, nParticipants-1, networks[0].Id)
	if err != nil {
		log.Println("Errored getting partners...", err)
		return
	}
	log.Println("Getting myself")
	user, err := api.GetUser(id)
	if err != nil {
		return
	}
	log.Println("Terminating participant's excess conversations")
	for _, u := range participants {
		conversations, err = api.db.ConversationsToTerminate(u.Id)
		if err == nil {
			for _, c := range conversations {
				e := api.terminateConversation(c)
				if e != nil {
					log.Println(e)
				}
			}
		}

	}
	participants = append(participants, user)
	log.Println("Creating a conversation")
	return api.CreateConversation(id, participants, live)
}

//CreateConversationWith generates a new conversation with a particular group of participants.
func (api *API) CreateConversationWith(initiator gp.UserID, with []gp.UserID, live bool) (conversation gp.Conversation, err error) {
	var participants []gp.User
	user, err := api.GetUser(initiator)
	if err != nil {
		return
	}
	participants = append(participants, user)
	for _, id := range with {
		canContact, e := api.CanContact(initiator, id)
		if e != nil {
			log.Println("Error determining contactability:", initiator, id, e)
			return conversation, e
		}
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
//This means that a) they are contacts already or b) recipient has posted somewhere a) can see.
//TODO: Include comments / attends too.
func (api *API) CanContact(initiator gp.UserID, recipient gp.UserID) (contactable bool, err error) {
	contacts, err := api.AreContacts(initiator, recipient)
	if err != nil {
		return
	}
	if !contacts {
		shared, e := api.HaveSharedNetwork(initiator, recipient)
		switch {
		case e != nil:
			return false, e
		case !shared:
			return false, nil
		default:
			posted, err := api.UserHasPosted(recipient, initiator)
			if err != nil {
				return false, err
			}
			return posted, nil
		}
	} else {
		return true, nil
	}
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

func (api *API) ConversationChangedEvent(conversation gp.Conversation) {
	chans := ConversationChannelKeys(conversation.Participants)
	go api.cache.PublishEvent("changed-conversation", ConversationURI(conversation.Id), conversation, chans)
}

func (api *API) AwaitOneMessage(userID gp.UserID) (resp []byte) {
	c := api.GetMessageChan(userID)
	select {
	case resp = <-c:
		return
	case <-time.After(60 * time.Second):
		return []byte("{}")
	}
}

func (api *API) GetMessageChan(userID gp.UserID) (c chan []byte) {
	return api.cache.MessageChan(userID)
}

//TODO: use conf.ConversationPageSize
func (api *API) addAllConversations(userID gp.UserID) (err error) {
	conversations, err := api.db.GetConversations(userID, 0, 2000, false)
	for _, conv := range conversations {
		go api.cache.AddConversation(conv.Conversation)
	}
	return
}

//GetConversation retrieves a particular conversation including up to ConversationPageSize most recent messages
//TODO: Restrict access to correct userId
func (api *API) GetConversation(userID gp.UserID, convID gp.ConversationId) (conversation gp.ConversationAndMessages, err error) {
	if api.UserCanViewConversation(userID, convID) {
		return api.getConversation(convID)
	}
	return conversation, &ENOTALLOWED
}

func (api *API) getConversation(convID gp.ConversationId) (conversation gp.ConversationAndMessages, err error) {
	return api.db.GetConversation(convID, api.Config.ConversationPageSize)
}

func (api *API) GetMessage(msgID gp.MessageId) (message gp.Message, err error) {
	message, err = api.cache.GetMessage(msgID)
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

//AddMessage creates a new message from userId in conversation convId, or returns ENOTALLOWED if the user is not a participant.
func (api *API) AddMessage(convID gp.ConversationId, userID gp.UserID, text string) (messageID gp.MessageId, err error) {
	if !api.UserCanViewConversation(userID, convID) {
		return messageID, &ENOTALLOWED
	}
	messageID, err = api.db.AddMessage(convID, userID, text)
	if err != nil {
		return
	}
	user, err := api.GetUser(userID)
	if err != nil {
		return
	}
	msg := gp.Message{
		Id:   gp.MessageId(messageID),
		By:   user,
		Text: text,
		Time: time.Now().UTC()}
	participants := api.db.GetParticipants(convID)
	go api.cache.Publish(msg, participants, convID)
	chans := ConversationChannelKeys(participants)
	go api.cache.PublishEvent("message", ConversationURI(convID), msg, chans)
	go api.cache.AddMessage(msg, convID)
	go api.updateConversation(convID)
	go api.messagePush(msg, convID)
	return
}

func ConversationURI(convID gp.ConversationId) (uri string) {
	return fmt.Sprintf("/conversations/%d", convID)
}

func ConversationChannelKeys(participants []gp.User) (keys []string) {
	for _, u := range participants {
		keys = append(keys, fmt.Sprintf("c:%d", u.Id))
	}
	return keys
}

func (api *API) UserCanViewConversation(userID gp.UserID, convID gp.ConversationId) (viewable bool) {
	participants := api.GetParticipants(convID)
	for _, u := range participants {
		if userID == u.Id {
			return true
		}
	}
	return false
}

//UserGetConversation returns the conversation convId if userId is allowed to view it; otherwise returns ENOTALLOWED.
func (api *API) UserGetConversation(userID gp.UserID, convID gp.ConversationId, start int64, count int) (conv gp.ConversationAndMessages, err error) {
	if api.UserCanViewConversation(userID, convID) {
		return api.GetFullConversation(convID, start, count)
	}
	return conv, &ENOTALLOWED
}

func (api *API) GetFullConversation(convID gp.ConversationId, start int64, count int) (conv gp.ConversationAndMessages, err error) {
	conv.Id = convID
	conv.LastActivity, err = api.ConversationLastActivity(convID)
	if err != nil {
		return
	}
	conv.Participants = api.GetParticipants(convID)
	conv.Read, err = api.readStatus(convID)
	if err != nil {
		return
	}
	conv.Messages, err = api.getMessages(convID, start, "start", count)
	return
}

//readStatus returns the point all participants have read until in a conversation, omitting any participants who have read nothing.
//TODO: Use cache
func (api *API) readStatus(convID gp.ConversationId) (read []gp.Read, err error) {
	return api.db.GetReadStatus(convID)
}

func (api *API) ConversationLastActivity(convID gp.ConversationId) (t time.Time, err error) {
	return api.db.ConversationActivity(convID)
}

func (api *API) GetParticipants(convID gp.ConversationId) []gp.User {
	participants, err := api.cache.GetParticipants(convID)
	if err != nil {
		participants = api.db.GetParticipants(convID)
		go api.cache.SetConversationParticipants(convID, participants)
	}
	return participants
}

//UserGetMessages returns count messages from the conversation convId, or ENOTALLOWED if the user is not allowed to view this conversation.
//sel may be one of:
//start (returns messages starting from the index'th)
//before (returns messages historically earlier than the one with id index)
//after (returns messages newer than index)
func (api *API) UserGetMessages(userID gp.UserID, convID gp.ConversationId, index int64, sel string, count int) (messages []gp.Message, err error) {
	if api.UserCanViewConversation(userID, convID) {
		return api.getMessages(convID, index, sel, count)
	}
	return messages, &ENOTALLOWED
}

func (api *API) getMessages(convID gp.ConversationId, index int64, sel string, count int) (messages []gp.Message, err error) {
	messages, err = api.cache.GetMessages(convID, index, sel, count)
	if err != nil {
		messages, err = api.db.GetMessages(convID, index, sel, count)
		go api.FillMessageCache(convID)
		return
	}
	return
}

func (api *API) FillMessageCache(convID gp.ConversationId) (err error) {
	messages, err := api.db.GetMessages(convID, 0, "start", api.Config.Redis.MessageCache)
	if err != nil {
		log.Println(err)
		return (err)
	}
	go api.cache.AddMessages(convID, messages)
	return
}

//GetConversations returns count non-ended conversations which userId participates in, starting from start and ordered by their last activity.
func (api *API) GetConversations(userID gp.UserID, start int64, count int) (conversations []gp.ConversationSmall, err error) {
	conversations, err = api.db.GetConversations(userID, start, count, false)
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

func (api *API) Expiry(convID gp.ConversationId) (expiry gp.Expiry, err error) {
	expiry, err = api.cache.ConversationExpiry(convID)
	if err != nil {
		expiry, err = api.db.ConversationExpiry(convID)
		if err == nil {
			api.cache.SetConversationExpiry(convID, expiry)
		}
	}
	return
}

//UserDeleteExpiry converts a conversation from live to regular.
//If the user isn't allowed to do this, it returns ENOTALLOWED.
func (api *API) UserDeleteExpiry(userID gp.UserID, convID gp.ConversationId) (err error) {
	if api.UserCanViewConversation(userID, convID) {
		return api.deleteExpiry(convID)
	}
	return &ENOTALLOWED
}

func (api *API) deleteExpiry(convID gp.ConversationId) (err error) {
	err = api.db.DeleteConversationExpiry(convID)
	if err == nil {
		go api.cache.DelConversationExpiry(convID)
	}
	return
}

//UnExpireBetweenUsers should fetch all of users[0] conversations, find the ones which contain
//exactly the same participants as users and delete its expiry(if it exists).
func (api *API) UnExpireBetween(users []gp.UserID) (err error) {
	if len(users) < 2 {
		return gp.APIerror{Reason: ">1 user required?"}
	}
	conversations, err := api.db.GetConversations(users[0], 0, 99999, true)
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
			err = api.deleteExpiry(c.Id)
			if err != nil {
				return
			}
			c.Expiry = nil
			go api.ConversationChangedEvent(c.Conversation)
		}
	}
	return
}

//MarkAllConversationsSeen sets "read" = LastMessage for all user's conversations.
func (api *API) MarkAllConversationsSeen(user gp.UserID) (err error) {
	conversations, err := api.db.GetConversations(user, 0, 10000, true)
	if err != nil {
		return
	}
	for _, c := range conversations {
		log.Println("Got conversation", c.Id)
		if c.LastMessage != nil {
			log.Printf("Marking conversation %d seen up to %d for user %d\n", c.Id, c.LastMessage.Id, user)
			err = api.MarkConversationSeen(user, c.Id, c.LastMessage.Id)
			if err != nil {
				return
			}
		}
	}
	return
}

func (api *API) UnreadMessageCount(user gp.UserID) (count int, err error) {
	return api.db.UnreadMessageCount(user)
}

func (api *API) TotalLiveConversations(user gp.UserID) (count int, err error) {
	return api.db.TotalLiveConversations(user)
}

func (api *API) EndOldConversations() {
	t := time.Tick(time.Duration(30) * time.Second)
	for {
		select {
		case <-t:
			convs, err := api.db.PrunableConversations()
			if err != nil {
				log.Println("Prune error:", err)
			}
			for _, c := range convs {
				go api.terminateConversation(c)
			}
		}
	}
}
