package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strings"
	"time"
)

const (
	//For parsing
	MysqlTime = "2006-01-02 15:04:05"
)

var (
	sqlStmt map[string]string
	stmt    map[string]*sql.Stmt
)

func keepalive(db *sql.DB) {
	tick := time.Tick(1 * time.Hour)
	conf := GetConfig()
	for {
		<-tick
		err := db.Ping()
		if err != nil {
			log.Print(err)
			db, err = sql.Open("mysql", conf.ConnectionString())
			if err != nil {
				log.Fatalf("Error opening database: %v", err)
			}
		}
	}
}

func prepare(db *sql.DB) (err error) {
	sqlStmt = make(map[string]string)
	stmt = make(map[string]*sql.Stmt)
	//Network
	sqlStmt["ruleSelect"] = "SELECT network_id, rule_type, rule_value FROM net_rules"
	sqlStmt["networkSelect"] = "SELECT user_network.network_id, network.name FROM user_network INNER JOIN network ON user_network.network_id = network.id WHERE user_id = ?"
	sqlStmt["networkInsert"] = "INSERT INTO user_network (user_id, network_id) VALUES (?, ?)"
	//User
	sqlStmt["createUser"] = "INSERT INTO users(name, password, email) VALUES (?,?,?)"
	sqlStmt["userSelect"] = "SELECT id, name, avatar FROM users WHERE id=?"
	sqlStmt["profileSelect"] = "SELECT name, `desc`, avatar FROM users WHERE id = ?"
	sqlStmt["passSelect"] = "SELECT id, password FROM users WHERE name = ?"
	sqlStmt["randomSelect"] = "SELECT id, name FROM users ORDER BY RAND()"
	sqlStmt["setAvatar"] = "UPDATE users SET avatar = ? WHERE id = ?"
	//Conversation
	sqlStmt["conversationInsert"] = "INSERT INTO conversations (initiator, last_mod) VALUES (?, NOW())"
	sqlStmt["conversationUpdate"] = "UPDATE conversations SET last_mod = NOW() WHERE id = ?"
	sqlStmt["conversationSelect"] = "SELECT conversation_participants.conversation_id, conversations.last_mod FROM conversation_participants JOIN conversations ON conversation_participants.conversation_id = conversations.id WHERE participant_id = ? ORDER BY conversations.last_mod DESC LIMIT ?, 20"
	sqlStmt["participantInsert"] = "INSERT INTO conversation_participants (conversation_id, participant_id) VALUES (?,?)"
	sqlStmt["participantSelect"] = "SELECT participant_id FROM conversation_participants JOIN users ON conversation_participants.participant_id = users.id WHERE conversation_id=?"
	sqlStmt["lastMessageSelect"] = "SELECT id, `from`, text, timestamp, seen FROM chat_messages WHERE conversation_id = ? ORDER BY timestamp DESC LIMIT 1"
	//Post
	sqlStmt["postInsert"] = "INSERT INTO wall_posts(`by`, `text`, network_id) VALUES (?,?,?)"
	sqlStmt["wallSelect"] = "SELECT id, `by`, time, text FROM wall_posts WHERE network_id = ? ORDER BY time DESC LIMIT ?, ?"
	sqlStmt["imageSelect"] = "SELECT url FROM post_images WHERE post_id = ?"
	sqlStmt["imageInsert"] = "INSERT INTO post_images (post_id, url) VALUES (?, ?)"
	sqlStmt["commentInsert"] = "INSERT INTO post_comments (post_id, `by`, text) VALUES (?, ?, ?)"
	sqlStmt["commentSelect"] = "SELECT id, `by`, text, timestamp FROM post_comments WHERE post_id = ? ORDER BY timestamp DESC LIMIT ?, ?"
	sqlStmt["commentCountSelect"] = "SELECT COUNT(*) FROM post_comments WHERE post_id = ?"
	//Message
	sqlStmt["messageInsert"] = "INSERT INTO chat_messages (conversation_id, `from`, `text`) VALUES (?,?,?)"
	sqlStmt["messageSelect"] = "SELECT id, `from`, text, timestamp, seen FROM chat_messages WHERE conversation_id = ? ORDER BY timestamp DESC LIMIT ?, ?"
	sqlStmt["messageSelectAfter"] = "SELECT id, `from`, text, timestamp, seen FROM chat_messages WHERE conversation_id = ? AND id > ? ORDER BY timestamp DESC LIMIT ?"
	sqlStmt["messageSelectBefore"] = "SELECT id, `from`, text, timestamp, seen FROM chat_messages WHERE conversation_id = ? AND id < ? ORDER BY timestamp DESC LIMIT ?"
	sqlStmt["messagesRead"] = "UPDATE chat_messages SET seen = 1 WHERE conversation_id=? AND id <= ? AND `from` != ?"
	//Token
	sqlStmt["tokenInsert"] = "INSERT INTO tokens (user_id, token, expiry) VALUES (?, ?, ?)"
	sqlStmt["tokenSelect"] = "SELECT expiry FROM tokens WHERE user_id = ? AND token = ?"
	//Contact
	sqlStmt["contactInsert"] = "INSERT INTO contacts (adder, addee) VALUES (?, ?)"
	sqlStmt["contactSelect"] = "SELECT adder, addee, confirmed FROM contacts WHERE adder = ? OR addee = ? ORDER BY time DESC"
	sqlStmt["contactUpdate"] = "UPDATE contacts SET confirmed = 1 WHERE addee = ? AND adder = ?"
	//device
	sqlStmt["deviceInsert"] = "INSERT INTO devices (user_id, device_type, device_id) VALUES (?, ?, ?)"
	sqlStmt["deviceSelect"] = "SELECT user_id, device_type, device_id FROM devices WHERE user_id = ?"
	//Upload
	sqlStmt["userUpload"] = "INSERT INTO uploads (user_id, url) VALUES (?, ?)"
	sqlStmt["uploadExists"] = "SELECT COUNT(*) FROM uploads WHERE user_id = ? AND url = ?"
	for k, str := range sqlStmt {
		stmt[k], err = db.Prepare(str)
		if err != nil {
			return
		}
	}
	return nil
}

