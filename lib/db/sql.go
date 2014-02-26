//db handles
package db

import (
	"database/sql"
	"github.com/draaglom/GleepostAPI/lib/gp"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	//For parsing
	mysqlTime = "2006-01-02 15:04:05"
)

var (
	sqlStmt map[string]string
)

type DB struct {
	stmt     map[string]*sql.Stmt
	database *sql.DB
	config   gp.MysqlConfig
}

func New(conf gp.MysqlConfig) (db *DB) {
	var err error
	db = new(DB)
	db.database, err = sql.Open("mysql", conf.ConnectionString())
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	db.database.SetMaxIdleConns(conf.MaxConns)
	db.stmt, err = prepare(db.database)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

//prepare wraps sql.DB.Prepare, storing prepared statements in a map.
func (db *DB) prepare(statement string) (stmt *sql.Stmt, err error) {
	stmt, ok := db.stmt[statement]
	if ok {
		return
	} else {
		stmt, err = db.database.Prepare(statement)
		if err == nil {
			db.stmt[statement] = stmt
		}
		return
	}
}

//why.png
func prepare(db *sql.DB) (stmt map[string]*sql.Stmt, err error) {
	sqlStmt = make(map[string]string)
	stmt = make(map[string]*sql.Stmt)
	//User
	sqlStmt["createUser"] = "INSERT INTO users(name, password, email) VALUES (?,?,?)"
	sqlStmt["setName"] = "UPDATE users SET firstname = ?, lastname = ? where id = ?"
	sqlStmt["userSelect"] = "SELECT id, name, avatar, firstname FROM users WHERE id=?"
	sqlStmt["profileSelect"] = "SELECT name, `desc`, avatar, firstname, lastname FROM users WHERE id = ?"
	sqlStmt["passSelect"] = "SELECT id, password FROM users WHERE email = ?"
	sqlStmt["hashById"] = "SELECT password FROM users WHERE id = ?"
	sqlStmt["passUpdate"] = "UPDATE users SET password = ? WHERE id = ?"
	sqlStmt["randomSelect"] = "SELECT id, name, firstname, avatar " +
		"FROM users " +
		"LEFT JOIN user_network ON id = user_id " +
		"WHERE network_id = ? " +
		"AND verified = 1 " +
		"ORDER BY RAND()"
	sqlStmt["setAvatar"] = "UPDATE users SET avatar = ? WHERE id = ?"
	sqlStmt["setBusy"] = "UPDATE users SET busy = ? WHERE id = ?"
	sqlStmt["getBusy"] = "SELECT busy FROM users WHERE id = ?"
	sqlStmt["idFromFacebook"] = "SELECT user_id FROM facebook WHERE fb_id = ? AND user_id IS NOT NULL"
	sqlStmt["fbInsert"] = "INSERT INTO facebook (fb_id, email) VALUES (?, ?)"
	sqlStmt["selectFBemail"] = "SELECT email FROM facebook WHERE fb_id = ?"
	sqlStmt["fbUserByEmail"] = "SELECT fb_id FROM facebook WHERE email = ?"
	sqlStmt["fbInsertVerification"] = "REPLACE INTO facebook_verification (fb_id, token) VALUES (?, ?)"
	sqlStmt["fbSetGPUser"] = "UPDATE facebook SET user_id = ? WHERE fb_id = ?"
	sqlStmt["insertVerification"] = "REPLACE INTO `verification` (user_id, token) VALUES (?, ?)"
	sqlStmt["verificationExists"] = "SELECT user_id FROM verification WHERE token = ?"
	sqlStmt["verify"] = "UPDATE users SET verified = 1 WHERE id = ?"
	sqlStmt["userIsVerified"] = "SELECT verified FROM users WHERE id = ?"
	sqlStmt["emailSelect"] = "SELECT email FROM users WHERE id = ?"
	sqlStmt["userWithEmail"] = "SELECT id FROM users WHERE email = ?"
	sqlStmt["addPasswordRecovery"] = "REPLACE INTO password_recovery (token, user) VALUES (?, ?)"
	sqlStmt["checkPasswordRecovery"] = "SELECT count(*) FROM password_recovery WHERE user = ? and token = ?"
	sqlStmt["deletePasswordRecovery"] = "DELETE FROM password_recovery WHERE user = ? and token = ?"
	//Conversation
	sqlStmt["conversationInsert"] = "INSERT INTO conversations (initiator, last_mod) VALUES (?, NOW())"
	sqlStmt["conversationUpdate"] = "UPDATE conversations SET last_mod = NOW() WHERE id = ?"
	sqlStmt["conversationSelect"] = "SELECT conversation_participants.conversation_id, conversations.last_mod " +
		"FROM conversation_participants " +
		"JOIN conversations ON conversation_participants.conversation_id = conversations.id " +
		"LEFT OUTER JOIN conversation_expirations ON conversation_expirations.conversation_id = conversations.id " +
		"WHERE participant_id = ? AND ( " +
			"conversation_expirations.ended IS NULL " +
			"OR conversation_expirations.ended =0 " +
		") " +
		"ORDER BY conversations.last_mod DESC LIMIT ?, ?"
	sqlStmt["conversationsAll"] = "SELECT conversation_participants.conversation_id, conversations.last_mod " +
		"FROM conversation_participants " +
		"JOIN conversations ON conversation_participants.conversation_id = conversations.id " +
		"LEFT OUTER JOIN conversation_expirations ON conversation_expirations.conversation_id = conversations.id " +
		"WHERE participant_id = ? " +
		"ORDER BY conversations.last_mod DESC LIMIT ?, ?"
	sqlStmt["conversationActivity"] = "SELECT last_mod FROM conversations WHERE id = ?"
	sqlStmt["conversationExpiry"] = "SELECT expiry, ended FROM conversation_expirations WHERE conversation_id = ?"
	sqlStmt["conversationSetExpiry"] = "REPLACE INTO conversation_expirations (conversation_id, expiry) VALUES (?, ?)"
	sqlStmt["deleteExpiry"] = "DELETE FROM conversation_expirations WHERE conversation_id = ?"
	sqlStmt["endConversation"] = "UPDATE conversation_expirations SET ended = 1 WHERE conversation_id = ?"
	sqlStmt["participantInsert"] = "INSERT INTO conversation_participants (conversation_id, participant_id) VALUES (?,?)"
	sqlStmt["participantSelect"] = "SELECT participant_id " +
		"FROM conversation_participants " +
		"JOIN users ON conversation_participants.participant_id = users.id " +
		"WHERE conversation_id=?"
	sqlStmt["lastMessageSelect"] = "SELECT id, `from`, text, `timestamp`" +
		"FROM chat_messages " +
		"WHERE conversation_id = ? " +
		"ORDER BY `timestamp` DESC LIMIT 1"
	sqlStmt["liveConversations"] = "SELECT conversation_participants.conversation_id, conversations.last_mod " +
		"FROM conversation_participants " +
		"JOIN conversations ON conversation_participants.conversation_id = conversations.id " +
		"JOIN conversation_expirations ON conversation_expirations.conversation_id = conversations.id " +
		"WHERE participant_id = ? " +
		"AND conversation_expirations.ended = 0 " +
		"ORDER BY conversations.last_mod DESC  " +
		"LIMIT 0 , 3"
	sqlStmt["readStatus"] = "SELECT participant_id, last_read FROM conversation_participants WHERE conversation_id = ?"
	//Post
	sqlStmt["postInsert"] = "INSERT INTO wall_posts(`by`, `text`, network_id) VALUES (?,?,?)"
	sqlStmt["wallSelect"] = "SELECT id, `by`, time, text FROM wall_posts WHERE network_id = ? ORDER BY time DESC LIMIT ?, ?"
	sqlStmt["wallSelectCategory"] = "SELECT wall_posts.id, `by`, time, text " +
		"FROM wall_posts " +
		"JOIN post_categories ON wall_posts.id = post_categories.post_id " +
		"JOIN categories ON post_categories.category_id = categories.id " +
		"WHERE network_id = ? AND categories.tag = ? " +
		"ORDER BY time DESC LIMIT ?, ?"
	sqlStmt["wallSelectAfter"] = "SELECT id, `by`, time, text " +
		"FROM wall_posts " +
		"WHERE network_id = ? AND id > ? " +
		"ORDER BY time DESC LIMIT 0, ?"
	sqlStmt["wallSelectCategoryAfter"] = "SELECT wall_posts.id, `by`, time, text " +
		"FROM wall_posts " +
		"JOIN post_categories ON wall_posts.id = post_categories.post_id " +
		"JOIN categories ON post_categories.category_id = categories.id " +
		"WHERE network_id = ? AND categories.tag = ? AND wall_posts.id > ? " +
		"ORDER BY time DESC LIMIT 0, ?"
	sqlStmt["wallSelectBefore"] = "SELECT id, `by`, time, text " +
		"FROM wall_posts " +
		"WHERE network_id = ? AND id < ? " +
		"ORDER BY time DESC LIMIT 0, ?"
	sqlStmt["wallSelectCategoryBefore"] = "SELECT wall_posts.id, `by`, time, text " +
		"FROM wall_posts " +
		"JOIN post_categories ON wall_posts.id = post_categories.post_id " +
		"JOIN categories ON post_categories.category_id = categories.id " +
		"WHERE network_id = ? AND categories.tag = ? AND wall_posts.id < ? " +
		"ORDER BY time DESC LIMIT 0, ?"
	sqlStmt["liveSelect"] = "SELECT wall_posts.id, `by`, time, text " +
		"FROM wall_posts " +
		"JOIN post_attribs ON wall_posts.id = post_attribs.post_id " +
		"WHERE network_id = ? AND attrib = 'event-time' AND value > ? " +
		"ORDER BY value ASC LIMIT 0, ?"
	sqlStmt["imageSelect"] = "SELECT url FROM post_images WHERE post_id = ?"
	sqlStmt["imageInsert"] = "INSERT INTO post_images (post_id, url) VALUES (?, ?)"
	sqlStmt["commentInsert"] = "INSERT INTO post_comments (post_id, `by`, text) VALUES (?, ?, ?)"
	sqlStmt["commentSelect"] = "SELECT id, `by`, text, `timestamp` " +
		"FROM post_comments " +
		"WHERE post_id = ? " +
		"ORDER BY `timestamp` DESC LIMIT ?, ?"
	sqlStmt["commentCountSelect"] = "SELECT COUNT(*) FROM post_comments WHERE post_id = ?"
	sqlStmt["postSelect"] = "SELECT `by`, `time`, text FROM wall_posts WHERE id = ?"
	sqlStmt["categoryAdd"] = "INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)"
	sqlStmt["addCategoryWhereExists"] = "INSERT INTO post_categories( post_id, category_id ) SELECT ? , categories.id FROM categories WHERE categories.tag = ?"
	sqlStmt["listCategories"] = "SELECT id, tag, name FROM categories WHERE 1"
	sqlStmt["postCategories"] = "SELECT categories.id, categories.tag, categories.name FROM post_categories JOIN categories ON post_categories.category_id = categories.id WHERE post_categories.post_id = ?"
	sqlStmt["setPostAttribs"] = "REPLACE INTO post_attribs (post_id, attrib, value) VALUES (?, ?, ?)"
	sqlStmt["getPostAttribs"] = "SELECT attrib, value FROM post_attribs WHERE post_id=?"
	//Message
	sqlStmt["messageInsert"] = "INSERT INTO chat_messages (conversation_id, `from`, `text`) VALUES (?,?,?)"
	sqlStmt["messageSelect"] = "SELECT id, `from`, text, `timestamp`" +
		"FROM chat_messages " +
		"WHERE conversation_id = ? " +
		"ORDER BY `timestamp` DESC LIMIT ?, ?"
	sqlStmt["messageSelectAfter"] = "SELECT id, `from`, text, `timestamp`" +
		"FROM chat_messages " +
		"WHERE conversation_id = ? AND id > ? " +
		"ORDER BY `timestamp` DESC LIMIT ?"
	sqlStmt["messageSelectBefore"] = "SELECT id, `from`, text, `timestamp`" +
		"FROM chat_messages " +
		"WHERE conversation_id = ? AND id < ? " +
		"ORDER BY `timestamp` DESC LIMIT ?"
	sqlStmt["messagesRead"] = "UPDATE conversation_participants " +
		"SET last_read = ? " +
		"WHERE `conversation_id` = ? AND `participant_id` = ?"
	//Token
	sqlStmt["tokenInsert"] = "INSERT INTO tokens (user_id, token, expiry) VALUES (?, ?, ?)"
	sqlStmt["tokenSelect"] = "SELECT expiry FROM tokens WHERE user_id = ? AND token = ?"
	//Contact
	sqlStmt["contactInsert"] = "INSERT INTO contacts (adder, addee) VALUES (?, ?)"
	sqlStmt["contactSelect"] = "SELECT adder, addee, confirmed FROM contacts WHERE adder = ? OR addee = ? ORDER BY time DESC"
	sqlStmt["contactUpdate"] = "UPDATE contacts SET confirmed = 1 WHERE addee = ? AND adder = ?"
	sqlStmt["contactExists"] = "SELECT COUNT(*) FROM contacts WHERE adder = ? AND addee = ?"
	//device
	sqlStmt["deviceInsert"] = "REPLACE INTO devices (user_id, device_type, device_id) VALUES (?, ?, ?)"
	sqlStmt["deviceSelect"] = "SELECT user_id, device_type, device_id FROM devices WHERE user_id = ?"
	sqlStmt["deviceDelete"] = "DELETE FROM devices WHERE user_id = ? AND device_id = ?"
	sqlStmt["feedbackDelete"] = "DELETE FROM devices WHERE device_id = ? AND last_update < ? AND device_type = 'ios'"
	//Upload
	sqlStmt["userUpload"] = "INSERT INTO uploads (user_id, url) VALUES (?, ?)"
	sqlStmt["uploadExists"] = "SELECT COUNT(*) FROM uploads WHERE user_id = ? AND url = ?"
	//Notification
	sqlStmt["notificationSelect"] = "SELECT id, type, time, `by`, post_id, seen FROM notifications WHERE recipient = ? AND seen = 0"
	sqlStmt["notificationUpdate"] = "UPDATE notifications SET seen = 1 WHERE recipient = ? AND id <= ?"
	sqlStmt["notificationInsert"] = "INSERT INTO notifications (type, time, `by`, recipient) VALUES (?, NOW(), ?, ?)"
	sqlStmt["postNotificationInsert"] = "INSERT INTO notifications (type, time, `by`, recipient, post_id) VALUES (?, NOW(), ?, ?, ?)"
	//Like
	sqlStmt["addLike"] = "INSERT INTO post_likes (post_id, user_id) VALUES (?, ?)"
	sqlStmt["delLike"] = "DELETE FROM post_likes WHERE post_id = ? AND user_id = ?"
	sqlStmt["likeSelect"] = "SELECT user_id, timestamp FROM post_likes WHERE post_id = ?"
	sqlStmt["likeExists"] = "SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND user_id = ?"
	sqlStmt["likeCount"] = "SELECT COUNT(*) FROM post_likes WHERE post_id = ?"
	for k, str := range sqlStmt {
		stmt[k], err = db.Prepare(str)
		if err != nil {
			return
		}
	}
	return stmt, nil
}

/********************************************************************
		Database functions
********************************************************************/

/********************************************************************
		Network
********************************************************************/

func (db *DB) GetRules() (rules []gp.Rule, err error) {
	ruleSelect := "SELECT network_id, rule_type, rule_value FROM net_rules"
	s, err := db.prepare(ruleSelect)
	if err != nil {
		return
	}
	rows, err := s.Query()
	log.Println("DB hit: validateEmail (rule.networkid, rule.type, rule.value)")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var rule gp.Rule
		if err = rows.Scan(&rule.NetworkID, &rule.Type, &rule.Value); err != nil {
			return
		}
		rules = append(rules, rule)
	}
	return
}

