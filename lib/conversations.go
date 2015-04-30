package lib

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//ENOTALLOWED is returned when a user attempts an action that they shouldn't.
var ENOTALLOWED = gp.APIerror{Reason: "You're not allowed to do that!", StatusCode: 403}

//ETOOFEW = You tried to create a conversation with 0 other participants (or you gave all invalid participants)
var ETOOFEW = gp.APIerror{Reason: "Must have at least one valid recipient."}

//ETOOMANY = You tried to create a conversation with a whole bunch of participants
var ETOOMANY = gp.APIerror{Reason: "Cannot send a message to more than 10 recipients"}

//UserDeleteConversation removes this conversation from the list; it also terminates it (if it's a live conversation).
func (api *API) UserDeleteConversation(userID gp.UserID, convID gp.ConversationID) (err error) {
	if api.UserCanViewConversation(userID, convID) {
		var group gp.NetworkID
		group, err = api.conversationGroup(convID)
		if group > 0 && err == nil {
			return &ENOTALLOWED
		}
		var primary bool
		primary, err = api.isPrimaryConversation(convID)
		if err == nil && primary {
			err = api.setDeletionThreshold(userID, convID, 999999999999)
			return
		}

		err = api.deleteConversation(userID, convID)
		if err != nil {
			return
		}
		go api.addSystemMessage(convID, userID, "PARTED")
		return
	}
	return &ENOTALLOWED
}

//MarkConversationSeen sets the "read" location to upTo for user id in conversation convId.
func (api *API) MarkConversationSeen(id gp.UserID, convID gp.ConversationID, upTo gp.MessageID) (err error) {
	read, err := api.getReadStatus(convID, false)
	if err != nil {
		return
	}
	for _, r := range read {
		if r.UserID == id {
			if r.LastRead < upTo {
				var actuallyUpto gp.MessageID
				actuallyUpto, err = api.markRead(id, convID, upTo)
				if err != nil {
					return
				}
				read := gp.Read{UserID: id, LastRead: actuallyUpto}
				conv, e := api.getConversation(id, convID, api.Config.MessagePageSize)
				if err != nil {
					log.Println(e)
					return e
				}
				chans := ConversationChannelKeys(conv.Participants)
				go api.cache.PublishEvent("read", conversationURI(convID), read, chans)
			}
			return
		}
	}
	return ENOTALLOWED
}

//CreateConversation generates a new conversation involving initiator and participants. If primary is true, it is the only permitted conversation between this set of participants. If group != 0, this is the conversation for that network.
func (api *API) createConversation(initiator gp.UserID, participants []gp.User, primary bool, group gp.NetworkID) (conversation gp.Conversation, err error) {
	conversation, err = api._createConversation(initiator, participants, primary, group)
	if err == nil {
		go api.newConversationEvent(conversation)
	}
	return
}

//CreateConversationWith generates a new conversation with a particular group of participants. If reuse is true, it will return the existing "primary" conversation with those users, creating one only if necessary.
func (api *API) CreateConversationWith(initiator gp.UserID, with []gp.UserID) (conversation gp.ConversationAndMessages, err error) {
	reuse := false
	switch {
	case len(with) > 50:
		err = ETOOMANY
		return
	case len(with) < 1:
		err = ETOOFEW
		return
	case len(with) == 1:
		reuse = true
	}
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
		canContact, e := api.sameUniversity(initiator, id)
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
	conv, err := api.createConversation(initiator, participants, len(with) == 1, 0)
	if err != nil {
		return
	}
	conversation = gp.ConversationAndMessages{Conversation: conv}
	return
}

//CanContact returns true if the initiator is allowed to contact the recipient.
func (api *API) canContact(initiator gp.UserID, recipient gp.UserID) (contactable bool, err error) {
	shared, e := api.sameUniversity(initiator, recipient)
	switch {
	case e != nil:
		return false, e
	case !shared:
		return false, nil
	default:
		posted, err := api.userHasPosted(recipient, initiator)
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
	conv, err := api.getConversation(0, conversation, api.Config.MessagePageSize) //0 means we will omit the unread count.
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
		return api.getConversation(userID, convID, api.Config.MessagePageSize)
	}
	return conversation, &ENOTALLOWED
}

