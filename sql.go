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
	//Network
	ruleSelect    = "SELECT network_id, rule_type, rule_value FROM net_rules"
	networkSelect = "SELECT user_network.network_id, network.name FROM user_network INNER JOIN network ON user_network.network_id = network.id WHERE user_id = ?"
	//User
	createUser    = "INSERT INTO users(name, password, email) VALUES (?,?,?)"
	userSelect    = "SELECT id, name FROM users WHERE id=?"
	profileSelect = "SELECT name, `desc`, avatar FROM users WHERE id = ?"
	PassSelect    = "SELECT id, password FROM users WHERE name = ?"
	randomSelect  = "SELECT id, name FROM users ORDER BY RAND()"
	//Conversation
	conversationInsert = "INSERT INTO conversations (initiator, last_mod) VALUES (?, NOW())"
	conversationUpdate = "UPDATE conversations SET last_mod = NOW() WHERE id = ?"
	conversationSelect = "SELECT conversation_participants.conversation_id, conversations.last_mod FROM conversation_participants JOIN conversations ON conversation_participants.conversation_id = conversations.id WHERE participant_id = ? ORDER BY conversations.last_mod DESC LIMIT ?, 20"
	participantInsert  = "INSERT INTO conversation_participants (conversation_id, participant_id) VALUES (?,?)"
	participantSelect  = "SELECT participant_id, users.name FROM conversation_participants JOIN users ON conversation_participants.participant_id = users.id WHERE conversation_id=?"
	lastMessageSelect  = "SELECT id, `from`, text, timestamp, seen FROM chat_messages WHERE conversation_id = ? ORDER BY timestamp DESC LIMIT 1"
	//Post
	postInsert         = "INSERT INTO wall_posts(`by`, `text`, network_id) VALUES (?,?,?)"
	wallSelect         = "SELECT id, `by`, time, text FROM wall_posts WHERE network_id = ? ORDER BY time DESC LIMIT ?, ?"
	imageSelect        = "SELECT url FROM post_images WHERE post_id = ?"
	commentInsert      = "INSERT INTO post_comments (post_id, `by`, text) VALUES (?, ?, ?)"
	commentSelect      = "SELECT id, `by`, text, timestamp FROM post_comments WHERE post_id = ? ORDER BY timestamp DESC LIMIT ?, ?"
	commentCountSelect = "SELECT COUNT(*) FROM post_comments WHERE post_id = ?"
	//Message
	messageInsert      = "INSERT INTO chat_messages (conversation_id, `from`, `text`) VALUES (?,?,?)"
	messageSelect      = "SELECT id, `from`, text, timestamp, seen FROM chat_messages WHERE conversation_id = ? ORDER BY timestamp DESC LIMIT ?, ?"
	messageSelectAfter = "SELECT id, `from`, text, timestamp, seen FROM chat_messages WHERE conversation_id = ? AND id > ? ORDER BY timestamp DESC LIMIT ?"
	//Token
	tokenInsert = "INSERT INTO tokens (user_id, token, expiry) VALUES (?, ?, ?)"
	tokenSelect = "SELECT expiry FROM tokens WHERE user_id = ? AND token = ?"
	//Contact
	contactInsert = "INSERT INTO contacts (adder, addee) VALUES (?, ?)"
	contactSelect = "SELECT adder, addee, confirmed FROM contacts WHERE adder = ? OR addee = ? ORDER BY time DESC"
	contactUpdate = "UPDATE contacts SET confirmed = 1 WHERE addee = ? AND adder = ?"
	//device
	deviceInsert = "INSERT INTO devices (user_id, device_type, device_id) VALUES (?, ?, ?)"
)