func (db *DB) GetUserNetworks(id gp.UserId) (networks []gp.Network, err error) {
	networkSelect :=  "SELECT user_network.network_id, network.name " +
		"FROM user_network " +
		"INNER JOIN network ON user_network.network_id = network.id " +
		"WHERE user_id = ?"
	s, err := db.prepare(networkSelect)
	if err != nil {
		return
	}
	rows, err := s.Query(id)
	defer rows.Close()
	log.Println("DB hit: getUserNetworks userid (network.id, network.name)")
	if err != nil {
		return
	}
	for rows.Next() {
		var network gp.Network
		err = rows.Scan(&network.Id, &network.Name)
		if err != nil {
			return
		} else {
			networks = append(networks, network)
		}
	}
	return
}

func (db *DB) SetNetwork(userId gp.UserId, networkId gp.NetworkId) (err error) {
	networkInsert := "INSERT INTO user_network (user_id, network_id) VALUES (?, ?)"
	s, err := db.prepare(networkInsert)
	if err != nil {
		return
	}
	_, err = s.Exec(userId, networkId)
	return
}

/********************************************************************
		User
********************************************************************/

func (db *DB) RegisterUser(user string, hash []byte, email string) (gp.UserId, error) {
	s := db.stmt["createUser"]
	res, err := s.Exec(user, hash, email)
	if err != nil && strings.HasPrefix(err.Error(), "Error 1062") { //Note to self:There must be a better way?
		return 0, gp.APIerror{"Username or email address already taken"}
	} else if err != nil {
		return 0, err
	} else {
		id, _ := res.LastInsertId()
		return gp.UserId(id), nil
	}
}