//AddMessage creates a new message from userId in conversation convId, or returns ENOTALLOWED if the user is not a participant.
func (api *API) AddMessage(convID gp.ConversationID, userID gp.UserID, text string) (messageID gp.MessageID, err error) {
	if !api.UserCanViewConversation(userID, convID) {
		return messageID, &ENOTALLOWED
	}
	messageID, err = api.addMessage(convID, userID, text, false)
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
	group, err := api.conversationGroup(convID)
	if group > 0 && err == nil {
		msg.Group = group
	}
	participants, err := api.getParticipants(convID, false)
	if err == nil {
		//Note to self: What is the difference between Publish and PublishEvent?
		go api.cache.Publish(msg, participants, convID)
		chans := ConversationChannelKeys(participants)
		go api.cache.PublishEvent("message", conversationURI(convID), msg, chans)
	} else {
		log.Println("Error getting participants; didn't bradcast event to websockets")
	}
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
	participants, err := api.getParticipants(convID, false)
	if err != nil {
		log.Println(err)
		return false
	}
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
		return api.GetFullConversation(userID, convID, start, count)
	}
	return conv, &ENOTALLOWED
}

//GetFullConversation returns a full conversation containing up to count messages.
//TODO(patrick) - clarify this vs getConversation etc
func (api *API) GetFullConversation(userID gp.UserID, convID gp.ConversationID, start int64, count int) (conv gp.ConversationAndMessages, err error) {
	conv.ID = convID
	lastActivity, err := api.conversationLastActivity(userID, convID)
	if err == nil {
		conv.LastActivity = lastActivity
	}
	conv.Participants, err = api.getParticipants(convID, true)
	if err != nil {
		return
	}
	conv.Read, err = api.readStatus(convID)
	if err != nil {
		return
	}
	conv.Group, err = api.conversationGroup(convID)
	if err != nil {
		log.Println(err)
	}
	conv.Messages, err = api.getMessages(userID, convID, gp.OSTART, start, count)
	return
}

//readStatus returns the point all participants have read until in a conversation, omitting any participants who have read nothing.
//TODO: Use cache
func (api *API) readStatus(convID gp.ConversationID) (read []gp.Read, err error) {
	return api.getReadStatus(convID, true)
}

//ConversationLastActivity returns the modification time (ie, creation  or last-message) for this conversation.
func (api *API) conversationLastActivity(userID gp.UserID, convID gp.ConversationID) (t time.Time, err error) {
	return api.conversationActivity(userID, convID)
}

//UserGetMessages returns count messages from the conversation convId, or ENOTALLOWED if the user is not allowed to view this conversation.
func (api *API) UserGetMessages(userID gp.UserID, convID gp.ConversationID, mode int, index int64, count int) (messages []gp.Message, err error) {
	messages = make([]gp.Message, 0)
	if api.UserCanViewConversation(userID, convID) {
		return api.getMessages(userID, convID, mode, index, count)
	}
	return messages, &ENOTALLOWED
}

//GetConversations returns count non-ended conversations which userId participates in, starting from start and ordered by their last activity.
func (api *API) GetConversations(userID gp.UserID, start int64, count int) (conversations []gp.ConversationSmall, err error) {
	conversations, err = api.getConversations(userID, start, count)
	return
}

//GetLastMessage returns the most recent message in this conversation.
//this function doesn't appear to be used
func (api *API) getLastMessage(id gp.ConversationID) (message gp.Message, err error) {
	return api.getLastMessage(id)
}

//MarkAllConversationsSeen sets "read" = LastMessage for all user's conversations.
func (api *API) MarkAllConversationsSeen(user gp.UserID) (err error) {
	conversations, err := api.getConversations(user, 0, 10000)
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
	return unreadMessageCount(api.db, user, useThreshold)
}

//UserMuteBadges marks the user as having seen the badge for conversations before t; this means any unread messages before t will no longer be included in any badge values.
func (api *API) UserMuteBadges(userID gp.UserID, t time.Time) (err error) {
	return api.userMuteBadges(userID, t)
}

