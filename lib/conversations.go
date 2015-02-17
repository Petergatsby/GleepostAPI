package lib

import (
	"fmt"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//ENOTALLOWED is returned when a user attempts an action that they shouldn't.
var ENOTALLOWED = gp.APIerror{Reason: "You're not allowed to do that!"}

//UserDeleteConversation removes this conversation from the list; it also terminates it (if it's a live conversation).
func (api *API) UserDeleteConversation(userID gp.UserID, convID gp.ConversationID) (err error) {
	if api.UserCanViewConversation(userID, convID) {
		err = api.db.DeleteConversation(userID, convID)
		if err != nil {
			return
		}
		go api.addSystemMessage(convID, userID, "PARTED")
	}
	return &ENOTALLOWED
}

//MarkConversationSeen sets the "read" location to upTo for user id in conversation convId.
func (api *API) MarkConversationSeen(id gp.UserID, convID gp.ConversationID, upTo gp.MessageID) (err error) {
	err = api.db.MarkRead(id, convID, upTo)
	if err != nil {
		return
	}
	read := gp.Read{UserID: id, LastRead: upTo}
	conv, err := api.getConversation(id, convID)
	if err != nil {
		log.Println(err)
		return
	}
	chans := ConversationChannelKeys(conv.Participants)
	go api.cache.PublishEvent("read", conversationURI(convID), read, chans)
	return
}

//CreateConversation generates a new conversation involving initiator and participants. If primary is true, it is the only permitted conversation between this set of participants. If group != 0, this is the conversation for that network.
func (api *API) CreateConversation(initiator gp.UserID, participants []gp.User, primary bool, group gp.NetworkID) (conversation gp.Conversation, err error) {
	conversation, err = api.db.CreateConversation(initiator, participants, primary, group)
	if err == nil {
		go api.newConversationEvent(conversation)
	}
	return
}

//CreateConversationWith generates a new conversation with a particular group of participants. If reuse is true, it will return the existing "primary" conversation with those users, creating one only if necessary.
func (api *API) CreateConversationWith(initiator gp.UserID, reuse bool, with []gp.UserID) (conversation gp.ConversationAndMessages, err error) {
	var participants []gp.User
	user, err := api.getUser(initiator)
	if err != nil {
		return
	}
	participants = append(participants, user)
	if reuse && len(with) == 1 {
		primaryConversation, err := api.getPrimaryConversation(initiator, with[0])
		if err == nil {
			return primaryConversation, nil
		}
	}
	for _, id := range with {
		canContact, e := api.haveSharedNetwork(initiator, id)
		if e != nil {
			log.Println("Error determining contactability:", initiator, id, e)
			return conversation, e
		}
		if canContact {
			user, err = api.getUser(id)
			if err != nil {
				return
			}
			participants = append(participants, user)
		} else {
			err = &ENOTALLOWED
			return
		}
	}
	conv, err := api.CreateConversation(initiator, participants, len(with) == 1, 0)
	if err != nil {
		return
	}
	conversation = gp.ConversationAndMessages{Conversation: conv}
	return
}

func (api *API) getPrimaryConversation(participantA, participantB gp.UserID) (conversation gp.ConversationAndMessages, err error) {
	return api.db.GetPrimaryConversation(participantA, participantB)

}

//CanContact returns true if the initiator is allowed to contact the recipient.
func (api *API) canContact(initiator gp.UserID, recipient gp.UserID) (contactable bool, err error) {
	shared, e := api.haveSharedNetwork(initiator, recipient)
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
}

//NewConversationEvent publishes an event to all listening participants to let them know they have a new conversation.
func (api *API) newConversationEvent(conversation gp.Conversation) {
	chans := ConversationChannelKeys(conversation.Participants)
	go api.cache.PublishEvent("new-conversation", conversationURI(conversation.ID), conversation, chans)
}

//EndConversationEvent publishes an event to all listening participants to let them know the conversation is terminated.
func (api *API) endConversationEvent(conversation gp.ConversationID) {
	conv, err := api.getConversation(0, conversation) //0 means we will omit the unread count.
	if err != nil {
		log.Println(err)
		return
	}
	chans := ConversationChannelKeys(conv.Participants)
	go api.cache.PublishEvent("ended-conversation", conversationURI(conversation), conv, chans)
}

//ConversationChangedEvent publishes an event to all listening participants that this conversation has changed in some way, typically because its expiry has been removed.
func (api *API) conversationChangedEvent(conversation gp.Conversation) {
	chans := ConversationChannelKeys(conversation.Participants)
	go api.cache.PublishEvent("changed-conversation", conversationURI(conversation.ID), conversation, chans)
}

//GetConversation retrieves a particular conversation including up to ConversationPageSize most recent messages
func (api *API) GetConversation(userID gp.UserID, convID gp.ConversationID) (conversation gp.ConversationAndMessages, err error) {
	if api.UserCanViewConversation(userID, convID) {
		return api.getConversation(userID, convID)
	}
	return conversation, &ENOTALLOWED
}

func (api *API) getConversation(userID gp.UserID, convID gp.ConversationID) (conversation gp.ConversationAndMessages, err error) {
	return api.db.GetConversation(userID, convID, api.Config.ConversationPageSize)
}

func (api *API) updateConversation(id gp.ConversationID) (err error) {
	err = api.db.UpdateConversation(id)
	if err != nil {
		return err
	}
	return nil
}

//AddMessage creates a new message from userId in conversation convId, or returns ENOTALLOWED if the user is not a participant.
func (api *API) AddMessage(convID gp.ConversationID, userID gp.UserID, text string) (messageID gp.MessageID, err error) {
	if !api.UserCanViewConversation(userID, convID) {
		return messageID, &ENOTALLOWED
	}
	messageID, err = api.db.AddMessage(convID, userID, text, false)
	if err != nil {
		return
	}
	user, err := api.getUser(userID)
	if err != nil {
		return
	}
	msg := gp.Message{
		ID:   gp.MessageID(messageID),
		By:   user,
		Text: text,
		Time: time.Now().UTC()}
	participants, err := api.db.GetParticipants(convID, false)
	if err == nil {
		//Note to self: What is the difference between Publish and PublishEvent?
		go api.cache.Publish(msg, participants, convID)
		chans := ConversationChannelKeys(participants)
		go api.cache.PublishEvent("message", conversationURI(convID), msg, chans)
	} else {
		log.Println("Error getting participants; didn't bradcast event to websockets")
	}
	go api.updateConversation(convID)
	go api.messagePush(msg, convID)
	return
}

//conversationURI returns the URI of this conversation relative to the API root.
func conversationURI(convID gp.ConversationID) (uri string) {
	return fmt.Sprintf("/conversations/%d", convID)
}

//ConversationChannelKeys returns all of the message channel keys for these users (typically used to publish messages to all participants of a conversation)
func ConversationChannelKeys(participants []gp.User) (keys []string) {
	for _, u := range participants {
		keys = append(keys, fmt.Sprintf("c:%d", u.ID))
	}
	return keys
}

//UserCanViewConversation returns true if userID is a participant of convID
func (api *API) UserCanViewConversation(userID gp.UserID, convID gp.ConversationID) (viewable bool) {
	participants := api.getParticipants(convID, false)
	for _, u := range participants {
		if userID == u.ID {
			return true
		}
	}
	return false
}

//UserGetConversation returns the conversation convId if userId is allowed to view it; otherwise returns ENOTALLOWED.
func (api *API) UserGetConversation(userID gp.UserID, convID gp.ConversationID, start int64, count int) (conv gp.ConversationAndMessages, err error) {
	if api.UserCanViewConversation(userID, convID) {
		return api.GetFullConversation(convID, start, count)
	}
	return conv, &ENOTALLOWED
}

//GetFullConversation returns a full conversation containing up to count messages.
//TODO(patrick) - clarify this vs getConversation etc
func (api *API) GetFullConversation(convID gp.ConversationID, start int64, count int) (conv gp.ConversationAndMessages, err error) {
	conv.ID = convID
	conv.LastActivity, err = api.conversationLastActivity(convID)
	if err != nil {
		return
	}
	conv.Participants = api.getParticipants(convID, true)
	conv.Read, err = api.readStatus(convID)
	if err != nil {
		return
	}
	conv.Group, err = api.db.ConversationGroup(convID)
	if err != nil {
		log.Println(err)
	}
	conv.Messages, err = api.getMessages(convID, start, "start", count)
	return
}

//readStatus returns the point all participants have read until in a conversation, omitting any participants who have read nothing.
//TODO: Use cache
func (api *API) readStatus(convID gp.ConversationID) (read []gp.Read, err error) {
	return api.db.GetReadStatus(convID)
}

//ConversationLastActivity returns the modification time (ie, creation  or last-message) for this conversation.
func (api *API) conversationLastActivity(convID gp.ConversationID) (t time.Time, err error) {
	return api.db.ConversationActivity(convID)
}

//GetParticipants returns all participants of this conversation, or omits the `deleted` participants if includeDeleted is false.
func (api *API) getParticipants(convID gp.ConversationID, includeDeleted bool) []gp.User {
	participants, err := api.db.GetParticipants(convID, includeDeleted)
	if err != nil {
		log.Println(err)
	}
	return participants
}

//UserGetMessages returns count messages from the conversation convId, or ENOTALLOWED if the user is not allowed to view this conversation.
//sel may be one of:
//start (returns messages starting from the index'th)
//before (returns messages historically earlier than the one with id index)
//after (returns messages newer than index)
func (api *API) UserGetMessages(userID gp.UserID, convID gp.ConversationID, index int64, sel string, count int) (messages []gp.Message, err error) {
	messages = make([]gp.Message, 0)
	if api.UserCanViewConversation(userID, convID) {
		return api.getMessages(convID, index, sel, count)
	}
	return messages, &ENOTALLOWED
}

func (api *API) getMessages(convID gp.ConversationID, index int64, sel string, count int) (messages []gp.Message, err error) {
	messages, err = api.db.GetMessages(convID, index, sel, count)
	return
}

//GetConversations returns count non-ended conversations which userId participates in, starting from start and ordered by their last activity.
func (api *API) GetConversations(userID gp.UserID, start int64, count int) (conversations []gp.ConversationSmall, err error) {
	conversations, err = api.db.GetConversations(userID, start, count)
	return
}

//GetLastMessage returns the most recent message in this conversation.
//this function doesn't appear to be used
func (api *API) getLastMessage(id gp.ConversationID) (message gp.Message, err error) {
	return api.db.GetLastMessage(id)
}

//MarkAllConversationsSeen sets "read" = LastMessage for all user's conversations.
func (api *API) MarkAllConversationsSeen(user gp.UserID) (err error) {
	conversations, err := api.db.GetConversations(user, 0, 10000)
	if err != nil {
		return
	}
	for _, c := range conversations {
		log.Println("Got conversation", c.ID)
		if c.LastMessage != nil {
			log.Printf("Marking conversation %d seen up to %d for user %d\n", c.ID, c.LastMessage.ID, user)
			err = api.MarkConversationSeen(user, c.ID, c.LastMessage.ID)
			if err != nil {
				return
			}
		}
	}
	return
}

//UnreadMessageCount returns the number of messages this user hasn't seen yet across all his active conversations, optionally ignoring ones before the user's configured threshold time.
func (api *API) UnreadMessageCount(user gp.UserID, useThreshold bool) (count int, err error) {
	return api.db.UnreadMessageCount(user, useThreshold)
}

//UserMuteBadges marks the user as having seen the badge for conversations before t; this means any unread messages before t will no longer be included in any badge values.
func (api *API) UserMuteBadges(userID gp.UserID, t time.Time) (err error) {
	return api.db.UserMuteBadges(userID, t)
}

//UserAddParticipants adds new user(s) to this conversation, iff userID is in conversation && userID and participants share at least one network (ie, university)
func (api *API) UserAddParticipants(userID gp.UserID, convID gp.ConversationID, participants ...gp.UserID) (updatedParticipants []gp.User, err error) {
	updatedParticipants = make([]gp.User, 0)
	if !api.UserCanViewConversation(userID, convID) {
		err = &ENOTALLOWED
		return
	}
	addable, err := api.addableParticipants(userID, convID, participants...)
	if err != nil {
		return
	}
	err = api.db.AddConversationParticipants(userID, addable, convID)
	if err != nil {
		return
	}
	conv, err := api.GetConversation(userID, convID)
	if err != nil {
		return
	}
	go api.conversationChangedEvent(conv.Conversation)
	for _, p := range addable {
		api.addSystemMessage(convID, p, "JOINED")
	}
	updatedParticipants = api.getParticipants(convID, false)
	return
}

//addableParticipants returns all the participants who can be added to this conversation -- ie, purges those with no shared networks and those already in the conv.
func (api *API) addableParticipants(userID gp.UserID, convID gp.ConversationID, participants ...gp.UserID) (addableParticipants []gp.UserID, err error) {
	for _, p := range participants {
		shared, err := api.haveSharedNetwork(userID, p) //Not someone who you can see
		if !shared || err != nil {
			continue
		}
		if api.UserCanViewConversation(p, convID) { //Already in conversation
			continue
		}
		addableParticipants = append(addableParticipants, p)
	}
	return
}

func (api *API) addSystemMessage(convID gp.ConversationID, userID gp.UserID, text string) (messageID gp.MessageID, err error) {
	messageID, err = api.db.AddMessage(convID, userID, text, true)
	if err != nil {
		return
	}
	user, err := api.getUser(userID)
	if err != nil {
		return
	}
	msg := gp.Message{
		ID:     gp.MessageID(messageID),
		By:     user,
		Text:   text,
		Time:   time.Now().UTC(),
		System: true}
	participants, err := api.db.GetParticipants(convID, false)
	if err == nil {
		//Note to self: What is the difference between Publish and PublishEvent?
		go api.cache.Publish(msg, participants, convID)
		chans := ConversationChannelKeys(participants)
		go api.cache.PublishEvent("message", conversationURI(convID), msg, chans)
	} else {
		log.Println("Error getting participants; didn't bradcast event to websockets")
	}
	go api.updateConversation(convID)
	return
}

//ConversationMergedInto returns the id of the conversation this one has merged with, or err if it hasn't merged.
func (api *API) ConversationMergedInto(convID gp.ConversationID) (mergedInto gp.ConversationID, err error) {
	return api.db.ConversationMergedInto(convID)
}
