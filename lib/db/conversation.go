package db

import (
	"database/sql"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//GetLiveConversations returns the three most recent unfinished live conversations for a given user.
//TODO: retrieve conversation & expiry in a single query
func (db *DB) GetLiveConversations(id gp.UserID) (conversations []gp.ConversationSmall, err error) {
	conversations = make([]gp.ConversationSmall, 0)
	q := "SELECT conversation_participants.conversation_id, conversations.last_mod " +
		"FROM conversation_participants " +
		"JOIN conversations ON conversation_participants.conversation_id = conversations.id " +
		"JOIN conversation_expirations ON conversation_expirations.conversation_id = conversations.id " +
		"WHERE participant_id = ? " +
		"AND conversation_expirations.ended = 0 " +
		"ORDER BY conversations.last_mod DESC  " +
		"LIMIT 0 , 3"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(id)
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
		LastMessage, err := db.GetLastMessage(conv.ID)
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		Expiry, err := db.ConversationExpiry(conv.ID)
		if err == nil {
			conv.Expiry = &Expiry
		}
		conv.Unread, err = db.UserConversationUnread(id, conv.ID)
		if err != nil {
			log.Println("error getting unread count:", err)
		}
		conversations = append(conversations, conv)
	}
	return conversations, nil
}

//CreateConversation generates a new conversation with these participants and an initiator id. Expiry is optional.
func (db *DB) CreateConversation(id gp.UserID, participants []gp.User, expiry *gp.Expiry) (conversation gp.Conversation, err error) {
	s, err := db.prepare("INSERT INTO conversations (initiator, last_mod) VALUES (?, NOW())")
	if err != nil {
		return
	}
	r, _ := s.Exec(id)
	cID, _ := r.LastInsertId()
	conversation.ID = gp.ConversationID(cID)
	if err != nil {
		return
	}
	log.Println("DB hit: createConversation (user.Name, user.Id)")
	s, err = db.prepare("INSERT INTO conversation_participants (conversation_id, participant_id) VALUES (?,?)")
	if err != nil {
		return
	}
	for _, u := range participants {
		_, err = s.Exec(conversation.ID, u.ID)
		if err != nil {
			return
		}
	}
	conversation.Participants = participants
	conversation.LastActivity = time.Now().UTC()
	if expiry != nil {
		conversation.Expiry = expiry
		err = db.ConversationSetExpiry(conversation.ID, *conversation.Expiry)
	}
	return
}

//RandomPartners generates count users randomly (id ∉ participants).
func (db *DB) RandomPartners(id gp.UserID, count int, network gp.NetworkID) (partners []gp.User, err error) {
	q := "SELECT DISTINCT id, firstname, avatar, official " +
		"FROM users " +
		"LEFT JOIN user_network ON id = user_id " +
		"JOIN devices ON users.id = devices.user_id " +
		"WHERE network_id = ? " +
		"AND verified = 1 " +
		"ORDER BY RAND()"
	log.Println(q, id, count, network)

	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(network)
	if err != nil {
		log.Println("Error after initial query when generating partners")
		return
	}
	defer rows.Close()
	for rows.Next() && count > 0 {
		var user gp.User
		var av sql.NullString
		err = rows.Scan(&user.ID, &user.Name, &av, &user.Official)
		if err != nil {
			log.Println("Error scanning from user query", err)
			return
		}
		log.Println("Got a partner")
		liveCount, err := db.LiveCount(user.ID)
		if err == nil && liveCount < 3 && user.ID != id {
			if av.Valid {
				user.Avatar = av.String
			}
			partners = append(partners, user)
			count--
		}
	}
	return
}

//LiveCount returns the total number of conversations which are currently live for this user (ie, have an expiry, which is in the future, and are not ended.)
func (db *DB) LiveCount(userID gp.UserID) (count int, err error) {
	q := "SELECT COUNT( conversation_participants.conversation_id ) FROM conversation_participants JOIN conversations ON conversation_participants.conversation_id = conversations.id JOIN conversation_expirations ON conversation_expirations.conversation_id = conversations.id WHERE participant_id = ? AND conversation_expirations.ended = 0 AND conversation_expirations.expiry > NOW( )"
	stmt, err := db.prepare(q)
	if err != nil {
		return
	}
	err = stmt.QueryRow(userID).Scan(&count)
	return
}