var (
	//Network
	ruleStmt    *sql.Stmt
	networkStmt *sql.Stmt
	//User
	registerStmt      *sql.Stmt
	userStmt          *sql.Stmt
	profileSelectStmt *sql.Stmt
	passStmt          *sql.Stmt
	randomStmt        *sql.Stmt
	//Conversation
	conversationStmt       *sql.Stmt
	conversationUpdateStmt *sql.Stmt
	conversationSelectStmt *sql.Stmt
	participantStmt        *sql.Stmt
	participantSelectStmt  *sql.Stmt
	lastMessageSelectStmt  *sql.Stmt
	//Post
	postStmt               *sql.Stmt
	wallSelectStmt         *sql.Stmt
	imageSelectStmt        *sql.Stmt
	commentInsertStmt      *sql.Stmt
	commentSelectStmt      *sql.Stmt
	commentCountSelectStmt *sql.Stmt
	//<essage
	messageInsertStmt      *sql.Stmt
	messageSelectStmt      *sql.Stmt
	messageSelectAfterStmt *sql.Stmt
	//Token
	tokenInsertStmt *sql.Stmt
	tokenSelectStmt *sql.Stmt
	//Contact
	contactInsertStmt *sql.Stmt
	contactSelectStmt *sql.Stmt
	contactUpdateStmt *sql.Stmt
	//Devices
	deviceInsertStmt *sql.Stmt
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
	//Network
	ruleStmt, err = db.Prepare(ruleSelect)
	if err != nil {
		return
	}
	networkStmt, err = db.Prepare(networkSelect)
	if err != nil {
		return
	}
	//User
	registerStmt, err = db.Prepare(createUser)
	if err != nil {
		return
	}
	userStmt, err = db.Prepare(userSelect)
	if err != nil {
		return
	}
	profileSelectStmt, err = db.Prepare(profileSelect)
	if err != nil {
		return
	}
	passStmt, err = db.Prepare(PassSelect)
	if err != nil {
		return
	}
	randomStmt, err = db.Prepare(randomSelect)
	if err != nil {
		return
	}
	//Conversation
	conversationStmt, err = db.Prepare(conversationInsert)
	if err != nil {
		return
	}
	conversationUpdateStmt, err = db.Prepare(conversationUpdate)
	if err != nil {
		return
	}
	conversationSelectStmt, err = db.Prepare(conversationSelect)
	if err != nil {
		return
	}
	participantStmt, err = db.Prepare(participantInsert)
	if err != nil {
		return
	}
	participantSelectStmt, err = db.Prepare(participantSelect)
	if err != nil {
		return
	}
	lastMessageSelectStmt, err = db.Prepare(lastMessageSelect)
	if err != nil {
		return
	}
	//Post
	postStmt, err = db.Prepare(postInsert)
	if err != nil {
		return
	}
	wallSelectStmt, err = db.Prepare(wallSelect)
	if err != nil {
		return
	}
	imageSelectStmt, err = db.Prepare(imageSelect)
	if err != nil {
		return
	}
	commentInsertStmt, err = db.Prepare(commentInsert)
	if err != nil {
		return
	}
	commentSelectStmt, err = db.Prepare(commentSelect)
	if err != nil {
		return
	}
	commentCountSelectStmt, err = db.Prepare(commentCountSelect)
	if err != nil {
		return
	}
	messageInsertStmt, err = db.Prepare(messageInsert)
	if err != nil {
		return
	}
	messageSelectStmt, err = db.Prepare(messageSelect)
	if err != nil {
		return
	}
	messageSelectAfterStmt, err = db.Prepare(messageSelectAfter)
	if err != nil {
		return
	}
	//Token
	tokenInsertStmt, err = db.Prepare(tokenInsert)
	if err != nil {
		return
	}
	tokenSelectStmt, err = db.Prepare(tokenSelect)
	if err != nil {
		return
	}
	//Contact
	contactInsertStmt, err = db.Prepare(contactInsert)
	if err != nil {
		return
	}
	contactSelectStmt, err = db.Prepare(contactSelect)
	if err != nil {
		return
	}
	contactUpdateStmt, err = db.Prepare(contactUpdate)
	if err != nil {
		return
	}
	//Devices
	deviceInsertStmt, err = db.Prepare(deviceInsert)
	if err != nil {
		return
	}
	return nil
}

/********************************************************************
Database functions
********************************************************************/

func dbUpdateContact(user UserId, contact UserId) (err error) {
	_, err = contactUpdateStmt.Exec(user, contact)
	return
}

func dbGetContacts(user UserId) (contacts []Contact, err error) {
	rows, err := contactSelectStmt.Query(user, user)
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
			contact.YouConfirmed = confirmed
			contact.TheyConfirmed = true
		}
		contacts = append(contacts, contact)
	}
	return
}

func dbAddContact(adder UserId, addee UserId) (err error) {
	log.Println("DB hit: addContact")
	_, err = contactInsertStmt.Exec(adder, addee)
	return
}