//UserAddParticipants adds new user(s) to this conversation, iff userID is in conversation && userID and participants share at least one network (ie, university)
func (api *API) UserAddParticipants(userID gp.UserID, convID gp.ConversationID, participants ...gp.UserID) (updatedParticipants []gp.User, err error) {
	updatedParticipants = make([]gp.User, 0)
	if !api.UserCanViewConversation(userID, convID) {
		log.Println("Adding participants: adder can't see the conversation themself")
		err = &ENOTALLOWED
		return
	}
	addable, err := api.addableParticipants(userID, convID, participants...)
	if err != nil {
		return
	}
	err = api.addConversationParticipants(userID, addable, convID)
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
	updatedParticipants, err = api.getParticipants(convID, false)
	return
}

//addableParticipants returns all the participants who can be added to this conversation -- ie, purges those with no shared networks and those already in the conv.
func (api *API) addableParticipants(userID gp.UserID, convID gp.ConversationID, participants ...gp.UserID) (addableParticipants []gp.UserID, err error) {
	for _, p := range participants {
		shared, err := api.sameUniversity(userID, p) //Not someone who you can see
		if !shared || err != nil {
			log.Printf("%d and %d aren't in the same uni\n", userID, p)
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
	messageID, err = api.addMessage(convID, userID, text, true)
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
	participants, err := api.getParticipants(convID, false)
	if err == nil {
		//Note to self: What is the difference between Publish and PublishEvent?
		go api.cache.Publish(msg, participants, convID)
		chans := ConversationChannelKeys(participants)
		go api.cache.PublishEvent("message", conversationURI(convID), msg, chans)
	} else {
		log.Println("Error getting participants; didn't bradcast event to websockets")
	}
	return
}

//ConversationMergedInto returns the id of the conversation this one has merged with, or err if it hasn't merged.
func (api *API) ConversationMergedInto(convID gp.ConversationID) (mergedInto gp.ConversationID, err error) {
	return api.conversationMergedInto(convID)
}

//NoSuchConversation happens when you try to find the primary conversation for a pair of users and it doesn't exist.
var NoSuchConversation = gp.APIerror{Reason: "No such conversation"}

//CreateConversation generates a new conversation with these participants and an initiator id.
func (api *API) _createConversation(id gp.UserID, participants []gp.User, primary bool, group gp.NetworkID) (conversation gp.Conversation, err error) {
	var s *sql.Stmt
	if group > 0 {
		s, err = api.db.Prepare("INSERT INTO conversations (initiator, primary_conversation, group_id) VALUES (?, ?, ?)")
	} else {
		s, err = api.db.Prepare("INSERT INTO conversations (initiator, primary_conversation) VALUES (?, ?)")
	}
	if err != nil {
		return
	}
	var r sql.Result
	if group > 0 {
		r, err = s.Exec(id, primary, group)
	} else {
		r, err = s.Exec(id, primary)
	}
	if err != nil {
		log.Println(err)
		return
	}
	cID, _ := r.LastInsertId()
	conversation.ID = gp.ConversationID(cID)
	if err != nil {
		return
	}
	log.Printf("DB hit: createConversation(userID: %d, participants: %v, primary: %t, group: %d)\n", id, participants, primary, group)
	var pids []gp.UserID
	for _, u := range participants {
		pids = append(pids, u.ID)
	}
	err = api.addConversationParticipants(id, pids, conversation.ID)
	if err != nil {
		return
	}
	conversation.Participants = participants
	conversation.LastActivity = time.Now().UTC()
	conversation.Group = group
	return
}

//AddConversationParticipants adds these participants to convID, or does nothing if they are already members.
func (api *API) addConversationParticipants(adder gp.UserID, participants []gp.UserID, convID gp.ConversationID) (err error) {
	s, err := api.db.Prepare("REPLACE INTO conversation_participants (conversation_id, participant_id, deleted) VALUES (?, ?, 0)")
	if err != nil {
		return
	}
	for _, u := range participants {
		_, err = s.Exec(convID, u)
		if err != nil {
			return
		}
	}
	return nil
}

//GetConversations returns this user's conversations;
func (api *API) getConversations(userID gp.UserID, start int64, count int) (conversations []gp.ConversationSmall, err error) {
	conversations = make([]gp.ConversationSmall, 0)
	var s *sql.Stmt
	q := "SELECT conversation_participants.conversation_id, MAX( chat_messages.`timestamp` ) AS last_mod " +
		"FROM conversation_participants " +
		"JOIN  `chat_messages` ON conversation_participants.conversation_id = chat_messages.conversation_id " +
		"JOIN conversations ON conversation_participants.conversation_id = conversations.id " +
		"WHERE conversation_participants.participant_id = ? " +
		"AND conversation_participants.deleted =0 " +
		"AND conversations.group_id IS NULL " +
		"AND chat_messages.id > conversation_participants.deletion_threshold " +
		"GROUP BY chat_messages.conversation_id " +
		"ORDER BY last_mod DESC " +
		"LIMIT ? , ? "
	s, err = api.db.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(userID, start, count)
	log.Printf("DB hit: GetConversations(userID: %d, start: %d, count: %d)\n", userID, start, count)
	if err != nil {
		return conversations, err
	}
	defer rows.Close()
	for rows.Next() {
		var conv gp.ConversationSmall
		var t string
		err = rows.Scan(&conv.ID, &t)
		if err != nil {
			return conversations, err
		}
		conv.LastActivity, _ = time.Parse(mysqlTime, t)
		conv.Participants, err = api.getParticipants(conv.ID, true)
		if err != nil {
			return conversations, err
		}
		//Drop all the weird one-participant conversations...
		if len(conv.Participants) < 2 {
			continue
		}
		LastMessage, err := api.getLastMessage(conv.ID)
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		read, err := api.getReadStatus(conv.ID, true)
		if err == nil {
			conv.Read = read
		}
		conv.Unread, err = api.userConversationUnread(userID, conv.ID)
		if err != nil {
			log.Println("error getting unread count:", err)
		}
		conversations = append(conversations, conv)
	}
	return conversations, nil
}

//ConversationActivity returns the time this conversation last changed.
func (api *API) conversationActivity(userID gp.UserID, convID gp.ConversationID) (t time.Time, err error) {
	s, err := api.db.Prepare("SELECT MAX(chat_messages.`timestamp`) AS last_mod FROM conversation_participants JOIN chat_messages ON conversation_participants.conversation_id = chat_messages.conversation_id WHERE conversation_participants.participant_id = ? AND conversation_participants.conversation_id = ?")
	if err != nil {
		return
	}
	var tstring string
	err = s.QueryRow(userID, convID).Scan(&tstring)
	if err != nil {
		return
	}
	t, err = time.Parse(mysqlTime, tstring)
	return
}

//DeleteConversation removes this conversation for this user.
func (api *API) deleteConversation(userID gp.UserID, convID gp.ConversationID) (err error) {
	s, err := api.db.Prepare("UPDATE conversation_participants SET deleted = 1 WHERE participant_id = ? AND conversation_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(userID, convID)
	return
}

//SetDeletionThreshold marks all messages before or equal to this message as "deleted" for this user. Attempting to set a lower threshold does nothing. High thresholds are reinterpreted as the <= message.
func (api *API) setDeletionThreshold(userID gp.UserID, convID gp.ConversationID, threshold gp.MessageID) (err error) {
	s, err := api.db.Prepare("UPDATE conversation_participants SET deletion_threshold = (SELECT MAX(id) FROM chat_messages WHERE chat_messages.conversation_id = ? AND chat_messages.id <= ?) WHERE conversation_participants.conversation_id = ? AND conversation_participants.participant_id = ? AND conversation_participants.deletion_threshold < ?")
	if err != nil {
		return
	}
	_, err = s.Exec(convID, threshold, convID, userID, threshold)
	return
}

//GetConversation returns the conversation convId, including up to count messages.
func (api *API) getConversation(userID gp.UserID, convID gp.ConversationID, count int) (conversation gp.ConversationAndMessages, err error) {
	conversation.ID = convID
	lastActivity, err := api.conversationActivity(userID, convID)
	if err == nil {
		conversation.LastActivity = lastActivity
	}
	conversation.Participants, err = api.getParticipants(convID, true)
	if err != nil {
		return conversation, err
	}
	read, err := api.getReadStatus(convID, true)
	if err == nil {
		conversation.Read = read
	}
	conversation.Unread, err = api.userConversationUnread(userID, convID)
	if err != nil {
		log.Println("error getting unread count:", err)
	}
	conversation.Group, err = api.conversationGroup(convID)
	if err != nil {
		log.Println(err)
	}
	conversation.Messages, err = api.getMessages(userID, convID, gp.OSTART, 0, count)
	return
}

//GetReadStatus returns all the positions the participants in this conversation have read to. If omitZeros is true, it omits participants who haven't read any messages.
func (api *API) getReadStatus(convID gp.ConversationID, omitZeros bool) (read []gp.Read, err error) {
	s, err := api.db.Prepare("SELECT participant_id, last_read FROM conversation_participants WHERE conversation_id = ?")
	if err != nil {
		return
	}
	rows, err := s.Query(convID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var r gp.Read
		err = rows.Scan(&r.UserID, &r.LastRead)
		if err != nil {
			return
		}
		if r.LastRead > 0 || !omitZeros {
			read = append(read, r)
		}
	}
	return
}

//GetParticipants returns all of the participants in conv, or omits the ones who have deleted this conversation if includeDeleted is false.
func (api *API) getParticipants(conv gp.ConversationID, includeDeleted bool) (participants []gp.User, err error) {
	q := "SELECT participant_id " +
		"FROM conversation_participants " +
		"JOIN users ON conversation_participants.participant_id = users.id " +
		"WHERE conversation_id=?"
	if !includeDeleted {
		q += " AND deleted = 0"
	}
	s, err := api.db.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(conv)
	log.Printf("DB hit: GetParticipants(conv: %d, includeDeleted: %t)\n", conv, includeDeleted)
	if err != nil {
		log.Printf("Error getting participant: %v", err)
		return
	}
	defer rows.Close()
	participants = make([]gp.User, 0, 5)
	for rows.Next() {
		var id gp.UserID
		err = rows.Scan(&id)
		user, err := api.getUser(id)
		if err == nil {
			participants = append(participants, user)
		}
	}
	return participants, nil
}

//GetLastMessage retrieves the most recent message in conversation id.
func (api *API) GetLastMessage(id gp.ConversationID) (message gp.Message, err error) {
	var timeString string
	var by gp.UserID
	//Ordered by id rather than timestamp because timestamps are limited to 1-second resolution
	//ie, the last message by timestamp may be _several
	//note: this won't work if we move away from incremental message ids.
	q := "SELECT id, `from`, text, `timestamp`, `system`" +
		"FROM chat_messages " +
		"WHERE conversation_id = ? " +
		"ORDER BY `id` DESC LIMIT 1"
	s, err := api.db.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&message.ID, &by, &message.Text, &timeString, &message.System)
	log.Printf("DB hit: db.GetLastMessage(%d)\n", id)
	log.Println("Message is:", message, "Len of message.Text:", len(message.Text))
	if err != nil {
		return message, err
	}
	message.By, err = api.getUser(by)
	if err != nil {
		log.Printf("error getting user %d %v", by, err)
	}
	message.Time, _ = time.Parse(mysqlTime, timeString)

	return message, nil
}

//AddMessage records this message in the database. System represents whether this is a system- or user-generated message.
func (api *API) addMessage(convID gp.ConversationID, userID gp.UserID, text string, system bool) (id gp.MessageID, err error) {
	log.Printf("Adding message to db: %d, %d %s, system: %v\n", convID, userID, text, system)
	s, err := api.db.Prepare("INSERT INTO chat_messages (conversation_id, `from`, `text`, `system`) VALUES (?,?,?,?)")
	if err != nil {
		return
	}
	res, err := s.Exec(convID, userID, text, system)
	if err != nil {
		return 0, err
	}
	_id, err := res.LastInsertId()
	id = gp.MessageID(_id)
	return
}

//GetMessages retrieves n = count messages from the conversation convId.
//These can be starting from the offset index (when sel == "start"); or they can
//be the n messages before or after index when sel == "before" or "after" respectively.
//I don't know what will happen if you give sel something else, probably a null pointer
//exception.
//TODO: This could return a message which doesn't embed a user
//BUG(Patrick): Should return an error when sel isn't right!
func (api *API) getMessages(userID gp.UserID, convID gp.ConversationID, mode int, index int64, count int) (messages []gp.Message, err error) {
	messages = make([]gp.Message, 0)
	var s *sql.Stmt
	var q string
	switch {
	case mode == gp.OAFTER:
		q = "SELECT id, `from`, text, `timestamp`, `system`" +
			"FROM chat_messages " +
			"WHERE chat_messages.conversation_id = ? " +
			"AND chat_messages.id > (SELECT deletion_threshold FROM conversation_participants WHERE participant_id = ? AND conversation_id = ?) " +
			"AND id > ? " +
			"ORDER BY `timestamp` DESC LIMIT ?"
	case mode == gp.OBEFORE:
		q = "SELECT id, `from`, text, `timestamp`, `system`" +
			"FROM chat_messages " +
			"WHERE chat_messages.conversation_id = ? " +
			"AND chat_messages.id > (SELECT deletion_threshold FROM conversation_participants WHERE participant_id = ? AND conversation_id = ?) " +
			"AND id < ? " +
			"ORDER BY `timestamp` DESC LIMIT ?"
	case mode == gp.OSTART:
		q = "SELECT id, `from`, text, `timestamp`, `system`" +
			"FROM chat_messages " +
			"WHERE chat_messages.conversation_id = ? " +
			"AND chat_messages.id > (SELECT deletion_threshold FROM conversation_participants WHERE participant_id = ? AND conversation_id = ?) " +
			"ORDER BY `timestamp` DESC LIMIT ?, ?"
	}
	s, err = api.db.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(convID, userID, convID, index, count)
	log.Println("DB hit: GetMessages(user: %d, convID: %d, mode: %d, index: %d, count: %d)\n", userID, convID, mode, index, count)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var message gp.Message
		var timeString string
		var by gp.UserID
		err = rows.Scan(&message.ID, &by, &message.Text, &timeString, &message.System)
		if err != nil {
			log.Printf("%v", err)
		}
		message.Time, err = time.Parse(mysqlTime, timeString)
		if err != nil {
			log.Printf("%v", err)
		}
		message.By, err = api.getUser(by)
		if err != nil {
			log.Println("Error getting this message's sender:", err)
			continue
		}
		messages = append(messages, message)
	}
	return
}

//MarkRead moves this user's "read" marker up to this message in this conversation.
func (api *API) markRead(id gp.UserID, convID gp.ConversationID, upTo gp.MessageID) (read gp.MessageID, err error) {
	s, err := api.db.Prepare("UPDATE conversation_participants " +
		"SET last_read = (SELECT MAX(id) FROM chat_messages WHERE conversation_id = ? AND id <= ?) " +
		"WHERE `conversation_id` = ? AND `participant_id` = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(convID, upTo, convID, id)
	if err != nil {
		return
	}
	s, err = api.db.Prepare("SELECT last_read FROM conversation_participants WHERE conversation_id = ? AND participant_id = ?")
	err = s.QueryRow(convID, id).Scan(&read)
	return
}

//UnreadMessageCount returns the number of unread messages this user has, optionally omitting those before their threshold time.
//TODO(Patrick) - convert this into a single query
func unreadMessageCount(db *sql.DB, user gp.UserID, useThreshold bool) (count int, err error) {
	qParticipate := "SELECT conversation_id, last_read, deletion_threshold FROM conversation_participants WHERE participant_id = ? AND deleted = 0"
	sParticipate, err := db.Prepare(qParticipate)
	if err != nil {
		return
	}
	rows, err := sParticipate.Query(user)
	if err != nil {
		return
	}
	defer rows.Close()

	qUnreadCount := "SELECT count(*) FROM chat_messages WHERE chat_messages.conversation_id = ? AND chat_messages.id > ? AND `system` = 0 AND chat_messages.id > ?"
	if useThreshold {
		qUnreadCount = "SELECT count(*) FROM chat_messages WHERE chat_messages.conversation_id = ? AND chat_messages.id > ? AND chat_messages.timestamp > (SELECT new_message_threshold FROM users WHERE id = ?) AND `system` = 0 AND chat_messages.id > ?"

	}
	sUnreadCount, err := db.Prepare(qUnreadCount)
	if err != nil {
		return
	}
	var convID gp.ConversationID
	var lastID gp.MessageID
	var deletedID gp.MessageID
	for rows.Next() {
		err = rows.Scan(&convID, &lastID, &deletedID)
		if err != nil {
			return
		}
		log.Printf("Conversation %d, last read message was %d\n", convID, lastID)
		_count := 0
		if !useThreshold {
			err = sUnreadCount.QueryRow(convID, lastID, deletedID).Scan(&_count)
		} else {
			err = sUnreadCount.QueryRow(convID, lastID, user, deletedID).Scan(&_count)
		}
		if err == nil {
			log.Printf("Conversation %d, unread message count was %d\n", convID, _count)
			count += _count
		} else {
			log.Println("Error calculating badge for:", convID, _count, user, err)
		}
	}
	return count, nil
}

//UserMuteBadges marks the user as having seen the badge for conversations before t; this means any unread messages before t will no longer be included in any badge values.
func (api *API) userMuteBadges(userID gp.UserID, t time.Time) (err error) {
	q := "UPDATE users SET new_message_threshold = ? WHERE id = ?"
	s, err := api.db.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(t, userID)
	return
}

//UserConversationUnread returns the nubmer of unread messages in this conversation for this user.
//If userID == 0, will return 0, nil.
func (api *API) userConversationUnread(userID gp.UserID, convID gp.ConversationID) (unread int, err error) {
	if userID == 0 {
		return 0, nil
	}
	q := "SELECT COUNT(*) FROM chat_messages " +
		"JOIN conversations ON conversations.id = chat_messages.conversation_id " +
		"WHERE conversation_id = ? AND EXISTS " +
		"(SELECT last_read FROM conversation_participants WHERE conversation_id = ? AND participant_id = ?) " +
		"AND chat_messages.id > " +
		"(SELECT last_read FROM conversation_participants " +
		"WHERE conversation_id = ? AND participant_id = ?) " +
		"AND chat_messages.id > " +
		"(SELECT deletion_threshold FROM conversation_participants " +
		"WHERE conversation_id = ? AND participant_id = ?) " +
		"AND `system` = 0 AND conversations.group_id IS NULL"
	s, err := api.db.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(convID, convID, userID, convID, userID, convID, userID).Scan(&unread)
	return
}

//GetPrimaryConversation returns the primary conversation for this set of users, or NoSuchConversation otherwise.
func (api *API) getPrimaryConversation(participantA, participantB gp.UserID) (conversation gp.ConversationAndMessages, err error) {
	q := "SELECT conversation_id FROM conversation_participants JOIN conversations ON conversations.id = conversation_participants.conversation_id WHERE conversations.primary_conversation=1 AND participant_id IN (?, ?) GROUP BY conversation_id HAVING count(*) = 2"
	s, err := api.db.Prepare(q)
	if err != nil {
		return
	}
	var conv gp.ConversationID
	err = s.QueryRow(participantA, participantB).Scan(&conv)
	if err != nil {
		return
	}
	return api.getConversation(participantA, conv, 20)
}

//ErrNotMerged is the result when you try to find the conversation another has been merged into but the conversation has not been merged.
var ErrNotMerged = gp.APIerror{Reason: "Conversation not merged"}

//ConversationMergedInto returns the id of the conversation this one has merged with, or err if it hasn't merged.
func (api *API) conversationMergedInto(convID gp.ConversationID) (merged gp.ConversationID, err error) {
	q := "SELECT merged FROM conversations WHERE id = ?"
	s, err := api.db.Prepare(q)
	if err != nil {
		return
	}
	var _merged sql.NullInt64
	err = s.QueryRow(convID).Scan(&_merged)
	if err != nil {
		return
	}
	if !_merged.Valid {
		return merged, ErrNotMerged
	}
	return gp.ConversationID(_merged.Int64), nil
}

//ConversationGroup returns the id of the group this conversation is connected to, or zero if it isn't.
func (api *API) conversationGroup(convID gp.ConversationID) (group gp.NetworkID, err error) {
	q := "SELECT group_id FROM conversations WHERE id = ?"
	s, err := api.db.Prepare(q)
	if err != nil {
		return
	}
	var _group sql.NullInt64
	err = s.QueryRow(convID).Scan(&_group)
	if err != nil {
		return
	}
	return gp.NetworkID(_group.Int64), nil //This is OK because a missing group ID will be 0
}

//IsPrimaryConversation returns true if this is a primary conversation.
func (api *API) isPrimaryConversation(convID gp.ConversationID) (primary bool, err error) {
	s, err := api.db.Prepare("SELECT primary_conversation FROM conversations WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(convID).Scan(&primary)
	return
}
