//Package db contains everything to do with accessing the database.
//it's dependent on mysql-specific features (REPLACE INTO).
//As well as a prepared statement cache which arose more or less accidentally, but which will be useful for stats later.
package db

import (
	"database/sql"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/go-sql-driver/mysql"
)

const (
	//For parsing
	mysqlTime = "2006-01-02 15:04:05"
)

var (
	//UserAlreadyExists appens when creating an account with a dupe email address or username.
	UserAlreadyExists = gp.APIerror{Reason: "Username or email address already taken"}
)

//DB contains the database configuration and so forth.
type DB struct {
	stmt     map[string]*sql.Stmt
	database *sql.DB
	config   conf.MysqlConfig
}

//New creates a DB; it connects an underlying sql.db and will fatalf if it can't.
func New(conf conf.MysqlConfig) (db *DB) {
	var err error
	db = new(DB)
	db.database, err = sql.Open("mysql", conf.ConnectionString())
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	db.database.SetMaxIdleConns(conf.MaxConns)
	db.stmt = make(map[string]*sql.Stmt)
	return db
}

//prepare wraps sql.DB.Prepare, storing prepared statements in a map.
func (db *DB) prepare(statement string) (stmt *sql.Stmt, err error) {
	stmt, ok := db.stmt[statement]
	if ok {
		return
	}
	stmt, err = db.database.Prepare(statement)
	if err == nil {
		db.stmt[statement] = stmt
	}
	return
}

/********************************************************************
		Database functions
********************************************************************/

/********************************************************************
		User
********************************************************************/

//RegisterUser creates a user with a username (todo:remove), a password hash and an email address.
//They'll be created in an unverified state, and without a full name (which needs to be set separately).
func (db *DB) RegisterUser(user string, hash []byte, email string) (gp.UserID, error) {
	s, err := db.prepare("INSERT INTO users(name, password, email) VALUES (?,?,?)")
	if err != nil {
		return 0, err
	}
	res, err := s.Exec(user, hash, email)
	if err != nil {
		if err, ok := err.(*mysql.MySQLError); ok {
			if err.Number == 1062 {
				return 0, UserAlreadyExists
			}
		}
		return 0, err
	}
	id, _ := res.LastInsertId()
	return gp.UserID(id), nil
}