func (db *DB) SetUserName(id gp.UserId, firstName, lastName string) (err error) {
	_, err = db.stmt["setName"].Exec(firstName, lastName, id)
	return
}

func (db *DB) GetHash(user string) (hash []byte, id gp.UserId, err error) {
	s := db.stmt["passSelect"]
	err = s.QueryRow(user).Scan(&id, &hash)
	return
}

func (db *DB) GetHashById(id gp.UserId) (hash []byte, err error) {
	err = db.stmt["hashById"].QueryRow(id).Scan(&hash)
	return
}

func (db *DB) PassUpdate(id gp.UserId, newHash []byte) (err error) {
	_, err = db.stmt["passUpdate"].Exec(newHash, id)
	return
}

func (db *DB) GetUser(id gp.UserId) (user gp.User, err error) {
	var av, firstName sql.NullString
	s := db.stmt["userSelect"]
	err = s.QueryRow(id).Scan(&user.Id, &user.Name, &av, &firstName)
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
func (db *DB) GetProfile(id gp.UserId) (user gp.Profile, err error) {
	var av, desc, firstName, lastName sql.NullString
	s := db.stmt["profileSelect"]
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
	user.Id = id
	return
}

func (db *DB) SetProfileImage(id gp.UserId, url string) (err error) {
	_, err = db.stmt["setAvatar"].Exec(url, id)
	return
}

func (db *DB) SetBusyStatus(id gp.UserId, busy bool) (err error) {
	_, err = db.stmt["setBusy"].Exec(busy, id)
	return
}

func (db *DB) BusyStatus(id gp.UserId) (busy bool, err error) {
	err = db.stmt["getBusy"].QueryRow(id).Scan(&busy)
	return
}

func (db *DB) UserIdFromFB(fbid uint64) (id gp.UserId, err error) {
	err = db.stmt["idFromFacebook"].QueryRow(fbid).Scan(&id)
	return
}

func (db *DB) SetVerificationToken(id gp.UserId, token string) (err error) {
	_, err = db.stmt["insertVerification"].Exec(id, token)
	return
}

func (db *DB) VerificationTokenExists(token string) (id gp.UserId, err error) {
	err = db.stmt["verificationExists"].QueryRow(token).Scan(&id)
	return
}

func (db *DB) Verify(id gp.UserId) (err error) {
	_, err = db.stmt["verify"].Exec(id)
	return
}

func (db *DB) IsVerified(user gp.UserId) (verified bool, err error) {
	err = db.stmt["userIsVerified"].QueryRow(user).Scan(&verified)
	return
}

func (db *DB) GetEmail(id gp.UserId) (email string, err error) {
	err = db.stmt["emailSelect"].QueryRow(id).Scan(&email)
	return
}

func (db *DB) UserWithEmail(email string) (id gp.UserId, err error) {
	err = db.stmt["userWithEmail"].QueryRow(email).Scan(&id)
	return
}

func (db *DB) CreateFBUser(fbId uint64, email string) (err error) {
	_, err = db.stmt["fbInsert"].Exec(fbId, email)
	return
}

func (db *DB) FBUserEmail(fbid uint64) (email string, err error) {
	err = db.stmt["selectFBemail"].QueryRow(fbid).Scan(&email)
	return
}

func (db *DB) FBUserWithEmail(email string) (fbid uint64, err error) {
	err = db.stmt["fbUserByEmail"].QueryRow(email).Scan(&fbid)
	return
}
func (db *DB) CreateFBVerification(fbid uint64, token string) (err error) {
	_, err = db.stmt["fbInsertVerification"].Exec(fbid, token)
	return
}

func (db *DB) FBVerificationExists(token string) (fbid uint64, err error) {
	err = db.stmt["fbVerificationExists"].QueryRow(token).Scan(&fbid)
	return
}

func (db *DB) FBSetGPUser(fbid uint64, userId gp.UserId) (err error) {
	_, err = db.stmt["fbSetGPUser"].Exec(fbid, userId)
	return
}

func (db *DB) AddPasswordRecovery(userId gp.UserId, token string) (err error) {
	_, err = db.stmt["addPasswordRecovery"].Exec(token, userId)
	return
}

func (db *DB) CheckPasswordRecovery(userId gp.UserId, token string) (exists bool, err error) {
	err = db.stmt["checkPasswordRecovery"].QueryRow(userId, token).Scan(&exists)
	return
}

func (db *DB) DeletePasswordRecovery(userId gp.UserId, token string) (err error) {
	_, err = db.stmt["deletePasswordRecovery"].Exec(userId, token)
	return
}

/********************************************************************
		Conversation
********************************************************************/

//GetLiveConversations returns the three most recent unfinished live conversations for a given user.
//TODO: retrieve conversation & expiry in a single query
func (db *DB) GetLiveConversations(id gp.UserId) (conversations []gp.ConversationSmall, err error) {
	s := db.stmt["liveConversations"]
	rows, err := s.Query(id)
	if err != nil {
		return conversations, err
	}
	defer rows.Close()
	for rows.Next() {
		var conv gp.ConversationSmall
		var t string
		err = rows.Scan(&conv.Id, &t)
		if err != nil {
			return conversations, err
		}
		conv.LastActivity, _ = time.Parse(mysqlTime, t)
		conv.Participants = db.GetParticipants(conv.Id)
		LastMessage, err := db.GetLastMessage(conv.Id)
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		Expiry, err := db.ConversationExpiry(conv.Id)
		if err == nil {
			conv.Expiry = &Expiry
		}
		conversations = append(conversations, conv)
	}
	return conversations, nil
}

func (db *DB) CreateConversation(id gp.UserId, participants []gp.User, expiry *gp.Expiry) (conversation gp.Conversation, err error) {
	s := db.stmt["conversationInsert"]
	r, _ := s.Exec(id)
	cId, _ := r.LastInsertId()
	conversation.Id = gp.ConversationId(cId)
	if err != nil {
		return
	}
	log.Println("DB hit: createConversation (user.Name, user.Id)")
	sta := db.stmt["participantInsert"]
	for _, u := range participants {
		_, err = sta.Exec(conversation.Id, u.Id)
		if err != nil {
			return
		}
	}
	conversation.Participants = participants
	conversation.LastActivity = time.Now().UTC()
	if expiry != nil {
		conversation.Expiry = expiry
		err = db.ConversationSetExpiry(conversation.Id, *conversation.Expiry)
	}
	return
}

func (db *DB) RandomPartners(id gp.UserId, count int, network gp.NetworkId) (partners []gp.User, err error) {
q :="SELECT id, name, firstname, avatar " +
                "FROM users " +
                "LEFT JOIN user_network ON id = user_id " +
                "WHERE network_id = ? " +
                "ORDER BY RAND()"
	log.Println(q, id, count, network)

	s := db.stmt["randomSelect"]
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
		err = rows.Scan(&user.Id, &user.Name, &first, &av)
		if err != nil {
			log.Println("Error scanning from user query", err)
			return
		} else {
			log.Println("Got a partner")
			liveCount, err := db.LiveCount(user.Id)
			if err == nil && liveCount < 3 && user.Id != id {
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
	}
	return
}

func (db *DB) LiveCount(userId gp.UserId) (count int, err error) {
	q := "SELECT COUNT( conversation_participants.conversation_id ) FROM conversation_participants JOIN conversations ON conversation_participants.conversation_id = conversations.id JOIN conversation_expirations ON conversation_expirations.conversation_id = conversations.id WHERE participant_id = ? AND conversation_expirations.ended = 0 AND conversation_expirations.expiry > NOW( )"
	stmt, err := db.prepare(q)
	if err != nil {
		return
	}
	err = stmt.QueryRow(userId).Scan(&count)
	return
}

func (db *DB) UpdateConversation(id gp.ConversationId) (err error) {
	s := db.stmt["conversationUpdate"]
	_, err = s.Exec(id)
	log.Println("DB hit: updateConversation convid ")
	if err != nil {
		log.Printf("Error: %v", err)
	}
	return err
}

func (db *DB) GetConversations(userId gp.UserId, start int64, count int, all bool) (conversations []gp.ConversationSmall, err error) {
	var s *sql.Stmt
	if all {
		s = db.stmt["conversationsAll"]
	} else {
		s = db.stmt["conversationSelect"]
	}
	rows, err := s.Query(userId, start, count)
	log.Println("DB hit: getConversations user_id, start (conversation.id)")
	if err != nil {
		return conversations, err
	}
	defer rows.Close()
	for rows.Next() {
		var conv gp.ConversationSmall
		var t string
		err = rows.Scan(&conv.Id, &t)
		if err != nil {
			return conversations, err
		}
		conv.LastActivity, _ = time.Parse(mysqlTime, t)
		conv.Participants = db.GetParticipants(conv.Id)
		LastMessage, err := db.GetLastMessage(conv.Id)
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		Expiry, err := db.ConversationExpiry(conv.Id)
		if err == nil {
			conv.Expiry = &Expiry
		}
		read, err := db.GetReadStatus(conv.Id)
		if err == nil {
			conv.Read = read
		}
		conversations = append(conversations, conv)
	}
	return conversations, nil
}

func (db *DB) ConversationActivity(convId gp.ConversationId) (t time.Time, err error) {
	s := db.stmt["conversationActivity"]
	var tstring string
	err = s.QueryRow(convId).Scan(&tstring)
	if err != nil {
		return
	}
	t, err = time.Parse(mysqlTime, tstring)
	return
}

func (db *DB) ConversationExpiry(convId gp.ConversationId) (expiry gp.Expiry, err error) {
	s := db.stmt["conversationExpiry"]
	var t string
	err = s.QueryRow(convId).Scan(&t, &expiry.Ended)
	if err != nil {
		return
	}
	expiry.Time, err = time.Parse(mysqlTime, t)
	return
}

func (db *DB) DeleteConversationExpiry(convId gp.ConversationId) (err error) {
	_, err = db.stmt["deleteExpiry"].Exec(convId)
	return
}

func (db *DB) TerminateConversation(convId gp.ConversationId) (err error) {
	_, err = db.stmt["endConversation"].Exec(convId)
	return
}

func (db *DB) ConversationSetExpiry(convId gp.ConversationId, expiry gp.Expiry) (err error) {
	s := db.stmt["conversationSetExpiry"]
	_, err = s.Exec(convId, expiry.Time)
	return
}

//GetConversation returns the conversation convId, including up to count messages.
func (db *DB) GetConversation(convId gp.ConversationId, count int) (conversation gp.ConversationAndMessages, err error) {
	conversation.Id = convId
	conversation.LastActivity, err = db.ConversationActivity(convId)
	if err != nil {
		return
	}
	conversation.Participants = db.GetParticipants(convId)
	read, err := db.GetReadStatus(convId)
	if err == nil {
		conversation.Read = read
	}
	expiry, err := db.ConversationExpiry(convId)
	if err == nil {
		conversation.Expiry = &expiry
	}
	conversation.Messages, err = db.GetMessages(convId, 0, "start", count)
	return
}

func (db *DB) ConversationsToTerminate(id gp.UserId) (conversations []gp.ConversationId, err error) {
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
		var id gp.ConversationId
		err = rows.Scan(&id)
		if err != nil {
			return
		}
		conversations = append(conversations, id)
	}
	return
}

//GetReadStatus returns all the positions the participants in this conversation have read to. It omits participants who haven't read.
func (db *DB) GetReadStatus(convId gp.ConversationId) (read []gp.Read, err error) {
	rows, err := db.stmt["readStatus"].Query(convId)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var r gp.Read
		err = rows.Scan(&r.UserId, &r.LastRead)
		if err != nil {
			return
		}
		if r.LastRead > 0 {
			read = append(read, r)
		}
	}
	return
}