/********************************************************************
		Database functions
********************************************************************/

/********************************************************************
		Network
********************************************************************/

func dbValidateEmail(email string) bool {
	s := stmt["ruleSelect"]
	rows, err := s.Query()
	log.Println("DB hit: validateEmail (rule.networkid, rule.type, rule.value)")
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		rule := new(Rule)
		if err = rows.Scan(&rule.NetworkID, &rule.Type, &rule.Value); err != nil {
			log.Printf("Error getting rule: %v", err)
		}
		if rule.Type == "email" && strings.HasSuffix(email, rule.Value) {
			return (true)
		}
	}
	return (false)
}

func dbGetUserNetworks(id UserId) []Network {
	s := stmt["networkSelect"]
	rows, err := s.Query(id)
	defer rows.Close()
	log.Println("DB hit: getUserNetworks userid (network.id, network.name)")
	nets := make([]Network, 0, 5)
	if err != nil {
		log.Printf("Error querying db: %v", err)
	}
	for rows.Next() {
		var network Network
		err = rows.Scan(&network.Id, &network.Name)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
		} else {
			nets = append(nets, network)
		}
	}
	return (nets)
}

func dbSetNetwork(userId UserId, networkId NetworkId) (err error) {
	_, err = stmt["networkInsert"].Exec(userId, networkId)
	return
}

/********************************************************************
		User
********************************************************************/

func dbRegisterUser(user string, hash []byte, email string) (UserId, error) {
	s := stmt["createUser"]
	res, err := s.Exec(user, hash, email)
	if err != nil && strings.HasPrefix(err.Error(), "Error 1062") { //Note to self:There must be a better way?
		return 0, APIerror{"Username or email address already taken"}
	} else if err != nil {
		return 0, err
	} else {
		id, _ := res.LastInsertId()
		return UserId(id), nil
	}
}