//UpdateConversation marks this conversation as modified now.
func (db *DB) UpdateConversation(id gp.ConversationID) (err error) {
	s, err := db.prepare("UPDATE conversations SET last_mod = NOW() WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(id)
	log.Println("DB hit: updateConversation convid ")
	if err != nil {
		log.Printf("Error: %v", err)
	}
	return err
}

//GetConversations returns this user's conversations; if all is false, it will omit live conversations.
func (db *DB) GetConversations(userID gp.UserID, start int64, count int, all bool) (conversations []gp.ConversationSmall, err error) {
	conversations = make([]gp.ConversationSmall, 0)
	var s *sql.Stmt
	var q string
	if all {
		q = "SELECT conversation_participants.conversation_id, conversations.last_mod " +
			"FROM conversation_participants " +
			"JOIN conversations ON conversation_participants.conversation_id = conversations.id " +
			"LEFT OUTER JOIN conversation_expirations ON conversation_expirations.conversation_id = conversations.id " +
			"WHERE participant_id = ? " +
			"ORDER BY conversations.last_mod DESC LIMIT ?, ?"
	} else {
		q = "SELECT conversation_participants.conversation_id, conversations.last_mod " +
			"FROM conversation_participants " +
			"JOIN conversations ON conversation_participants.conversation_id = conversations.id " +
			"LEFT OUTER JOIN conversation_expirations ON conversation_expirations.conversation_id = conversations.id " +
			"WHERE participant_id = ? AND ( " +
			"conversation_expirations.ended IS NULL " +
			"OR conversation_expirations.ended =0 " +
			") " +
			"AND deleted = 0 " +
			"ORDER BY conversations.last_mod DESC LIMIT ?, ?"
	}
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
		Expiry, err := db.ConversationExpiry(conv.ID)
		if err == nil {
			conv.Expiry = &Expiry
		}
		read, err := db.GetReadStatus(conv.ID)
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
func (db *DB) ConversationActivity(convID gp.ConversationID) (t time.Time, err error) {
	s, err := db.prepare("SELECT last_mod FROM conversations WHERE id = ?")
	if err != nil {
		return
	}
	var tstring string
	err = s.QueryRow(convID).Scan(&tstring)
	if err != nil {
		return
	}
	t, err = time.Parse(mysqlTime, tstring)
	return
}

//ConversationExpiry returns this conversation's expiry, or an error if it doesn't have one.
func (db *DB) ConversationExpiry(convID gp.ConversationID) (expiry gp.Expiry, err error) {
	s, err := db.prepare("SELECT expiry, ended FROM conversation_expirations WHERE conversation_id = ?")
	if err != nil {
		return
	}
	var t string
	err = s.QueryRow(convID).Scan(&t, &expiry.Ended)
	if err != nil {
		return
	}
	expiry.Time, err = time.Parse(mysqlTime, t)
	return
}

//DeleteConversationExpiry removes this conversation's expiry, effectively converting it to a regular conversation.
func (db *DB) DeleteConversationExpiry(convID gp.ConversationID) (err error) {
	s, err := db.prepare("DELETE FROM conversation_expirations WHERE conversation_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(convID)
	return
}

//TerminateConversation ends this conversation.
func (db *DB) TerminateConversation(convID gp.ConversationID) (err error) {
	s, err := db.prepare("UPDATE conversation_expirations SET ended = 1 WHERE conversation_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(convID)
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

//ConversationSetExpiry updates this conversation's expiry to equal expiry.
func (db *DB) ConversationSetExpiry(convID gp.ConversationID, expiry gp.Expiry) (err error) {
	s, err := db.prepare("REPLACE INTO conversation_expirations (conversation_id, expiry) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(convID, expiry.Time)
	return
}

//GetConversation returns the conversation convId, including up to count messages.
func (db *DB) GetConversation(userID gp.UserID, convID gp.ConversationID, count int) (conversation gp.ConversationAndMessages, err error) {
	conversation.ID = convID
	conversation.LastActivity, err = db.ConversationActivity(convID)
	if err != nil {
		return
	}
	conversation.Participants, err = db.GetParticipants(convID, true)
	if err != nil {
		return conversation, err
	}
	read, err := db.GetReadStatus(convID)
	if err == nil {
		conversation.Read = read
	}
	expiry, err := db.ConversationExpiry(convID)
	if err == nil {
		conversation.Expiry = &expiry
	}
	conversation.Unread, err = db.UserConversationUnread(userID, convID)
	if err != nil {
		log.Println("error getting unread count:", err)
	}
	conversation.Messages, err = db.GetMessages(convID, 0, "start", count)
	return
}

//ConversationsToTerminate finds all of this user's live conversations except for the three with their expiries furthest in the future.
func (db *DB) ConversationsToTerminate(id gp.UserID) (conversations []gp.ConversationID, err error) {
	q := "SELECT conversation_participants.conversation_id " +
		"FROM conversation_participants " +
		"JOIN conversations ON conversation_participants.conversation_id = conversations.id " +
		"JOIN conversation_expirations ON conversation_expirations.conversation_id = conversations.id " +
		"WHERE participant_id = ? " +
		"AND conversation_expirations.ended = 0 " +
		"ORDER BY conversation_expirations.expiry DESC  " +
		"LIMIT 2 , 20"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(id)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id gp.ConversationID
		err = rows.Scan(&id)
		if err != nil {
			return
		}
		conversations = append(conversations, id)
	}
	return
}

//GetReadStatus returns all the positions the participants in this conversation have read to. It omits participants who haven't read.
func (db *DB) GetReadStatus(convID gp.ConversationID) (read []gp.Read, err error) {
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
		if r.LastRead > 0 {
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
	q := "SELECT id, `from`, text, `timestamp`" +
		"FROM chat_messages " +
		"WHERE conversation_id = ? " +
		"ORDER BY `id` DESC LIMIT 1"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&message.ID, &by, &message.Text, &timeString)
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

//AddMessage records this message in the database.
func (db *DB) AddMessage(convID gp.ConversationID, userID gp.UserID, text string) (id gp.MessageID, err error) {
	log.Printf("Adding message to db: %d, %d %s", convID, userID, text)
	s, err := db.prepare("INSERT INTO chat_messages (conversation_id, `from`, `text`) VALUES (?,?,?)")
	if err != nil {
		return
	}
	res, err := s.Exec(convID, userID, text)
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
func (db *DB) GetMessages(convID gp.ConversationID, index int64, sel string, count int) (messages []gp.Message, err error) {
	messages = make([]gp.Message, 0)
	var s *sql.Stmt
	var q string
	switch {
	case sel == "after":
		q = "SELECT id, `from`, text, `timestamp`" +
			"FROM chat_messages " +
			"WHERE conversation_id = ? AND id > ? " +
			"ORDER BY `timestamp` DESC LIMIT ?"
	case sel == "before":
		q = "SELECT id, `from`, text, `timestamp`" +
			"FROM chat_messages " +
			"WHERE conversation_id = ? AND id < ? " +
			"ORDER BY `timestamp` DESC LIMIT ?"
	case sel == "start":
		q = "SELECT id, `from`, text, `timestamp`" +
			"FROM chat_messages " +
			"WHERE conversation_id = ? " +
			"ORDER BY `timestamp` DESC LIMIT ?, ?"
	}
	s, err = db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(convID, index, count)
	log.Println("DB hit: getMessages convid, start (message.id, message.by, message.text, message.time)")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var message gp.Message
		var timeString string
		var by gp.UserID
		err = rows.Scan(&message.ID, &by, &message.Text, &timeString)
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
func (db *DB) MarkRead(id gp.UserID, convID gp.ConversationID, upTo gp.MessageID) (err error) {
	s, err := db.prepare("UPDATE conversation_participants " +
		"SET last_read = ? " +
		"WHERE `conversation_id` = ? AND `participant_id` = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(upTo, convID, id)
	return
}

//UnreadMessageCount returns the number of unread messages this user has, optionally omitting those before their threshold time.
func (db *DB) UnreadMessageCount(user gp.UserID, useThreshold bool) (count int, err error) {
	qParticipate := "SELECT conversation_id, last_read FROM conversation_participants WHERE participant_id = ? AND deleted = 0"
	sParticipate, err := db.prepare(qParticipate)
	if err != nil {
		return
	}
	rows, err := sParticipate.Query(user)
	if err != nil {
		return
	}
	defer rows.Close()

	qUnreadCount := "SELECT count(*) FROM chat_messages WHERE chat_messages.conversation_id = ? AND chat_messages.id > ?"
	if useThreshold {
		qUnreadCount = "SELECT count(*) FROM chat_messages WHERE chat_messages.conversation_id = ? AND chat_messages.id > ? AND chat_messages.timestamp > (SELECT new_message_threshold FROM users WHERE id = ?)"

	}
	sUnreadCount, err := db.prepare(qUnreadCount)
	if err != nil {
		return
	}
	var convID gp.ConversationID
	var lastID gp.MessageID
	for rows.Next() {
		err = rows.Scan(&convID, &lastID)
		if err != nil {
			return
		}
		log.Printf("Conversation %d, last read message was %d\n", convID, lastID)
		_count := 0
		if !useThreshold {
			err = sUnreadCount.QueryRow(convID, lastID).Scan(&_count)
		} else {
			err = sUnreadCount.QueryRow(convID, lastID, user).Scan(&_count)
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

//TotalLiveConversations returns the number of non-ended live conversations this user has.
func (db *DB) TotalLiveConversations(user gp.UserID) (count int, err error) {
	q := "SELECT conversation_participants.conversation_id, conversations.last_mod " +
		"FROM conversation_participants " +
		"JOIN conversations ON conversation_participants.conversation_id = conversations.id " +
		"JOIN conversation_expirations ON conversation_expirations.conversation_id = conversations.id " +
		"WHERE participant_id = ? " +
		"AND conversation_expirations.ended = 0 " +
		"ORDER BY conversations.last_mod DESC"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(user)
	if err != nil {
		return count, err
	}
	defer rows.Close()
	var conversations []gp.ConversationSmall
	for rows.Next() {
		var conv gp.ConversationSmall
		var t string
		err = rows.Scan(&conv.ID, &t)
		if err != nil {
			return 0, err
		}
		conv.LastActivity, _ = time.Parse(mysqlTime, t)
		conv.Participants, err = db.GetParticipants(conv.ID, true)
		if err != nil {
			return 0, err
		}
		LastMessage, err := db.GetLastMessage(conv.ID)
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		Expiry, err := db.ConversationExpiry(conv.ID)
		if err == nil {
			conv.Expiry = &Expiry
		}
		conversations = append(conversations, conv)
	}
	return len(conversations), nil
}

//PrunableConversations returns all the conversations whose expiry is in the past and yet haven't finished yet.
func (db *DB) PrunableConversations() (conversations []gp.ConversationID, err error) {
	q := "SELECT conversation_id FROM conversation_expirations WHERE expiry < NOW() AND ended = 0"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query()
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var c gp.ConversationID
		err = rows.Scan(&c)
		if err != nil {
			return
		}
		conversations = append(conversations, c)
	}
	return conversations, nil
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
	q := "SELECT COUNT(*) FROM chat_messages WHERE conversation_id = ? AND EXISTS (SELECT last_read FROM conversation_participants WHERE conversation_id = ? AND participant_id = ?) AND id > (SELECT last_read FROM conversation_participants WHERE conversation_id = ? AND participant_id = ?)"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(convID, convID, userID, convID, userID).Scan(&unread)
	return
}