//GetParticipants returns all of the participants in conv.
//TODO: Return an error when appropriate
func (db *DB) GetParticipants(conv gp.ConversationId) []gp.User {
	s := db.stmt["participantSelect"]
	rows, err := s.Query(conv)
	log.Println("DB hit: getParticipants convid (user.id)")
	if err != nil {
		log.Printf("Error getting participant: %v", err)
	}
	defer rows.Close()
	participants := make([]gp.User, 0, 5)
	for rows.Next() {
		var id gp.UserId
		err = rows.Scan(&id)
		user, err := db.GetUser(id)
		if err == nil {
			participants = append(participants, user)
		}
	}
	return (participants)
}

//GetLastMessage retrieves the most recent message in conversation id.
func (db *DB) GetLastMessage(id gp.ConversationId) (message gp.Message, err error) {
	var timeString string
	var by gp.UserId
	s := db.stmt["lastMessageSelect"]
	err = s.QueryRow(id).Scan(&message.Id, &by, &message.Text, &timeString)
	log.Println("DB hit: db.GetLastMessage convid (message.id, message.by, message.text, message.time)")
	if err != nil {
		return message, err
	} else {
		message.By, err = db.GetUser(by)
		if err != nil {
			log.Printf("error getting user %d %v", by, err)
		}
		message.Time, _ = time.Parse(mysqlTime, timeString)

		return message, nil
	}
}