func dbGetUser(id UserId) (user User, err error) {
	var av sql.NullString
	s := stmt["userSelect"]
	err = s.QueryRow(id).Scan(&user.Id, &user.Name, &av)
	log.Println("DB hit: dbGetUser id(user.Name, user.Id, user.Avatar)")
	if av.Valid {
		user.Avatar = av.String
	}
	if err != nil {
		return user, err
	} else {
		return user, nil
	}
}

func dbGetProfile(id UserId) (user Profile, err error) {
	var av, desc sql.NullString
	s := stmt["profileSelect"]
	err = s.QueryRow(id).Scan(&user.Name, &desc, &av)
	log.Println("DB hit: getProfile id(user.Name, user.Desc)")
	if av.Valid {
		user.Avatar = av.String
	}
	if desc.Valid {
		user.Desc = desc.String
	}
	user.Id = id
	nets := getUserNetworks(user.Id)
	user.Network = nets[0]
	return user, err
}

func dbSetProfileImage(id UserId, url string) (err error) {
	_, err = stmt["setAvatar"].Exec(url, id)
	return
}

/********************************************************************
		Conversation
********************************************************************/

func dbCreateConversation(id UserId, participants []User) (conversation Conversation, err error) {
	s := stmt["conversationInsert"]
	r, _ := s.Exec(id)
	cId, _ := r.LastInsertId()
	conversation.Id = ConversationId(cId)
	if err != nil {
		return
	}
	log.Println("DB hit: createConversation (user.Name, user.Id)")
	sta := stmt["participantInsert"]
	for _, u := range participants {
		_, err = sta.Exec(conversation.Id, u.Id)
		if err != nil {
			return
		}
	}
	conversation.Participants = participants
	return
}

func dbRandomPartners(id UserId, count int) (partners []User, err error) {
	s := stmt["randomSelect"]
	rows, err := s.Query()
	if err != nil {
		return
	}
	defer rows.Close()
	for count > 0 {
		rows.Next()
		var user User
		if err = rows.Scan(&user.Id, &user.Name); err != nil {
			return
		} else {
			if user.Id != id {
				partners = append(partners, user)
				count--
			}
		}
	}
	return
}

func dbUpdateConversation(id ConversationId) (err error) {
	s := stmt["conversationUpdate"]
	_, err = s.Exec(id)
	log.Println("DB hit: updateConversation convid ")
	if err != nil {
		log.Printf("Error: %v", err)
	}
	return err
}

func dbGetConversations(user_id UserId, start int64) (conversations []ConversationSmall, err error) {
	s := stmt["conversationSelect"]
	rows, err := s.Query(user_id, start)
	log.Println("DB hit: getConversations user_id, start (conversation.id)")
	if err != nil {
		return conversations, err
	}
	defer rows.Close()
	for rows.Next() {
		var conv ConversationSmall
		var t string
		err = rows.Scan(&conv.Conversation.Id, &t)
		if err != nil {
			return conversations, err
		}
		conv.LastActivity, _ = time.Parse(MysqlTime, t)
		conv.Conversation.Participants = getParticipants(conv.Id)
		LastMessage, err := getLastMessage(conv.Id)
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		conversations = append(conversations, conv)
	}
	return conversations, nil
}

func dbGetConversation(convId ConversationId) (conversation ConversationAndMessages, err error) {
	conversation.Id = convId
	conversation.Participants = getParticipants(convId)
	conversation.Messages, err = dbGetMessages(convId, 0, "start")
	return
}

func dbGetParticipants(conv ConversationId) []User {
	s := stmt["participantSelect"]
	rows, err := s.Query(conv)
	log.Println("DB hit: getParticipants convid (user.id)")
	if err != nil {
		log.Printf("Error getting participant: %v", err)
	}
	defer rows.Close()
	participants := make([]User, 0, 5)
	for rows.Next() {
		var id UserId
		err = rows.Scan(&id)
		user, err := getUser(id)
		if err == nil {
			participants = append(participants, user)
		}
	}
	return (participants)
}