//SetUserName sets a user's real name.
func (db *DB) SetUserName(id gp.UserID, firstName, lastName string) (err error) {
	s, err := db.prepare("UPDATE users SET firstname = ?, lastname = ? where id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(firstName, lastName, id)
	return
}

//UserChangeTagline sets this user's tagline (obviously enough)
func (db *DB) UserChangeTagline(userID gp.UserID, tagline string) (err error) {
	s, err := db.prepare("UPDATE users SET `desc` = ? WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(tagline, userID)
	return
}

//GetHash returns this user's password hash (by username).
func (db *DB) GetHash(user string) (hash []byte, id gp.UserID, err error) {
	s, err := db.prepare("SELECT id, password FROM users WHERE email = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(user).Scan(&id, &hash)
	return
}

//GetHashByID returns this user's password hash.
func (db *DB) GetHashByID(id gp.UserID) (hash []byte, err error) {
	s, err := db.prepare("SELECT password FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&hash)
	return
}

//PassUpdate replaces this user's password hash with a new one.
func (db *DB) PassUpdate(id gp.UserID, newHash []byte) (err error) {
	s, err := db.prepare("UPDATE users SET password = ? WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(newHash, id)
	return
}

//GetUser returns a gp.User with User.Name set to their firstname if available (username if not).
func (db *DB) GetUser(id gp.UserID) (user gp.User, err error) {
	var av, firstName sql.NullString
	s, err := db.prepare("SELECT id, name, avatar, firstname FROM users WHERE id=?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&user.ID, &user.Name, &av, &firstName)
	log.Println("DB hit: db.GetUser id(user.Name, user.Id, user.Avatar)")
	if err != nil {
		if err == sql.ErrNoRows {
			err = &gp.ENOSUCHUSER
		}
		return
	}
	if av.Valid {
		user.Avatar = av.String
	}
	if firstName.Valid {
		user.Name = firstName.String
	}
	return
}

//GetProfile fetches a user but DOES NOT GET THEIR NETWORK.
func (db *DB) GetProfile(id gp.UserID) (user gp.Profile, err error) {
	var av, desc, firstName, lastName sql.NullString
	s, err := db.prepare("SELECT name, `desc`, avatar, firstname, lastname FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&user.Name, &desc, &av, &firstName, &lastName)
	log.Println("DB hit: getProfile id(user.Name, user.Desc)")
	if err != nil {
		if err == sql.ErrNoRows {
			return user, &gp.ENOSUCHUSER
		}
		return
	}
	if av.Valid {
		user.Avatar = av.String
	}
	if desc.Valid {
		user.Desc = desc.String
	}
	if firstName.Valid {
		user.Name = firstName.String
	}
	if lastName.Valid {
		user.FullName = firstName.String + " " + lastName.String
	}
	user.ID = id
	return
}

//SetProfileImage updates this user's avatar to url.
func (db *DB) SetProfileImage(id gp.UserID, url string) (err error) {
	s, err := db.prepare("UPDATE users SET avatar = ? WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(url, id)
	return
}

//SetBusyStatus records whether this user is busy or not.
func (db *DB) SetBusyStatus(id gp.UserID, busy bool) (err error) {
	s, err := db.prepare("UPDATE users SET busy = ? WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(busy, id)
	return
}

//BusyStatus returns this user's busy status.
func (db *DB) BusyStatus(id gp.UserID) (busy bool, err error) {
	s, err := db.prepare("SELECT busy FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&busy)
	return
}

//UserIDFromFB gets the gleepost user who has fbid associated, or an error if there is none.
func (db *DB) UserIDFromFB(fbid uint64) (id gp.UserID, err error) {
	s, err := db.prepare("SELECT user_id FROM facebook WHERE fb_id = ? AND user_id IS NOT NULL")
	if err != nil {
		return
	}
	err = s.QueryRow(fbid).Scan(&id)
	return
}

//TODO: return ENOSUCHUSER instead.

//SetVerificationToken records a (hopefully random!) verification token for this user.
func (db *DB) SetVerificationToken(id gp.UserID, token string) (err error) {
	s, err := db.prepare("REPLACE INTO `verification` (user_id, token) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(id, token)
	return
}

//VerificationTokenExists returns the user who this verification token belongs to, or an error if there isn't one.
func (db *DB) VerificationTokenExists(token string) (id gp.UserID, err error) {
	s, err := db.prepare("SELECT user_id FROM verification WHERE token = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(token).Scan(&id)
	return
}

//Verify marks a user as verified.
func (db *DB) Verify(id gp.UserID) (err error) {
	s, err := db.prepare("UPDATE users SET verified = 1 WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(id)
	return
}

//IsVerified returns true if this user is verified.
func (db *DB) IsVerified(user gp.UserID) (verified bool, err error) {
	s, err := db.prepare("SELECT verified FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(user).Scan(&verified)
	return
}

//GetEmail returns this user's email address.
func (db *DB) GetEmail(id gp.UserID) (email string, err error) {
	s, err := db.prepare("SELECT email FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&email)
	return
}

//UserWithEmail returns the user whose email this is, or an error if they don't exist.
func (db *DB) UserWithEmail(email string) (id gp.UserID, err error) {
	s, err := db.prepare("SELECT id FROM users WHERE email = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(email).Scan(&id)
	return
}

//CreateFBUser records the existence of this (fbid:email) pair; when the user is verified it will be converted to a full gleepost user.
func (db *DB) CreateFBUser(fbID uint64, email string) (err error) {
	s, err := db.prepare("INSERT INTO facebook (fb_id, email) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(fbID, email)
	return
}

//FBUserEmail returns this facebook user's email address.
func (db *DB) FBUserEmail(fbid uint64) (email string, err error) {
	s, err := db.prepare("SELECT email FROM facebook WHERE fb_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(fbid).Scan(&email)
	return
}

//FBUserWithEmail returns the facebook id we've seen associated with this email, or error if none exists.
func (db *DB) FBUserWithEmail(email string) (fbid uint64, err error) {
	s, err := db.prepare("SELECT fb_id FROM facebook WHERE email = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(email).Scan(&fbid)
	return
}

//CreateFBVerification records a (hopefully random!) verification token for this facebook user.
func (db *DB) CreateFBVerification(fbid uint64, token string) (err error) {
	s, err := db.prepare("REPLACE INTO facebook_verification (fb_id, token) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(fbid, token)
	return
}

//FBVerificationExists returns the user this verification token is for, or an error if there is none.
func (db *DB) FBVerificationExists(token string) (fbid uint64, err error) {
	s, err := db.prepare("SELECT fb_id FROM facebook_verification WHERE token = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(token).Scan(&fbid)
	return
}

//FBSetGPUser records the association of this facebook user with this gleepost user.
//After this, the user should be able to log in with this facebook account.
func (db *DB) FBSetGPUser(fbid uint64, userID gp.UserID) (err error) {
	fbSetGPUser := "REPLACE INTO facebook (user_id, fb_id) VALUES (?, ?)"
	stmt, err := db.prepare(fbSetGPUser)
	if err != nil {
		return
	}
	res, err := stmt.Exec(userID, fbid)
	log.Println(res.RowsAffected())
	return
}

//AddPasswordRecovery records a password recovery token for this user.
func (db *DB) AddPasswordRecovery(userID gp.UserID, token string) (err error) {
	s, err := db.prepare("REPLACE INTO password_recovery (token, user) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(token, userID)
	return
}

//CheckPasswordRecovery returns true if this password recovery user:token pair exists.
func (db *DB) CheckPasswordRecovery(userID gp.UserID, token string) (exists bool, err error) {
	s, err := db.prepare("SELECT count(*) FROM password_recovery WHERE user = ? and token = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(userID, token).Scan(&exists)
	return
}

//DeletePasswordRecovery removes this password recovery token so it can't be used again.
func (db *DB) DeletePasswordRecovery(userID gp.UserID, token string) (err error) {
	s, err := db.prepare("DELETE FROM password_recovery WHERE user = ? and token = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(userID, token)
	return
}

/********************************************************************
		Conversation
********************************************************************/

//GetLiveConversations returns the three most recent unfinished live conversations for a given user.
//TODO: retrieve conversation & expiry in a single query
func (db *DB) GetLiveConversations(id gp.UserID) (conversations gp.ConversationSmallList, err error) {
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
	q := "SELECT DISTINCT id, name, firstname, avatar " +
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
		var first sql.NullString
		err = rows.Scan(&user.ID, &user.Name, &first, &av)
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
			if first.Valid {
				user.Name = first.String
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
func (db *DB) GetConversations(userID gp.UserID, start int64, count int, all bool) (conversations gp.ConversationSmallList, err error) {
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
func (db *DB) GetConversation(convID gp.ConversationID, count int) (conversation gp.ConversationAndMessages, err error) {
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

/********************************************************************
		Message
********************************************************************/

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
func (db *DB) GetMessages(convID gp.ConversationID, index int64, sel string, count int) (messages gp.MessageList, err error) {
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

//MarkRead will set all messages in the conversation convId read = true
//up to and including upTo and excluding messages sent by user id.
//TODO: This won't generalize to >2 participants
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

//AddCategory marks the post id as a member of category.
func (db *DB) AddCategory(id gp.PostID, category gp.CategoryID) (err error) {
	s, err := db.prepare("INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(id, category)
	return
}

//CategoryList returns all existing categories.
func (db *DB) CategoryList() (categories []gp.PostCategory, err error) {
	s, err := db.prepare("SELECT id, tag, name FROM categories WHERE 1")
	if err != nil {
		return
	}
	rows, err := s.Query()
	defer rows.Close()
	for rows.Next() {
		c := gp.PostCategory{}
		err = rows.Scan(&c.ID, &c.Tag, &c.Name)
		if err != nil {
			return
		}
		categories = append(categories, c)
	}
	return
}

//TagPost accepts a post id and any number of string tags. Any of the tags that exist will be added to the post.
func (db *DB) TagPost(post gp.PostID, tags ...string) (err error) {
	s, err := db.prepare("INSERT INTO post_categories( post_id, category_id ) SELECT ? , categories.id FROM categories WHERE categories.tag = ?")
	if err != nil {
		return
	}
	for _, tag := range tags {
		_, err = s.Exec(post, tag)
		if err != nil {
			return
		}
	}
	return
}

//PostCategories returns all the categories which post belongs to.
func (db *DB) PostCategories(post gp.PostID) (categories []gp.PostCategory, err error) {
	s, err := db.prepare("SELECT categories.id, categories.tag, categories.name FROM post_categories JOIN categories ON post_categories.category_id = categories.id WHERE post_categories.post_id = ?")
	if err != nil {
		return
	}
	rows, err := s.Query(post)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		c := gp.PostCategory{}
		err = rows.Scan(&c.ID, &c.Tag, &c.Name)
		if err != nil {
			return
		}
		categories = append(categories, c)
	}
	return
}

/********************************************************************
		Token
********************************************************************/

//TokenExists returns true if this user:token pair exists, false otherwise (or in the case of error)
func (db *DB) TokenExists(id gp.UserID, token string) bool {
	var expiry string
	s, err := db.prepare("SELECT expiry FROM tokens WHERE user_id = ? AND token = ?")
	if err != nil {
		return false
	}
	err = s.QueryRow(id, token).Scan(&expiry)
	if err != nil {
		return false
	}
	t, _ := time.Parse(mysqlTime, expiry)
	if t.After(time.Now()) {
		return (true)
	}
	return (false)
}

//AddToken records this session token in the database.
func (db *DB) AddToken(token gp.Token) (err error) {
	s, err := db.prepare("INSERT INTO tokens (user_id, token, expiry) VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(token.UserID, token.Token, token.Expiry)
	return
}

/********************************************************************
		Contact
********************************************************************/

//AddContact records that adder has added addee as a contact.
func (db *DB) AddContact(adder gp.UserID, addee gp.UserID) (err error) {
	log.Println("DB hit: addContact")
	s, err := db.prepare("INSERT INTO contacts (adder, addee) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(adder, addee)
	return
}

//GetContacts retrieves all the contacts for user.
//TODO: This could return contacts which doesn't embed a user
func (db *DB) GetContacts(user gp.UserID) (contacts gp.ContactList, err error) {
	s, err := db.prepare("SELECT adder, addee, confirmed FROM contacts WHERE adder = ? OR addee = ? ORDER BY time DESC")
	if err != nil {
		return
	}
	rows, err := s.Query(user, user)
	log.Println("DB hit: db.GetContacts")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var contact gp.Contact
		var adder, addee gp.UserID
		var confirmed bool
		err = rows.Scan(&adder, &addee, &confirmed)
		if err != nil {
			return
		}
		switch {
		case adder == user:
			contact.User, err = db.GetUser(addee)
			if err == nil {
				contact.YouConfirmed = true
				contact.TheyConfirmed = confirmed
				contacts = append(contacts, contact)
			} else {
				log.Println(err)
			}
		case addee == user:
			contact.User, err = db.GetUser(adder)
			if err == nil {
				contact.YouConfirmed = confirmed
				contact.TheyConfirmed = true
				contacts = append(contacts, contact)
			} else {
				log.Println(err)
			}
		}
	}
	return contacts, nil
}

//UpdateContact marks this adder/addee pair as "accepted"
func (db *DB) UpdateContact(user gp.UserID, contact gp.UserID) (err error) {
	s, err := db.prepare("UPDATE contacts SET confirmed = 1 WHERE addee = ? AND adder = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(user, contact)
	return
}

//ContactRequestExists returns true if this adder has already added addee.
func (db *DB) ContactRequestExists(adder gp.UserID, addee gp.UserID) (exists bool, err error) {
	s, err := db.prepare("SELECT COUNT(*) FROM contacts WHERE adder = ? AND addee = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(adder, addee).Scan(&exists)
	return
}

/********************************************************************
		Notification
********************************************************************/

//GetUserNotifications returns all the notifications for a given user, optionally including the seen ones.
func (db *DB) GetUserNotifications(id gp.UserID, includeSeen bool) (notifications gp.NotificationList, err error) {
	var notificationSelect string
	if !includeSeen {
		notificationSelect = "SELECT id, type, time, `by`, location_id, seen FROM notifications WHERE recipient = ? AND seen = 0 ORDER BY `id` DESC"
	} else {
		notificationSelect = "SELECT id, type, time, `by`, location_id, seen FROM notifications WHERE recipient = ? ORDER BY `id` DESC LIMIT 0, 20"
	}
	s, err := db.prepare(notificationSelect)
	if err != nil {
		return
	}
	rows, err := s.Query(id)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var notification gp.Notification
		var t string
		var location sql.NullInt64
		var by gp.UserID
		if err = rows.Scan(&notification.ID, &notification.Type, &t, &by, &location, &notification.Seen); err != nil {
			return
		}
		notification.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return
		}
		notification.By, err = db.GetUser(by)
		if err != nil {
			log.Println(err)
			continue
		}
		if location.Valid {
			switch {
			case notification.Type == "liked":
				fallthrough
			case notification.Type == "commented":
				np := gp.PostNotification{Notification: notification, Post: gp.PostID(location.Int64)}
				notifications = append(notifications, np)
			case notification.Type == "group_post":
				fallthrough
			case notification.Type == "added_group":
				ng := gp.GroupNotification{Notification: notification, Group: gp.NetworkID(location.Int64)}
				notifications = append(notifications, ng)
			default:
				notifications = append(notifications, notification)
			}
		} else {
			notifications = append(notifications, notification)
		}
	}
	return
}

//MarkNotificationsSeen records that this user has seen all their notifications.
func (db *DB) MarkNotificationsSeen(user gp.UserID, upTo gp.NotificationID) (err error) {
	s, err := db.prepare("UPDATE notifications SET seen = 1 WHERE recipient = ? AND id <= ?")
	if err != nil {
		return
	}
	_, err = s.Exec(user, upTo)
	return
}

//CreateNotification creates a notification ntype for recipient, "from" by, with a location which is interpreted as a post id if ntype is like/comment.
//TODO: All this stuff should not be in the db layer.
func (db *DB) CreateNotification(ntype string, by gp.UserID, recipient gp.UserID, location uint64) (notification interface{}, err error) {
	var res sql.Result
	notificationInsert := "INSERT INTO notifications (type, time, `by`, recipient) VALUES (?, NOW(), ?, ?)"
	notificationInsertLocation := "INSERT INTO notifications (type, time, `by`, recipient, location_id) VALUES (?, NOW(), ?, ?, ?)"
	var s *sql.Stmt
	n := gp.Notification{
		Type: ntype,
		Time: time.Now().UTC(),
		Seen: false,
	}
	n.By, err = db.GetUser(by)
	if err != nil {
		return
	}
	switch {
	case ntype == "liked":
		fallthrough
	case ntype == "commented":
		fallthrough
	case ntype == "group_post":
		fallthrough
	case ntype == "added_group":
		s, err = db.prepare(notificationInsertLocation)
		if err != nil {
			return
		}
		res, err = s.Exec(ntype, by, recipient, location)
	default:
		s, err = db.prepare(notificationInsert)
		if err != nil {
			return
		}
		res, err = s.Exec(ntype, by, recipient)
	}
	if err != nil {
		return
	}
	id, iderr := res.LastInsertId()
	if iderr != nil {
		return n, iderr
	}
	n.ID = gp.NotificationID(id)
	switch {
	case ntype == "liked":
		fallthrough
	case ntype == "commented":
		np := gp.PostNotification{Notification: n, Post: gp.PostID(location)}
		return np, nil
	case ntype == "group_post":
		fallthrough
	case ntype == "added_group":
		ng := gp.GroupNotification{Notification: n, Group: gp.NetworkID(location)}
		return ng, nil
	default:
		return n, nil
	}
}

//CreateLike records that this user has liked this post. Acts idempotently.
func (db *DB) CreateLike(user gp.UserID, post gp.PostID) (err error) {
	s, err := db.prepare("REPLACE INTO post_likes (post_id, user_id) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(post, user)
	return
}

//RemoveLike un-likes a post.
func (db *DB) RemoveLike(user gp.UserID, post gp.PostID) (err error) {
	s, err := db.prepare("DELETE FROM post_likes WHERE post_id = ? AND user_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(post, user)
	return
}

//GetLikes returns all this post's likes
func (db *DB) GetLikes(post gp.PostID) (likes []gp.Like, err error) {
	s, err := db.prepare("SELECT user_id, timestamp FROM post_likes WHERE post_id = ?")
	if err != nil {
		return
	}
	rows, err := s.Query(post)
	defer rows.Close()
	if err != nil {
		return
	}
	for rows.Next() {
		var t string
		var like gp.Like
		err = rows.Scan(&like.UserID, &t)
		if err != nil {
			return
		}
		like.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return
		}
		likes = append(likes, like)
	}
	return
}

//HasLiked retuns true if this user has already liked this post.
func (db *DB) HasLiked(user gp.UserID, post gp.PostID) (liked bool, err error) {
	s, err := db.prepare("SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND user_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(post, user).Scan(&liked)
	return
}

//LikeCount returns the number of likes this post has.
func (db *DB) LikeCount(post gp.PostID) (count int, err error) {
	s, err := db.prepare("SELECT COUNT(*) FROM post_likes WHERE post_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(post).Scan(&count)
	return
}

//Attend adds the user to the "attending" list for this event. It's idempotent, and should only return an error if the database is down.
//The results are undefined for a post which isn't an event.
//(ie: it will work even though it shouldn't, until I can get round to enforcing it.)
func (db *DB) Attend(event gp.PostID, user gp.UserID) (err error) {
	query := "REPLACE INTO event_attendees (post_id, user_id) VALUES (?, ?)"
	s, err := db.prepare(query)
	if err != nil {
		return
	}
	_, err = s.Exec(event, user)
	return
}

//UnAttend removes a user's attendance to an event. Idempotent, returns an error if the DB is down.
func (db *DB) UnAttend(event gp.PostID, user gp.UserID) (err error) {
	query := "DELETE FROM event_attendees WHERE post_id = ? AND user_id = ?"
	s, err := db.prepare(query)
	if err != nil {
		return
	}
	_, err = s.Exec(event, user)
	return
}

//UserAttends returns all the event IDs that a user is attending.
func (db *DB) UserAttends(user gp.UserID) (events gp.PostIDList, err error) {
	query := "SELECT post_id FROM event_attendees WHERE user_id = ?"
	s, err := db.prepare(query)
	if err != nil {
		return
	}
	rows, err := s.Query(user)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var post gp.PostID
		err = rows.Scan(&post)
		if err != nil {
			return
		}
		events = append(events, post)
	}
	return
}

//UserAttending returns all the events this user is attending.
func (db *DB) UserAttending(perspective, user gp.UserID, category string, mode int, index int64, count int) (events gp.PostSmallList, err error) {
	where := WhereClause{Mode: WATTENDS, User: user, Perspective: perspective, Category: category}
	log.Println("Where:", where)
	return db.NewGetPosts(where, mode, index, count)
}

//UnreadMessageCount returns the number of unread messages this user has.
func (db *DB) UnreadMessageCount(user gp.UserID) (count int, err error) {
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
		err = sUnreadCount.QueryRow(convID, lastID).Scan(&_count)
		if err == nil {
			log.Printf("Conversation %d, unread message count was %d\n", convID, _count)
			count += _count
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

//SubjectiveRSVPCount shows the number of events otherID has attended, from the perspective of the `perspective` user (ie, not counting those events perspective can't see...)
func (db *DB) SubjectiveRSVPCount(perspective gp.UserID, otherID gp.UserID) (count int, err error) {
	q := "SELECT COUNT(*) FROM event_attendees JOIN wall_posts ON event_attendees.post_id = wall_posts.id "
	q += "WHERE wall_posts.network_id IN ( SELECT network_id FROM user_network WHERE user_network.user_id = ? ) "
	q += "AND wall_posts.deleted = 0 "
	q += "AND event_attendees.user_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(otherID, perspective).Scan(&count)
	return
}