/********************************************************************
		Post
********************************************************************/

//GetUserPosts returns the most recent count posts by userId after the post with id after.
func (db *DB) GetUserPosts(userId gp.UserId, index int64, count int, sel string) (posts []gp.PostSmall, err error) {
	var q string
	switch {
	case sel == "start":
		q = "SELECT wall_posts.id, `by`, time, text " +
			"FROM wall_posts " +
			"WHERE `by` = ? " +
			"ORDER BY time DESC LIMIT ?, ?"
	case sel == "before":
		q = "SELECT wall_posts.id, `by`, time, text " +
			"FROM wall_posts " +
			"WHERE `by` = ? AND id < ? " +
			"ORDER BY time DESC LIMIT 0, ?"
	case sel == "after":
		q = "SELECT wall_posts.id, `by`, time, text " +
			"FROM wall_posts " +
			"WHERE `by` = ? AND id > ? " +
			"ORDER BY time DESC LIMIT 0, ?"
	default:
		return posts, gp.APIerror{"Invalid selector"}
	}

	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(userId, index, count)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var post gp.PostSmall
		var t string
		var by gp.UserId
		err = rows.Scan(&post.Id, &by, &t, &post.Text)
		if err != nil {
			return posts, err
		}
		post.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return posts, err
		}
		post.By, err = db.GetUser(by)
		if err == nil {
			post.CommentCount = db.GetCommentCount(post.Id)
			post.Images, err = db.GetPostImages(post.Id)
			if err != nil {
				return
			}
			post.LikeCount, err = db.LikeCount(post.Id)
			if err != nil {
				return
			}
			posts = append(posts, post)
		} else {
			log.Println("Bad post: ", post)
		}
	}
	return
}

func (db *DB) AddPost(userId gp.UserId, text string, network gp.NetworkId) (postId gp.PostId, err error) {
	s := db.stmt["postInsert"]
	res, err := s.Exec(userId, text, network)
	if err != nil {
		return 0, err
	}
	_postId, err := res.LastInsertId()
	postId = gp.PostId(_postId)
	if err != nil {
		return 0, err
	}
	return postId, nil
}

//GetLive returns a list of events whose event time is after "after", ordered by time.
func (db *DB) GetLive(netId gp.NetworkId, after time.Time, count int) (posts []gp.PostSmall, err error) {
	s := db.stmt["liveSelect"]
	rows, err := s.Query(netId, after.Unix(), count)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var post gp.PostSmall
		var t string
		var by gp.UserId
		err = rows.Scan(&post.Id, &by, &t, &post.Text)
		if err != nil {
			return posts, err
		}
		post.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return posts, err
		}
		post.By, err = db.GetUser(by)
		if err == nil {
			post.CommentCount = db.GetCommentCount(post.Id)
			post.Images, err = db.GetPostImages(post.Id)
			if err != nil {
				return
			}
			post.LikeCount, err = db.LikeCount(post.Id)
			if err != nil {
				return
			}
			posts = append(posts, post)
		} else {
			log.Println("Bad post: ", post)
		}
	}
	return
}