func dbGetLastMessage(id ConversationId) (message Message, err error) {
	var timeString string
	var by UserId
	s := stmt["lastMessageSelect"]
	err = s.QueryRow(id).Scan(&message.Id, &by, &message.Text, &timeString, &message.Seen)
	log.Println("DB hit: dbGetLastMessage convid (message.id, message.by, message.text, message.time, message.seen)")
	if err != nil {
		return message, err
	} else {
		message.By, err = getUser(by)
		if err != nil {
			log.Printf("error getting user %d %v", by, err)
		}
		message.Time, _ = time.Parse(MysqlTime, timeString)

		return message, nil
	}
}

/********************************************************************
		Post
********************************************************************/

func dbAddPost(userId UserId, text string) (postId PostId, err error) {
	networks := getUserNetworks(userId)
	s := stmt["postInsert"]
	res, err := s.Exec(userId, text, networks[0].Id)
	if err != nil {
		return 0, err
	}
	_postId, err := res.LastInsertId()
	postId = PostId(_postId)
	if err != nil {
		return 0, err
	}
	return postId, nil
}

func dbGetPosts(netId NetworkId, start int64, count int) (posts []PostSmall, err error) {
	s := stmt["wallSelect"]
	rows, err := s.Query(netId, start, count)
	defer rows.Close()
	log.Println("DB hit: getPosts netId(post.id, post.by, post.time, post.texts)")
	if err != nil {
		return
	}
	for rows.Next() {
		var post PostSmall
		var t string
		var by UserId
		err = rows.Scan(&post.Id, &by, &t, &post.Text)
		if err != nil {
			return posts, err
		}
		post.Time, err = time.Parse(MysqlTime, t)
		if err != nil {
			return posts, err
		}
		post.By, err = getUser(by)
		if err != nil {
			return posts, err
		}
		post.CommentCount = getCommentCount(post.Id)
		post.Images = getPostImages(post.Id)
		posts = append(posts, post)
	}
	return
}