func dbGetMessagesAfter(convId ConversationId, after int64) (messages []Message, err error) {
	conf := GetConfig()
	rows, err := messageSelectAfterStmt.Query(convId, after, conf.MessagePageSize)
	log.Println("DB hit: getMessages convid, after (message.id, message.by, message.text, message.time, message.seen)")
	if err != nil {
		return
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

func dbGetComments(postId PostId, start int64, count int) (comments []Comment, err error) {
	rows, err := commentSelectStmt.Query(postId, start, count)
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

func dbAddPost(userId UserId, text string) (postId PostId, err error) {
	networks := getUserNetworks(userId)
	res, err := postStmt.Exec(userId, text, networks[0].Id)
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

func dbAddMessage(convId ConversationId, userId UserId, text string) (id MessageId, err error) {
	res, err := messageInsertStmt.Exec(convId, userId, text)
	if err != nil {
		return 0, err
	}
	_id, err := res.LastInsertId()
	id = MessageId(_id)
	return
}

func dbUpdateConversation(id ConversationId) (err error) {
	_, err = conversationUpdateStmt.Exec(id)
	log.Println("DB hit: updateConversation convid ")
	if err != nil {
		log.Printf("Error: %v", err)
	}
	return err
}

func dbGetCommentCount(id PostId) (count int) {
	err := commentCountSelectStmt.QueryRow(id).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

func dbGetLastMessage(id ConversationId) (message Message, err error) {
	var timeString string
	var by UserId
	err = lastMessageSelectStmt.QueryRow(id).Scan(&message.Id, &by, &message.Text, &timeString, &message.Seen)
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

func dbValidateEmail(email string) bool {
	rows, err := ruleStmt.Query()
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

func dbRegisterUser(user string, hash []byte, email string) (UserId, error) {
	res, err := registerStmt.Exec(user, hash, email)
	if err != nil && strings.HasPrefix(err.Error(), "Error 1062") { //Note to self:There must be a better way?
		return 0, APIerror{"Username or email address already taken"}
	} else if err != nil {
		return 0, err
	} else {
		id, _ := res.LastInsertId()
		return UserId(id), nil
	}
}

func dbGetUserNetworks(id UserId) []Network {
	rows, err := networkStmt.Query(id)
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

func dbGetParticipants(conv ConversationId) []User {
	rows, err := participantSelectStmt.Query(conv)
	log.Println("DB hit: getParticipants convid (message.id, message.by, message.text, message.time, message.seen)")
	if err != nil {
		log.Printf("Error getting participant: %v", err)
	}
	defer rows.Close()
	participants := make([]User, 0, 5)
	for rows.Next() {
		var user User
		err = rows.Scan(&user.Id, &user.Name)
		participants = append(participants, user)
	}
	return (participants)
}

func dbGetMessages(convId ConversationId, start int64) (messages []Message, err error) {
	conf := GetConfig()
	rows, err := messageSelectStmt.Query(convId, start, conf.MessagePageSize)
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

func dbGetConversations(user_id UserId, start int64) (conversations []ConversationSmall, err error) {
	rows, err := conversationSelectStmt.Query(user_id, start)
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

func dbCreateConversation(id UserId, nParticipants int) (conversation Conversation, err error) {
	r, _ := conversationStmt.Exec(id)
	cId, _ := r.LastInsertId()
	conversation.Id = ConversationId(cId)
	participants := make([]User, 0, 10)
	user, err := getUser(id)
	if err != nil {
		return
	}
	participants = append(participants, user)
	nParticipants--

	rows, err := randomStmt.Query()
	log.Println("DB hit: createConversation (user.Name, user.Id)")
	if err != nil {
		return
	}
	defer rows.Close()
	for nParticipants > 0 {
		rows.Next()
		if err = rows.Scan(&user.Id, &user.Name); err != nil {
			return
		} else {
			participants = append(participants, user)
			nParticipants--
		}
	}
	for _, u := range participants {
		_, err = participantStmt.Exec(conversation.Id, u.Id)
		if err != nil {
			return
		}
	}
	conversation.Participants = participants
	return
}

func dbGetUser(id UserId) (user User, err error) {
	err = userStmt.QueryRow(id).Scan(&user.Id, &user.Name)
	log.Println("DB hit: dbGetUser id(user.Name, user.Id)")
	if err != nil {
		return user, err
	} else {
		return user, nil
	}
}

func dbGetPosts(netId NetworkId, start int64, count int) (posts []PostSmall, err error) {
	rows, err := wallSelectStmt.Query(netId, start, count)
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
	rows, err := imageSelectStmt.Query(postId)
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

func dbGetProfile(id UserId) (user Profile, err error) {
	err = profileSelectStmt.QueryRow(id).Scan(&user.User.Name, &user.Desc, &user.Avatar)
	log.Println("DB hit: getProfile id(user.Name, user.Desc)")
	user.User.Id = id
	//at the moment all the urls in the db aren't real ones :/
	user.Avatar = "https://gleepost.com/" + user.Avatar
	nets := getUserNetworks(user.User.Id)
	user.Network = nets[0]
	return user, err
}

func dbCreateComment(postId PostId, userId UserId, text string) (commId CommentId, err error) {
	if res, err := commentInsertStmt.Exec(postId, userId, text); err == nil {
		cId, err := res.LastInsertId()
		commId = CommentId(cId)
		return commId, err
	} else {
		return 0, err
	}
}

func dbAddDevice(user UserId, deviceType string, deviceId string) (err error) {
	_, err = deviceInsertStmt.Exec(user, deviceType, deviceId)
	return
}