//GetPosts finds posts in the network netId.
func (db *DB) GetPosts(netId gp.NetworkId, index int64, count int, sel string) (posts []gp.PostSmall, err error) {
	var s *sql.Stmt
	switch {
	case sel == "start":
		s = db.stmt["wallSelect"]
	case sel == "before":
		s = db.stmt["wallSelectBefore"]
	case sel == "after":
		s = db.stmt["wallSelectAfter"]
	default:
		return posts, gp.APIerror{"Invalid selector"}
	}
	rows, err := s.Query(netId, index, count)
	log.Println(rows, err, netId, index, count)
	log.Println("DB hit: getPosts netId(post.id, post.by, post.time, post.texts)")
	if err != nil {
		log.Println("Error yo! ", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		log.Println("Post!")
		var post gp.PostSmall
		var t string
		var by gp.UserId
		err = rows.Scan(&post.Id, &by, &t, &post.Text)
		if err != nil {
			return posts, err
		}
		post.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return posts, err
		}
		post.By, err = db.GetUser(by)
		if err == nil {
			post.CommentCount = db.GetCommentCount(post.Id)
			post.Images, err = db.GetPostImages(post.Id)
			if err != nil {
				return
			}
			post.LikeCount, err = db.LikeCount(post.Id)
			if err != nil {
				return
			}
			posts = append(posts, post)
		} else {
			log.Println("Bad post: ", post)
		}
	}
	return
}

//GetPosts finds posts in the network netId.
func (db *DB) GetPostsByCategory(netId gp.NetworkId, index int64, count int, sel string, categoryTag string) (posts []gp.PostSmall, err error) {
	var s *sql.Stmt
	switch {
	case sel == "start":
		s = db.stmt["wallSelectCategory"]
	case sel == "before":
		s = db.stmt["wallSelectCategoryBefore"]
	case sel == "after":
		s = db.stmt["wallSelectCategoryAfter"]
	default:
		return posts, gp.APIerror{"Invalid selector"}
	}
	rows, err := s.Query(netId, categoryTag, index, count)
	defer rows.Close()
	log.Printf("DB hit: getPostsByCategory network: %s category: %s index: %d count: %d", netId, categoryTag, index, count)
	if err != nil {
		log.Println(err)
		return
	}
	for rows.Next() {
		log.Println("Got a post")
		var post gp.PostSmall
		var t string
		var by gp.UserId
		err = rows.Scan(&post.Id, &by, &t, &post.Text)
		log.Println("Scanned a post")
		if err != nil {
			log.Println("Error scanning post: ", err)
			return posts, err
		}
		post.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			log.Println("Error parsing time: ", err)
			return posts, err
		}
		post.By, err = db.GetUser(by)
		if err == nil {
			post.CommentCount = db.GetCommentCount(post.Id)
			post.Images, err = db.GetPostImages(post.Id)
			if err != nil {
				return
			}
			post.LikeCount, err = db.LikeCount(post.Id)
			if err != nil {
				return
			}
			posts = append(posts, post)
			log.Println("Added a post")
		} else {
			log.Println("Bad post: ", post)
		}
	}
	return
}

func (db *DB) GetPostImages(postId gp.PostId) (images []string, err error) {
	s := db.stmt["imageSelect"]
	rows, err := s.Query(postId)
	defer rows.Close()
	log.Println("DB hit: getImages postId(image)")
	if err != nil {
		return
	}
	for rows.Next() {
		var image string
		err = rows.Scan(&image)
		if err != nil {
			return
		}
		images = append(images, image)
	}
	return
}

func (db *DB) AddPostImage(postId gp.PostId, url string) (err error) {
	_, err = db.stmt["imageInsert"].Exec(postId, url)
	return
}

func (db *DB) CreateComment(postId gp.PostId, userId gp.UserId, text string) (commId gp.CommentId, err error) {
	s := db.stmt["commentInsert"]
	if res, err := s.Exec(postId, userId, text); err == nil {
		cId, err := res.LastInsertId()
		commId = gp.CommentId(cId)
		return commId, err
	} else {
		return 0, err
	}
}