func dbGetPostImages(postId PostId) (images []string, err error) {
	s := stmt["imageSelect"]
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

func dbAddPostImage(postId PostId, url string) (err error) {
	_, err = stmt["imageInsert"].Exec(postId, url)
	return
}

func dbCreateComment(postId PostId, userId UserId, text string) (commId CommentId, err error) {
	s := stmt["commentInsert"]
	if res, err := s.Exec(postId, userId, text); err == nil {
		cId, err := res.LastInsertId()
		commId = CommentId(cId)
		return commId, err
	} else {
		return 0, err
	}
}

func dbGetComments(postId PostId, start int64, count int) (comments []Comment, err error) {
	s := stmt["commentSelect"]
	rows, err := s.Query(postId, start, count)
	log.Println("DB hit: getComments postid, start(comment.id, comment.by, comment.text, comment.time)")
	if err != nil {
		return comments, err
	}
	defer rows.Close()
	for rows.Next() {
		var comment Comment
		comment.Post = postId
		var timeString string
		var by UserId
		err := rows.Scan(&comment.Id, &by, &comment.Text, &timeString)
		if err != nil {
			return comments, err
		}
		comment.Time, _ = time.Parse(MysqlTime, timeString)
		comment.By, err = getUser(by)
		if err != nil {
			log.Printf("error getting user %d %v", by, err)
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func dbGetCommentCount(id PostId) (count int) {
	s := stmt["commentCountSelect"]
	err := s.QueryRow(id).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

/********************************************************************
		Message
********************************************************************/

func dbAddMessage(convId ConversationId, userId UserId, text string) (id MessageId, err error) {
	log.Printf("Adding message to db: %d, %d %s", convId, userId, text)
	s := stmt["messageInsert"]
	res, err := s.Exec(convId, userId, text)
	if err != nil {
		return 0, err
	}
	_id, err := res.LastInsertId()
	id = MessageId(_id)
	return
}

func dbGetMessages(convId ConversationId, index int64, sel string) (messages []Message, err error) {
	conf := GetConfig()
	var s *sql.Stmt
	switch {
	case sel == "after":
		s = stmt["messageSelectAfter"]
	case sel == "before":
		s = stmt["messageSelectBefore"]
	case sel == "start":
		s = stmt["messageSelect"]
	}
	rows, err := s.Query(convId, index, conf.MessagePageSize)
	log.Println("DB hit: getMessages convid, start (message.id, message.by, message.text, message.time, message.seen)")
	if err != nil {
		log.Printf("%v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var message Message
		var timeString string
		var by UserId
		err := rows.Scan(&message.Id, &by, &message.Text, &timeString, &message.Seen)
		if err != nil {
			log.Printf("%v", err)
		}
		message.Time, err = time.Parse(MysqlTime, timeString)
		if err != nil {
			log.Printf("%v", err)
		}
		message.By, err = getUser(by)
		if err != nil {
			//should only happen if a message is from a non-existent user
			//(or the db is fucked :))
			log.Println(err)
		}
		messages = append(messages, message)
	}
	return
}

//dbMarkRead sets all messages read in conversation convId
//that are a) not from user id and b) sent upto and including upTo.
func dbMarkRead(id UserId, convId ConversationId, upTo MessageId) (err error) {
	_, err = stmt["messagesRead"].Exec(convId, upTo, id)
	return
}

/********************************************************************
		Token
********************************************************************/

func dbTokenExists(id UserId, token string) bool {
	var expiry string
	s := stmt["tokenSelect"]
	err := s.QueryRow(id, token).Scan(&expiry)
	if err != nil {
		return (false)
	} else {
		t, _ := time.Parse(MysqlTime, expiry)
		if t.After(time.Now()) {
			return (true)
		}
		return (false)
	}
}

func dbAddToken(token Token) (err error) {
	s := stmt["tokenInsert"]
	_, err = s.Exec(token.UserId, token.Token, token.Expiry)
	return
}

/********************************************************************
		Contact
********************************************************************/

func dbAddContact(adder UserId, addee UserId) (err error) {
	log.Println("DB hit: addContact")
	s := stmt["contactInsert"]
	_, err = s.Exec(adder, addee)
	return
}

func dbGetContacts(user UserId) (contacts []Contact, err error) {
	s := stmt["contactSelect"]
	rows, err := s.Query(user, user)
	log.Println("DB hit: GetContacts")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var contact Contact
		var adder, addee UserId
		var confirmed bool
		err = rows.Scan(&adder, &addee, &confirmed)
		if err != nil {
			return
		}
		switch {
		case adder == user:
			contact.User, err = getUser(addee)
			if err != nil {
				return
			}
			contact.YouConfirmed = true
			contact.TheyConfirmed = confirmed
		case addee == user:
			contact.User, err = getUser(adder)
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

func dbUpdateContact(user UserId, contact UserId) (err error) {
	s := stmt["contactUpdate"]
	_, err = s.Exec(user, contact)
	return
}

/********************************************************************
		Device
********************************************************************/

func dbAddDevice(user UserId, deviceType string, deviceId string) (err error) {
	s := stmt["deviceInsert"]
	_, err = s.Exec(user, deviceType, deviceId)
	return
}

func dbGetDevices(user UserId) (devices []Device, err error) {
	s := stmt["deviceSelect"]
	rows, err := s.Query(user)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		device := Device{User: user}
		if err = rows.Scan(&device.Type, &device.Id); err != nil {
			return
		}
		devices = append(devices, device)
	}
	return
}

/********************************************************************
		Upload
********************************************************************/

func dbAddUpload(user UserId, url string) (err error) {
	_, err = stmt["userUpload"].Exec(user, url)
	return
}

func dbUploadExists(user UserId, url string) (exists bool, err error) {
	err = stmt["uploadExists"].QueryRow(user, url).Scan(&exists)
	return
}
