package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"time"
	"strings"
)

const (
	ruleSelect         = "SELECT network_id, rule_type, rule_value FROM net_rules"
	createUser         = "INSERT INTO users(name, password, email) VALUES (?,?,?)"
	PassSelect         = "SELECT id, password FROM users WHERE name = ?"
	randomSelect       = "SELECT id, name FROM users ORDER BY RAND()"
	conversationInsert = "INSERT INTO conversations (initiator, last_mod) VALUES (?, NOW())"
	userSelect         = "SELECT id, name FROM users WHERE id=?"
	participantInsert  = "INSERT INTO conversation_participants (conversation_id, participant_id) VALUES (?,?)"
	postInsert         = "INSERT INTO wall_posts(`by`, `text`, network_id) VALUES (?,?,?)"
	wallSelect         = "SELECT id, `by`, time, text FROM wall_posts WHERE network_id = ? ORDER BY time DESC LIMIT ?, ?"
	networkSelect      = "SELECT user_network.network_id, network.name FROM user_network INNER JOIN network ON user_network.network_id = network.id WHERE user_id = ?"
	conversationSelect = "SELECT conversation_participants.conversation_id, conversations.last_mod FROM conversation_participants JOIN conversations ON conversation_participants.conversation_id = conversations.id WHERE participant_id = ? ORDER BY conversations.last_mod DESC LIMIT ?, 20"
	participantSelect  = "SELECT participant_id, users.name FROM conversation_participants JOIN users ON conversation_participants.participant_id = users.id WHERE conversation_id=?"
	messageInsert      = "INSERT INTO chat_messages (conversation_id, `from`, `text`) VALUES (?,?,?)"
	messageSelect      = "SELECT id, `from`, text, timestamp, seen FROM chat_messages WHERE conversation_id = ? ORDER BY timestamp DESC LIMIT ?, ?"
	messageSelectAfter = "SELECT id, `from`, text, timestamp, seen FROM chat_messages WHERE conversation_id = ? AND id > ? ORDER BY timestamp DESC LIMIT ?"
	tokenInsert        = "INSERT INTO tokens (user_id, token, expiry) VALUES (?, ?, ?)"
	tokenSelect        = "SELECT expiry FROM tokens WHERE user_id = ? AND token = ?"
	conversationUpdate = "UPDATE conversations SET last_mod = NOW() WHERE id = ?"
	commentInsert      = "INSERT INTO post_comments (post_id, `by`, text) VALUES (?, ?, ?)"
	commentSelect      = "SELECT id, `by`, text, timestamp FROM post_comments WHERE post_id = ? ORDER BY timestamp DESC LIMIT ?, ?"
	lastMessageSelect  = "SELECT id, `from`, text, timestamp, seen FROM chat_messages WHERE conversation_id = ? ORDER BY timestamp DESC LIMIT 1"
	commentCountSelect = "SELECT COUNT(*) FROM post_comments WHERE post_id = ?"
	profileSelect      = "SELECT name, `desc`, avatar FROM users WHERE id = ?"
	imageSelect        = "SELECT url FROM post_images WHERE post_id = ?"
)

var (
	ruleStmt               *sql.Stmt
	registerStmt           *sql.Stmt
	passStmt               *sql.Stmt
	randomStmt             *sql.Stmt
	userStmt               *sql.Stmt
	conversationStmt       *sql.Stmt
	participantStmt        *sql.Stmt
	networkStmt            *sql.Stmt
	postStmt               *sql.Stmt
	wallSelectStmt         *sql.Stmt
	conversationSelectStmt *sql.Stmt
	participantSelectStmt  *sql.Stmt
	messageInsertStmt      *sql.Stmt
	messageSelectStmt      *sql.Stmt
	messageSelectAfterStmt *sql.Stmt
	tokenInsertStmt        *sql.Stmt
	tokenSelectStmt        *sql.Stmt
	conversationUpdateStmt *sql.Stmt
	commentInsertStmt      *sql.Stmt
	commentSelectStmt      *sql.Stmt
	lastMessageSelectStmt  *sql.Stmt
	commentCountSelectStmt *sql.Stmt
	profileSelectStmt      *sql.Stmt
	imageSelectStmt        *sql.Stmt
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
	ruleStmt, err = db.Prepare(ruleSelect)
	if err != nil {
		return
	}
	registerStmt, err = db.Prepare(createUser)
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
	conversationStmt, err = db.Prepare(conversationInsert)
	if err != nil {
		return
	}
	userStmt, err = db.Prepare(userSelect)
	if err != nil {
		return
	}
	participantStmt, err = db.Prepare(participantInsert)
	if err != nil {
		return
	}
	postStmt, err = db.Prepare(postInsert)
	if err != nil {
		return
	}
	wallSelectStmt, err = db.Prepare(wallSelect)
	if err != nil {
		return
	}
	networkStmt, err = db.Prepare(networkSelect)
	if err != nil {
		return
	}
	conversationSelectStmt, err = db.Prepare(conversationSelect)
	if err != nil {
		return
	}
	participantSelectStmt, err = db.Prepare(participantSelect)
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
	tokenInsertStmt, err = db.Prepare(tokenInsert)
	if err != nil {
		return
	}
	tokenSelectStmt, err = db.Prepare(tokenSelect)
	if err != nil {
		return
	}
	conversationUpdateStmt, err = db.Prepare(conversationUpdate)
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
	lastMessageSelectStmt, err = db.Prepare(lastMessageSelect)
	if err != nil {
		return
	}
	commentCountSelectStmt, err = db.Prepare(commentCountSelect)
	if err != nil {
		return
	}
	profileSelectStmt, err = db.Prepare(profileSelect)
	if err != nil {
		return
	}
	imageSelectStmt, err = db.Prepare(imageSelect)
	if err != nil {
		return
	}
	return nil
}

/********************************************************************
Database functions
********************************************************************/

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
	if res, err := commentInsertStmt.Exec(commId, userId, text); err != nil {
		cId, err := res.LastInsertId()
		commId = CommentId(cId)
		return commId, err
	} else {
		return 0, err
	}
}