func (db *DB) GetComments(postId gp.PostId, start int64, count int) (comments []gp.Comment, err error) {
	s := db.stmt["commentSelect"]
	rows, err := s.Query(postId, start, count)
	log.Println("DB hit: getComments postid, start(comment.id, comment.by, comment.text, comment.time)")
	if err != nil {
		return comments, err
	}
	defer rows.Close()
	for rows.Next() {
		var comment gp.Comment
		comment.Post = postId
		var timeString string
		var by gp.UserId
		err := rows.Scan(&comment.Id, &by, &comment.Text, &timeString)
		if err != nil {
			return comments, err
		}
		comment.Time, _ = time.Parse(mysqlTime, timeString)
		comment.By, err = db.GetUser(by)
		if err != nil {
			log.Printf("error getting user %d %v", by, err)
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func (db *DB) GetCommentCount(id gp.PostId) (count int) {
	s := db.stmt["commentCountSelect"]
	err := s.QueryRow(id).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

//GetPost returns the post postId or an error if it doesn't exist.
//TODO: This could return without an embedded user or images array
func (db *DB) GetPost(postId gp.PostId) (post gp.Post, err error) {
	s := db.stmt["postSelect"]
	post.Id = postId
	var by gp.UserId
	var t string
	err = s.QueryRow(postId).Scan(&by, &t, &post.Text)
	if err != nil {
		return
	}
	post.By, err = db.GetUser(by)
	if err != nil {
		return
	}
	post.Time, err = time.Parse(mysqlTime, t)
	if err != nil {
		return
	}
	post.Images, err = db.GetPostImages(postId)
	return
}

//SetPostAttribs associates all the attribute:value pairs in attrib with post.
//At the moment, it doesn't check if these attributes are at all reasonable;
//the onus is on the viewer of the attributes to look for just the ones which make sense,
//and on the caller of this function to ensure that the values conform to a particular format.
func (db *DB) SetPostAttribs(post gp.PostId, attribs map[string]string) (err error) {
	s := db.stmt["setPostAttribs"]
	for attrib, value := range attribs {
		//How could I be so foolish to store time strings rather than unix timestamps...
		if attrib == "event-time" {
			t, e := time.Parse(value, time.RFC3339)
			if e != nil {
				unixt, e := strconv.ParseInt(value, 10, 64)
				if e != nil {
					return e
				}
				t = time.Unix(unixt, 0)
			}
			unix := t.Unix()
			value = strconv.FormatInt(unix, 10)
		}
		_, err = s.Exec(post, attrib, value)
		if err != nil {
			return
		}
	}
	return
}

//GetPostAttribs returns a map of all attributes associated with post.
func (db *DB) GetPostAttribs(post gp.PostId) (attribs map[string]interface{}, err error) {
	s := db.stmt["getPostAttribs"]
	rows, err := s.Query(post)
	if err != nil {
		return
	}
	defer rows.Close()
	attribs = make(map[string]interface{})
	for rows.Next() {
		var attrib, val string
		err = rows.Scan(&attrib, &val)
		if err != nil {
			return
		}
		switch {
		case attrib == "event-time":
			log.Println("event-time")
			var unix int64
			unix, err = strconv.ParseInt(val, 10, 64)
			if err == nil {
				log.Println("no error")
				attribs[attrib] = time.Unix(unix, 0)
			}
		default:
			attribs[attrib] = val
		}
	}
	return
}

func (db *DB) GetEventPopularity(post gp.PostId) (popularity int, err error) {
	query := "SELECT COUNT(*) FROM event_attendees WHERE post_id = ?"
	s, err := db.prepare(query)
	if err != nil {
		return
	}
	err = s.QueryRow(post).Scan(&popularity)
	if err != nil {
		return
	}
	switch {
	case popularity > 20:
		popularity = 4
	case popularity > 10:
		popularity = 3
	case popularity > 5:
		popularity = 2
	case popularity > 0:
		popularity = 1
	default:
		popularity = 0
	}
	return
}

/********************************************************************
		Message
********************************************************************/

func (db *DB) AddMessage(convId gp.ConversationId, userId gp.UserId, text string) (id gp.MessageId, err error) {
	log.Printf("Adding message to db: %d, %d %s", convId, userId, text)
	s := db.stmt["messageInsert"]
	res, err := s.Exec(convId, userId, text)
	if err != nil {
		return 0, err
	}
	_id, err := res.LastInsertId()
	id = gp.MessageId(_id)
	return
}

//GetMessages retrieves n = count messages from the conversation convId.
//These can be starting from the offset index (when sel == "start"); or they can
//be the n messages before or after index when sel == "before" or "after" respectively.
//I don't know what will happen if you give sel something else, probably a null pointer
//exception.
//TODO: This could return a message which doesn't embed a user
//BUG(Patrick): Should return an error when sel isn't right!
func (db *DB) GetMessages(convId gp.ConversationId, index int64, sel string, count int) (messages []gp.Message, err error) {
	var s *sql.Stmt
	switch {
	case sel == "after":
		s = db.stmt["messageSelectAfter"]
	case sel == "before":
		s = db.stmt["messageSelectBefore"]
	case sel == "start":
		s = db.stmt["messageSelect"]
	}
	rows, err := s.Query(convId, index, count)
	log.Println("DB hit: getMessages convid, start (message.id, message.by, message.text, message.time)")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var message gp.Message
		var timeString string
		var by gp.UserId
		err = rows.Scan(&message.Id, &by, &message.Text, &timeString)
		if err != nil {
			log.Printf("%v", err)
		}
		message.Time, err = time.Parse(mysqlTime, timeString)
		if err != nil {
			log.Printf("%v", err)
		}
		message.By, err = db.GetUser(by)
		if err != nil {
			return
		}
		messages = append(messages, message)
	}
	return
}

//MarkRead will set all messages in the conversation convId read = true
//up to and including upTo and excluding messages sent by user id.
//TODO: This won't generalize to >2 participants
func (db *DB) MarkRead(id gp.UserId, convId gp.ConversationId, upTo gp.MessageId) (err error) {
	_, err = db.stmt["messagesRead"].Exec(upTo, convId, id)
	return
}

//AddCategory marks the post id as a member of category.
func (db *DB) AddCategory(id gp.PostId, category gp.CategoryId) (err error) {
	_, err = db.stmt["categoryAdd"].Exec(id, category)
	return
}

//CategoryList returns all existing categories.
func (db *DB) CategoryList() (categories []gp.PostCategory, err error) {
	rows, err := db.stmt["listCategories"].Query()
	defer rows.Close()
	for rows.Next() {
		c := gp.PostCategory{}
		err = rows.Scan(&c.Id, &c.Tag, &c.Name)
		if err != nil {
			return
		}
		categories = append(categories, c)
	}
	return
}

//SetCategories accepts a post id and any number of string tags. Any of the tags that exist will be added to the post.
func (db *DB) TagPost(post gp.PostId, tags ...string) (err error) {
	for _, tag := range tags {
		_, err = db.stmt["addCategoryWhereExists"].Exec(post, tag)
		if err != nil {
			return
		}
	}
	return
}

//PostCategories returns all the categories which post belongs to.
func (db *DB) PostCategories(post gp.PostId) (categories []gp.PostCategory, err error) {
	rows, err := db.stmt["postCategories"].Query(post)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		c := gp.PostCategory{}
		err = rows.Scan(&c.Id, &c.Tag, &c.Name)
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

func (db *DB) TokenExists(id gp.UserId, token string) bool {
	var expiry string
	s := db.stmt["tokenSelect"]
	err := s.QueryRow(id, token).Scan(&expiry)
	if err != nil {
		return (false)
	} else {
		t, _ := time.Parse(mysqlTime, expiry)
		if t.After(time.Now()) {
			return (true)
		}
		return (false)
	}
}

func (db *DB) AddToken(token gp.Token) (err error) {
	s := db.stmt["tokenInsert"]
	_, err = s.Exec(token.UserId, token.Token, token.Expiry)
	return
}

/********************************************************************
		Contact
********************************************************************/

func (db *DB) AddContact(adder gp.UserId, addee gp.UserId) (err error) {
	log.Println("DB hit: addContact")
	s := db.stmt["contactInsert"]
	_, err = s.Exec(adder, addee)
	return
}

//GetContacts retrieves all the contacts for user.
//TODO: This could return contacts which doesn't embed a user
func (db *DB) GetContacts(user gp.UserId) (contacts []gp.Contact, err error) {
	s := db.stmt["contactSelect"]
	rows, err := s.Query(user, user)
	log.Println("DB hit: db.GetContacts")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var contact gp.Contact
		var adder, addee gp.UserId
		var confirmed bool
		err = rows.Scan(&adder, &addee, &confirmed)
		if err != nil {
			return
		}
		switch {
		case adder == user:
			contact.User, err = db.GetUser(addee)
			if err != nil {
				return
			}
			contact.YouConfirmed = true
			contact.TheyConfirmed = confirmed
		case addee == user:
			contact.User, err = db.GetUser(adder)
			if err != nil {
				return
			}
			contact.YouConfirmed = confirmed
			contact.TheyConfirmed = true
		}
		contacts = append(contacts, contact)
	}
	return
}

func (db *DB) UpdateContact(user gp.UserId, contact gp.UserId) (err error) {
	s := db.stmt["contactUpdate"]
	_, err = s.Exec(user, contact)
	return
}

func (db *DB) ContactRequestExists(adder gp.UserId, addee gp.UserId) (exists bool, err error) {
	err = db.stmt["contactExists"].QueryRow(adder, addee).Scan(&exists)
	return
}

/********************************************************************
		Device
********************************************************************/

func (db *DB) AddDevice(user gp.UserId, deviceType string, deviceId string) (err error) {
	s := db.stmt["deviceInsert"]
	_, err = s.Exec(user, deviceType, deviceId)
	return
}

func (db *DB) GetDevices(user gp.UserId) (devices []gp.Device, err error) {
	s := db.stmt["deviceSelect"]
	rows, err := s.Query(user)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		device := gp.Device{}
		if err = rows.Scan(&device.User, &device.Type, &device.Id); err != nil {
			return
		}
		devices = append(devices, device)
	}
	return
}

func (db *DB) DeleteDevice(user gp.UserId, device string) (err error) {
	log.Printf("Deleting %d's device: %s\n", user, device)
	s := db.stmt["deviceDelete"]
	_, err = s.Exec(user, device)
	return
}

func (db *DB) Feedback(deviceId string, timestamp time.Time) (err error) {
	s := db.stmt["feedbackDelete"]
	r, err := s.Exec(deviceId, timestamp)
	n, _ := r.RowsAffected()
	log.Printf("Feedback: %d devices deleted\n", n)
	return
}

/********************************************************************
		Upload
********************************************************************/

func (db *DB) AddUpload(user gp.UserId, url string) (err error) {
	_, err = db.stmt["userUpload"].Exec(user, url)
	return
}

func (db *DB) UploadExists(user gp.UserId, url string) (exists bool, err error) {
	err = db.stmt["uploadExists"].QueryRow(user, url).Scan(&exists)
	return
}

/********************************************************************
		Notification
********************************************************************/

func (db *DB) GetUserNotifications(id gp.UserId) (notifications []interface{}, err error) {
	s := db.stmt["notificationSelect"]
	rows, err := s.Query(id)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var notification gp.Notification
		var t string
		var post sql.NullInt64
		var by gp.UserId
		if err = rows.Scan(&notification.Id, &notification.Type, &t, &by, &post, &notification.Seen); err != nil {
			return
		}
		notification.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return
		}
		notification.By, err = db.GetUser(by)
		if err != nil {
			return
		}
		if post.Valid {
			var np gp.PostNotification
			np.Notification = notification
			np.Post = gp.PostId(post.Int64)
			notifications = append(notifications, np)
		} else {
			notifications = append(notifications, notification)
		}
	}
	return
}

