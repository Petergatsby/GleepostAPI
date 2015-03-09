package db

import (
	"database/sql"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//NoSuchConversation happens when you try to find the primary conversation for a pair of users and it doesn't exist.
var NoSuchConversation = gp.APIerror{Reason: "No such conversation"}

//CreateConversation generates a new conversation with these participants and an initiator id.
func (db *DB) CreateConversation(id gp.UserID, participants []gp.User, primary bool, group gp.NetworkID) (conversation gp.Conversation, err error) {
	var s *sql.Stmt
	if group > 0 {
		s, err = db.prepare("INSERT INTO conversations (initiator, primary_conversation, group_id) VALUES (?, ?, ?)")
	} else {
		s, err = db.prepare("INSERT INTO conversations (initiator, primary_conversation) VALUES (?, ?)")
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
	log.Println("DB hit: createConversation (user.Name, user.Id)")
	var pids []gp.UserID
	for _, u := range participants {
		pids = append(pids, u.ID)
	}
	err = db.AddConversationParticipants(id, pids, conversation.ID)
	if err != nil {
		return
	}
	conversation.Participants = participants
	conversation.LastActivity = time.Now().UTC()
	conversation.Group = group
	return
}

//AddConversationParticipants adds these participants to convID, or does nothing if they are already members.
func (db *DB) AddConversationParticipants(adder gp.UserID, participants []gp.UserID, convID gp.ConversationID) (err error) {
	s, err := db.prepare("REPLACE INTO conversation_participants (conversation_id, participant_id, deleted) VALUES (?, ?, 0)")
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
func (db *DB) GetConversations(userID gp.UserID, start int64, count int) (conversations []gp.ConversationSmall, err error) {
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
	s, err = db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(userID, start, count)
	log.Println("DB hit: getConversations user_id, start (conversation.id)")
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
		conv.Participants, err = db.GetParticipants(conv.ID, true)
		if err != nil {
			return conversations, err
		}
		//Drop all the weird one-participant conversations...
		if len(conv.Participants) < 2 {
			continue
		}
		LastMessage, err := db.GetLastMessage(conv.ID)
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		read, err := db.GetReadStatus(conv.ID, true)
		if err == nil {
			conv.Read = read
		}
		conv.Unread, err = db.UserConversationUnread(userID, conv.ID)
		if err != nil {
			log.Println("error getting unread count:", err)
		}
		conversations = append(conversations, conv)
	}
	return conversations, nil
}

//ConversationActivity returns the time this conversation last changed.
func (db *DB) ConversationActivity(userID gp.UserID, convID gp.ConversationID) (t time.Time, err error) {
	s, err := db.prepare("SELECT MAX(chat_messages.`timestamp`) AS last_mod FROM conversation_participants JOIN chat_messages ON conversation_participants.conversation_id = chat_messages.conversation_id WHERE conversation_participants.participant_id = ? AND conversation_participants.conversation_id = ?")
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
func (db *DB) DeleteConversation(userID gp.UserID, convID gp.ConversationID) (err error) {
	s, err := db.prepare("UPDATE conversation_participants SET deleted = 1 WHERE participant_id = ? AND conversation_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(userID, convID)
	return
}

//SetDeletionThreshold marks all messages before or equal to this message as "deleted" for this user. Attempting to set a lower threshold does nothing. High thresholds are reinterpreted as the <= message.
func (db *DB) SetDeletionThreshold(userID gp.UserID, convID gp.ConversationID, threshold gp.MessageID) (err error) {
	s, err := db.prepare("UPDATE conversation_participants SET deletion_threshold = (SELECT MAX(id) FROM chat_messages WHERE chat_messages.conversation_id = ? AND chat_messages.id <= ?) WHERE conversation_participants.conversation_id = ? AND conversation_participants.participant_id = ? AND conversation_participants.deletion_threshold < ?")
	if err != nil {
		return
	}
	_, err = s.Exec(convID, threshold, convID, userID, threshold)
	return
}

//GetConversation returns the conversation convId, including up to count messages.
func (db *DB) GetConversation(userID gp.UserID, convID gp.ConversationID, count int) (conversation gp.ConversationAndMessages, err error) {
	conversation.ID = convID
	lastActivity, err := db.ConversationActivity(userID, convID)
	if err == nil {
		conversation.LastActivity = lastActivity
	}
	conversation.Participants, err = db.GetParticipants(convID, true)
	if err != nil {
		return conversation, err
	}
	read, err := db.GetReadStatus(convID, true)
	if err == nil {
		conversation.Read = read
	}
	conversation.Unread, err = db.UserConversationUnread(userID, convID)
	if err != nil {
		log.Println("error getting unread count:", err)
	}
	conversation.Group, err = db.ConversationGroup(convID)
	if err != nil {
		log.Println(err)
	}
	conversation.Messages, err = db.GetMessages(userID, convID, 0, "start", count)
	return
}

//GetReadStatus returns all the positions the participants in this conversation have read to. If omitZeros is true, it omits participants who haven't read any messages.
func (db *DB) GetReadStatus(convID gp.ConversationID, omitZeros bool) (read []gp.Read, err error) {
	s, err := db.prepare("SELECT participant_id, last_read FROM conversation_participants WHERE conversation_id = ?")
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
func (db *DB) GetParticipants(conv gp.ConversationID, includeDeleted bool) (participants []gp.User, err error) {
	q := "SELECT participant_id " +
		"FROM conversation_participants " +
		"JOIN users ON conversation_participants.participant_id = users.id " +
		"WHERE conversation_id=?"
	if !includeDeleted {
		q += " AND deleted = 0"
	}
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(conv)
	log.Println("DB hit: getParticipants convid (user.id)")
	if err != nil {
		log.Printf("Error getting participant: %v", err)
		return
	}
	defer rows.Close()
	participants = make([]gp.User, 0, 5)
	for rows.Next() {
		var id gp.UserID
		err = rows.Scan(&id)
		user, err := db.GetUser(id)
		if err == nil {
			participants = append(participants, user)
		}
	}
	return participants, nil
}

//GetLastMessage retrieves the most recent message in conversation id.
func (db *DB) GetLastMessage(id gp.ConversationID) (message gp.Message, err error) {
	var timeString string
	var by gp.UserID
	//Ordered by id rather than timestamp because timestamps are limited to 1-second resolution
	//ie, the last message by timestamp may be _several
	//note: this won't work if we move away from incremental message ids.
	q := "SELECT id, `from`, text, `timestamp`, `system`" +
		"FROM chat_messages " +
		"WHERE conversation_id = ? " +
		"ORDER BY `id` DESC LIMIT 1"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&message.ID, &by, &message.Text, &timeString, &message.System)
	log.Println("DB hit: db.GetLastMessage convid (message.id, message.by, message.text, message.time)")
	log.Println("Message is:", message, "Len of message.Text:", len(message.Text))
	if err != nil {
		return message, err
	}
	message.By, err = db.GetUser(by)
	if err != nil {
		log.Printf("error getting user %d %v", by, err)
	}
	message.Time, _ = time.Parse(mysqlTime, timeString)

	return message, nil
}

//AddMessage records this message in the database. System represents whether this is a system- or user-generated message.
func (db *DB) AddMessage(convID gp.ConversationID, userID gp.UserID, text string, system bool) (id gp.MessageID, err error) {
	log.Printf("Adding message to db: %d, %d %s, system: %v\n", convID, userID, text, system)
	s, err := db.prepare("INSERT INTO chat_messages (conversation_id, `from`, `text`, `system`) VALUES (?,?,?,?)")
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
func (db *DB) GetMessages(userID gp.UserID, convID gp.ConversationID, index int64, sel string, count int) (messages []gp.Message, err error) {
	messages = make([]gp.Message, 0)
	var s *sql.Stmt
	var q string
	switch {
	case sel == "after":
		q = "SELECT id, `from`, text, `timestamp`, `system`" +
			"FROM chat_messages " +
			"WHERE chat_messages.conversation_id = ? " +
			"AND chat_messages.id > (SELECT deletion_threshold FROM conversation_participants WHERE participant_id = ? AND conversation_id = ?) " +
			"AND id > ? " +
			"ORDER BY `timestamp` DESC LIMIT ?"
	case sel == "before":
		q = "SELECT id, `from`, text, `timestamp`, `system`" +
			"FROM chat_messages " +
			"WHERE chat_messages.conversation_id = ? " +
			"AND chat_messages.id > (SELECT deletion_threshold FROM conversation_participants WHERE participant_id = ? AND conversation_id = ?) " +
			"AND id < ? " +
			"ORDER BY `timestamp` DESC LIMIT ?"
	case sel == "start":
		q = "SELECT id, `from`, text, `timestamp`, `system`" +
			"FROM chat_messages " +
			"WHERE chat_messages.conversation_id = ? " +
			"AND chat_messages.id > (SELECT deletion_threshold FROM conversation_participants WHERE participant_id = ? AND conversation_id = ?) " +
			"ORDER BY `timestamp` DESC LIMIT ?, ?"
	}
	s, err = db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(convID, userID, convID, index, count)
	log.Println("DB hit: getMessages convid, start (message.id, message.by, message.text, message.time)")
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
		message.By, err = db.GetUser(by)
		if err != nil {
			log.Println("Error getting this message's sender:", err)
			continue
		}
		messages = append(messages, message)
	}
	return
}

//MarkRead moves this user's "read" marker up to this message in this conversation.
func (db *DB) MarkRead(id gp.UserID, convID gp.ConversationID, upTo gp.MessageID) (read gp.MessageID, err error) {
	s, err := db.prepare("UPDATE conversation_participants " +
		"SET last_read = (SELECT MAX(id) FROM chat_messages WHERE conversation_id = ? AND id <= ?) " +
		"WHERE `conversation_id` = ? AND `participant_id` = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(convID, upTo, convID, id)
	if err != nil {
		return
	}
	s, err = db.prepare("SELECT last_read FROM conversation_participants WHERE conversation_id = ? AND participant_id = ?")
	err = s.QueryRow(convID, id).Scan(&read)
	return
}

//UnreadMessageCount returns the number of unread messages this user has, optionally omitting those before their threshold time.
//TODO(Patrick) - convert this into a single query
func (db *DB) UnreadMessageCount(user gp.UserID, useThreshold bool) (count int, err error) {
	qParticipate := "SELECT conversation_id, last_read, deletion_threshold FROM conversation_participants WHERE participant_id = ? AND deleted = 0"
	sParticipate, err := db.prepare(qParticipate)
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
	sUnreadCount, err := db.prepare(qUnreadCount)
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
func (db *DB) UserMuteBadges(userID gp.UserID, t time.Time) (err error) {
	q := "UPDATE users SET new_message_threshold = ? WHERE id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(t, userID)
	return
}

//UserConversationUnread returns the nubmer of unread messages in this conversation for this user.
//If userID == 0, will return 0, nil.
func (db *DB) UserConversationUnread(userID gp.UserID, convID gp.ConversationID) (unread int, err error) {
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
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(convID, convID, userID, convID, userID, convID, userID).Scan(&unread)
	return
}

//GetPrimaryConversation returns the primary conversation for this set of users, or NoSuchConversation otherwise.
func (db *DB) GetPrimaryConversation(participantA, participantB gp.UserID) (conversation gp.ConversationAndMessages, err error) {
	q := "SELECT conversation_id FROM conversation_participants JOIN conversations ON conversations.id = conversation_participants.conversation_id WHERE conversations.primary_conversation=1 AND participant_id IN (?, ?) GROUP BY conversation_id HAVING count(*) = 2"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	var conv gp.ConversationID
	err = s.QueryRow(participantA, participantB).Scan(&conv)
	if err != nil {
		return
	}
	return db.GetConversation(participantA, conv, 20)
}

//ErrNotMerged is the result when you try to find the conversation another has been merged into but the conversation has not been merged.
var ErrNotMerged = gp.APIerror{Reason: "Conversation not merged"}

//ConversationMergedInto returns the id of the conversation this one has merged with, or err if it hasn't merged.
func (db *DB) ConversationMergedInto(convID gp.ConversationID) (merged gp.ConversationID, err error) {
	q := "SELECT merged FROM conversations WHERE id = ?"
	s, err := db.prepare(q)
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
func (db *DB) ConversationGroup(convID gp.ConversationID) (group gp.NetworkID, err error) {
	q := "SELECT group_id FROM conversations WHERE id = ?"
	s, err := db.prepare(q)
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
func (db *DB) IsPrimaryConversation(convID gp.ConversationID) (primary bool, err error) {
	s, err := db.prepare("SELECT primary_conversation FROM conversations WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(convID).Scan(&primary)
	return
}