func (db *DB) MarkNotificationsSeen(user gp.UserId, upTo gp.NotificationId) (err error) {
	_, err = db.stmt["notificationUpdate"].Exec(user, upTo)
	return
}

func (db *DB) CreateNotification(ntype string, by gp.UserId, recipient gp.UserId, isPN bool, post gp.PostId) (notification interface{}, err error) {
	var res sql.Result
	if isPN {
		s := db.stmt["postNotificationInsert"]
		res, err = s.Exec(ntype, by, recipient, post)
	} else {
		s := db.stmt["notificationInsert"]
		res, err = s.Exec(ntype, by, recipient)
	}
	if err != nil {
		return
	} else {
		n := gp.Notification{
			Type: ntype,
			Time: time.Now().UTC(),
			Seen: false,
		}
		id, iderr := res.LastInsertId()
		if iderr != nil {
			return n, iderr
		}
		n.Id = gp.NotificationId(id)
		n.By, err = db.GetUser(by)
		if err != nil {
			return
		}
		if isPN {
			np := gp.PostNotification{n, post}
			return np, nil
		} else {
			return n, nil
		}
	}
}

func (db *DB) CreateLike(user gp.UserId, post gp.PostId) (err error) {
	_, err = db.stmt["addLike"].Exec(post, user)
	// Suppress duplicate entry errors
	if err != nil {
		if strings.HasPrefix(err.Error(), "Error 1062") {
			err = nil
		}
	}
	return
}

func (db *DB) RemoveLike(user gp.UserId, post gp.PostId) (err error) {
	_, err = db.stmt["delLike"].Exec(post, user)
	return
}

func (db *DB) GetLikes(post gp.PostId) (likes []gp.Like, err error) {
	rows, err := db.stmt["likeSelect"].Query(post)
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

func (db *DB) HasLiked(user gp.UserId, post gp.PostId) (liked bool, err error) {
	err = db.stmt["likeExists"].QueryRow(post, user).Scan(&liked)
	return
}

func (db *DB) LikeCount(post gp.PostId) (count int, err error) {
	err = db.stmt["likeCount"].QueryRow(post).Scan(&count)
	return
}

//Attend adds the user to the "attending" list for this event. It's idempotent, and should only return an error if the database is down.
//The results are undefined for a post which isn't an event.
//(ie: it will work even though it shouldn't, until I can get round to enforcing it.)
func (db *DB) Attend(event gp.PostId, user gp.UserId) (err error) {
	query := "REPLACE INTO event_attendees (post_id, user_id) VALUES (?, ?)"
	s, err := db.prepare(query)
	if err != nil {
		return
	}
	_, err = s.Exec(event, user)
	return
}

//UnAttend removes a user's attendance to an event. Idempotent, returns an error if the DB is down.
func (db *DB) UnAttend(event gp.PostId, user gp.UserId) (err error) {
	query := "DELETE FROM event_attendees WHERE post_id = ? AND user_id = ?"
	s, err := db.prepare(query)
	if err != nil {
		return
	}
	_, err = s.Exec(event, user)
	return
}

//UserAttends returns all the event IDs that a user is attending.
func (db *DB) UserAttends(user gp.UserId) (events []gp.PostId, err error) {
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
		var post gp.PostId
		err = rows.Scan(&post)
		if err != nil {
			return
		}
		events = append(events, post)
	}
	return
}

func (db *DB) UnreadMessageCount(user gp.UserId) (count int, err error) {
	qParticipate := "SELECT conversation_id, last_read FROM conversation_participants WHERE participant_id = ?"
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
	var convId gp.ConversationId
	var lastId gp.MessageId
	for rows.Next() {
		err = rows.Scan(&convId, &lastId)
		if err != nil {
			return
		}
		log.Printf("Conversation %d, last read message was %d\n", convId, lastId)
		_count := 0
		err = sUnreadCount.QueryRow(convId, lastId).Scan(&_count)
		if err == nil {
			log.Printf("Conversation %d, unread message count was %d\n", convId, _count)
			count += _count
		}
	}
	return count, nil
}

func (db *DB) TotalLiveConversations(user gp.UserId) (count int, err error) {
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
		err = rows.Scan(&conv.Id, &t)
		if err != nil {
			return 0, err
		}
		conv.LastActivity, _ = time.Parse(mysqlTime, t)
		conv.Participants = db.GetParticipants(conv.Id)
		LastMessage, err := db.GetLastMessage(conv.Id)
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		Expiry, err := db.ConversationExpiry(conv.Id)
		if err == nil {
			conv.Expiry = &Expiry
		}
		conversations = append(conversations, conv)
	}
	return len(conversations), nil
}

func (db *DB) PrunableConversations() (conversations []gp.ConversationId, err error) {
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
		var c gp.ConversationId
		err = rows.Scan(&c)
		if err != nil {
			return
		}
		conversations = append(conversations, c)
	}
	return conversations, nil
}
