//db handles 
package db

import (
	"database/sql"
	"github.com/draaglom/GleepostAPI/lib/gp"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strings"
	"time"
)

const (
	//For parsing
	mysqlTime = "2006-01-02 15:04:05"
)

var (
	sqlStmt map[string]string
	stmt    map[string]*sql.Stmt
)

func keepalive(db *sql.DB) {
	tick := time.Tick(1 * time.Hour)
	conf := gp.GetConfig()
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

func init() {
	conf := gp.GetConfig()
	db, err := sql.Open("mysql", conf.ConnectionString())
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	db.SetMaxIdleConns(conf.Mysql.MaxConns)
	err = prepare(db)
	if err != nil {
		log.Fatal(err)
	}
	go keepalive(db)
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
	sqlStmt["randomSelect"] = "SELECT id, name, avatar FROM users LEFT JOIN user_network ON id = user_id WHERE network_id = ? ORDER BY RAND()"
	sqlStmt["setAvatar"] = "UPDATE users SET avatar = ? WHERE id = ?"
	sqlStmt["setBusy"] = "UPDATE users SET busy = ? WHERE id = ?"
	sqlStmt["getBusy"] = "SELECT busy FROM users WHERE id = ?"
	sqlStmt["idFromFacebook"] = "SELECT user_id FROM facebook WHERE fb_id = ? AND user_id IS NOT NULL"
	sqlStmt["fbInsert"] = "INSERT INTO facebook (fb_id, email) VALUES (?, ?)"
	sqlStmt["selectFBemail"] = "SELECT email FROM facebook WHERE fb_id = ?"
	sqlStmt["fbInsertVerification"] = "REPLACE INTO facebook_verification (fb_id, token) VALUES (?, ?)"
	sqlStmt["fbSetGPUser"] = "UPDATE facebook SET user_id = ? WHERE fb_id = ?"
	sqlStmt["insertVerification"] = "REPLACE INTO `verification` (user_id, token) VALUES (?, ?)"
	sqlStmt["verificationExists"] = "SELECT user_id FROM verification WHERE token = ?"
	sqlStmt["verify"] = "UPDATE users SET verified = 1 WHERE id = ?"
	sqlStmt["emailSelect"] = "SELECT email FROM users WHERE id = ?"
	sqlStmt["userWithEmail"] = "SELECT id FROM users WHERE email = ?"
	//Conversation
	sqlStmt["conversationInsert"] = "INSERT INTO conversations (initiator, last_mod) VALUES (?, NOW())"
	sqlStmt["conversationUpdate"] = "UPDATE conversations SET last_mod = NOW() WHERE id = ?"
	sqlStmt["conversationSelect"] = "SELECT conversation_participants.conversation_id, conversations.last_mod FROM conversation_participants JOIN conversations ON conversation_participants.conversation_id = conversations.id WHERE participant_id = ? ORDER BY conversations.last_mod DESC LIMIT ?, ?"
	sqlStmt["conversationActivity"] = "SELECT last_mod FROM conversations WHERE id = ?"
	sqlStmt["conversationExpiry"] = "SELECT expiry, ended FROM conversation_expirations WHERE conversation_id = ?"
	sqlStmt["conversationSetExpiry"] = "REPLACE INTO conversation_expirations (conversation_id, expiry) VALUES (?, ?)"
	sqlStmt["deleteExpiry"] = "DELETE FROM conversation_expirations WHERE conversation_id = ?"
	sqlStmt["endConversation"] = "UPDATE conversation_expirations SET ended = 1 WHERE conversation_id = ?"
	sqlStmt["participantInsert"] = "INSERT INTO conversation_participants (conversation_id, participant_id) VALUES (?,?)"
	sqlStmt["participantSelect"] = "SELECT participant_id FROM conversation_participants JOIN users ON conversation_participants.participant_id = users.id WHERE conversation_id=?"
	sqlStmt["lastMessageSelect"] = "SELECT id, `from`, text, timestamp, seen FROM chat_messages WHERE conversation_id = ? ORDER BY timestamp DESC LIMIT 1"
	//Post
	sqlStmt["postInsert"] = "INSERT INTO wall_posts(`by`, `text`, network_id) VALUES (?,?,?)"
	sqlStmt["wallSelect"] = "SELECT id, `by`, time, text FROM wall_posts WHERE network_id = ? ORDER BY time DESC LIMIT ?, ?"
	sqlStmt["wallSelectAfter"] = "SELECT id, `by`, time, text FROM wall_posts WHERE network_id = ? AND id > ? ORDER BY time DESC LIMIT 0, ?"
	sqlStmt["wallSelectBefore"] = "SELECT id, `by`, time, text FROM wall_posts WHERE network_id = ? AND id < ? ORDER BY time DESC LIMIT 0, ?"
	sqlStmt["imageSelect"] = "SELECT url FROM post_images WHERE post_id = ?"
	sqlStmt["imageInsert"] = "INSERT INTO post_images (post_id, url) VALUES (?, ?)"
	sqlStmt["commentInsert"] = "INSERT INTO post_comments (post_id, `by`, text) VALUES (?, ?, ?)"
	sqlStmt["commentSelect"] = "SELECT id, `by`, text, timestamp FROM post_comments WHERE post_id = ? ORDER BY timestamp DESC LIMIT ?, ?"
	sqlStmt["commentCountSelect"] = "SELECT COUNT(*) FROM post_comments WHERE post_id = ?"
	sqlStmt["postSelect"] = "SELECT `by`, `time`, text FROM wall_posts WHERE id = ?"
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
	sqlStmt["contactExists"] = "SELECT COUNT(*) FROM contacts WHERE adder = ? AND addee = ?"
	//device
	sqlStmt["deviceInsert"] = "REPLACE INTO devices (user_id, device_type, device_id) VALUES (?, ?, ?)"
	sqlStmt["deviceSelect"] = "SELECT user_id, device_type, device_id FROM devices WHERE user_id = ?"
	sqlStmt["deviceDelete"] = "DELETE FROM devices WHERE user_id = ? AND device_id = ?"
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
	return nil
}

/********************************************************************
		Database functions
********************************************************************/

/********************************************************************
		Network
********************************************************************/

func GetRules() (rules []gp.Rule, err error) {
	s := stmt["ruleSelect"]
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

func GetUserNetworks(id gp.UserId) (networks []gp.Network, err error) {
	s := stmt["networkSelect"]
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

func SetNetwork(userId gp.UserId, networkId gp.NetworkId) (err error) {
	_, err = stmt["networkInsert"].Exec(userId, networkId)
	return
}

/********************************************************************
		User
********************************************************************/

func RegisterUser(user string, hash []byte, email string) (gp.UserId, error) {
	s := stmt["createUser"]
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

func GetHash(user string, pass string) (hash []byte, id gp.UserId, err error) {
	s := stmt["passSelect"]
	err = s.QueryRow(user).Scan(&id, &hash)
	return
}

func GetUser(id gp.UserId) (user gp.User, err error) {
	var av sql.NullString
	s := stmt["userSelect"]
	err = s.QueryRow(id).Scan(&user.Id, &user.Name, &av)
	log.Println("DB hit: GetUser id(user.Name, user.Id, user.Avatar)")
	if err != nil {
		return
	}
	if av.Valid {
		user.Avatar = av.String
	}
	if err != nil {
		return user, err
	} else {
		return user, nil
	}
}

//GetProfile fetches a user but DOES NOT GET THEIR NETWORK.
func GetProfile(id gp.UserId) (user gp.Profile, err error) {
	var av, desc sql.NullString
	s := stmt["profileSelect"]
	err = s.QueryRow(id).Scan(&user.Name, &desc, &av)
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
	user.Id = id
	return
}

func SetProfileImage(id gp.UserId, url string) (err error) {
	_, err = stmt["setAvatar"].Exec(url, id)
	return
}

func SetBusyStatus(id gp.UserId, busy bool) (err error) {
	_, err = stmt["setBusy"].Exec(busy, id)
	return
}

func BusyStatus(id gp.UserId) (busy bool, err error) {
	err = stmt["getBusy"].QueryRow(id).Scan(&busy)
	return
}

func UserIdFromFB(fbid uint64) (id gp.UserId, err error) {
	err = stmt["idFromFacebook"].QueryRow(fbid).Scan(&id)
	return
}

func SetVerificationToken(id gp.UserId, token string) (err error) {
	_, err = stmt["insertVerification"].Exec(id, token)
	return
}

func VerificationTokenExists(token string) (id gp.UserId, err error) {
	err = stmt["verificationExists"].QueryRow(token).Scan(&id)
	return
}

func Verify(id gp.UserId) (err error) {
	_, err = stmt["verify"].Exec(id)
	return
}

func GetEmail(id gp.UserId) (email string, err error) {
	err = stmt["emailSelect"].QueryRow(id).Scan(&email)
	return
}

func UserWithEmail(email string) (id gp.UserId, err error) {
	err = stmt["userWithEmail"].QueryRow(email).Scan(&id)
	return
}

func CreateFBUser(fbId uint64, email string) (err error) {
	_, err = stmt["fbInsert"].Exec(fbId, email)
	return
}

func FBUserEmail(fbid uint64) (email string, err error) {
	err = stmt["selectFBemail"].QueryRow(fbid).Scan(&email)
	return
}

func CreateFBVerification(fbid uint64, token string) (err error) {
	_, err = stmt["fbInsertVerification"].Exec(fbid, token)
	return
}

func FBVerificationExists(token string) (fbid uint64, err error) {
	err = stmt["fbVerificationExists"].QueryRow(token).Scan(&fbid)
	return
}

func FBSetGPUser(fbid uint64, userId gp.UserId) (err error) {
	_, err = stmt["fbSetGPUser"].Exec(fbid, userId)
	return
}

/********************************************************************
		Conversation
********************************************************************/

func CreateConversation(id gp.UserId, participants []gp.User, live bool) (conversation gp.Conversation, err error) {
	s := stmt["conversationInsert"]
	r, _ := s.Exec(id)
	cId, _ := r.LastInsertId()
	conversation.Id = gp.ConversationId(cId)
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
	conversation.LastActivity = time.Now().UTC()
	if live {
		conf := gp.GetConfig()
		conversation.Expiry = &gp.Expiry{Time:time.Now().Add(time.Duration(conf.Expiry) * time.Second), Ended:false}
		err = ConversationSetExpiry(conversation.Id, *conversation.Expiry)
	}
	return
}

func RandomPartners(id gp.UserId, count int, network gp.NetworkId) (partners []gp.User, err error) {
	s := stmt["randomSelect"]
	rows, err := s.Query(network)
	if err != nil {
		return
	}
	defer rows.Close()
	for count > 0 {
		rows.Next()
		var user gp.User
		var av sql.NullString
		if err = rows.Scan(&user.Id, &user.Name, &av); err != nil {
			return
		} else {
			if av.Valid {
				user.Avatar = av.String
			}
			if user.Id != id {
				partners = append(partners, user)
				count--
			}
		}
	}
	return
}

func UpdateConversation(id gp.ConversationId) (err error) {
	s := stmt["conversationUpdate"]
	_, err = s.Exec(id)
	log.Println("DB hit: updateConversation convid ")
	if err != nil {
		log.Printf("Error: %v", err)
	}
	return err
}

func GetConversations(userId gp.UserId, start int64, count int) (conversations []gp.ConversationSmall, err error) {
	s := stmt["conversationSelect"]
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
		conv.Participants = GetParticipants(conv.Id)
		LastMessage, err := GetLastMessage(conv.Id)
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		Expiry, err := ConversationExpiry(conv.Id)
		if err == nil {
			conv.Expiry = &Expiry
		}
		conversations = append(conversations, conv)
	}
	return conversations, nil
}

func ConversationActivity(convId gp.ConversationId) (t time.Time, err error) {
	s := stmt["conversationActivity"]
	var tstring string
	err = s.QueryRow(convId).Scan(&tstring)
	if err != nil {
		return
	}
	t, err = time.Parse(mysqlTime, tstring)
	return
}

func ConversationExpiry(convId gp.ConversationId) (expiry gp.Expiry, err error) {
	s := stmt["conversationExpiry"]
	var t string
	err = s.QueryRow(convId).Scan(&t, &expiry.Ended)
	if err != nil {
		return
	}
	expiry.Time, err = time.Parse(mysqlTime, t)
	return
}

func DeleteConversationExpiry(convId gp.ConversationId) (err error) {
	_, err = stmt["deleteExpiry"].Exec(convId)
	return
}

func TerminateConversation(convId gp.ConversationId) (err error) {
	_, err = stmt["endConversation"].Exec(convId)
	return
}

func ConversationSetExpiry(convId gp.ConversationId, expiry gp.Expiry) (err error) {
	s := stmt["conversationSetExpiry"]
	_, err = s.Exec(convId, expiry.Time)
	return
}

func GetConversation(convId gp.ConversationId) (conversation gp.ConversationAndMessages, err error) {
	conf := gp.GetConfig()
	conversation.Id = convId
	conversation.LastActivity, err = ConversationActivity(convId)
	if err != nil {
		return
	}
	conversation.Participants = GetParticipants(convId)
	expiry, err := ConversationExpiry(convId)
	if err == nil {
		conversation.Expiry = &expiry
	}
	conversation.Messages, err = GetMessages(convId, 0, "start", conf.MessagePageSize)
	return
}

//GetParticipants returns all of the participants in conv.
//TODO: Return an error when appropriate
func GetParticipants(conv gp.ConversationId) []gp.User {
	s := stmt["participantSelect"]
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
		user, err := GetUser(id)
		if err == nil {
			participants = append(participants, user)
		}
	}
	return (participants)
}

func GetLastMessage(id gp.ConversationId) (message gp.Message, err error) {
	var timeString string
	var by gp.UserId
	s := stmt["lastMessageSelect"]
	err = s.QueryRow(id).Scan(&message.Id, &by, &message.Text, &timeString, &message.Seen)
	log.Println("DB hit: GetLastMessage convid (message.id, message.by, message.text, message.time, message.seen)")
	if err != nil {
		return message, err
	} else {
		message.By, err = GetUser(by)
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

func AddPost(userId gp.UserId, text string, network gp.NetworkId) (postId gp.PostId, err error) {
	s := stmt["postInsert"]
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

//GetPosts finds posts in the network netId.
func GetPosts(netId gp.NetworkId, index int64, count int, sel string) (posts []gp.PostSmall, err error) {
	var s *sql.Stmt
	switch {
	case sel == "start":
		s = stmt["wallSelect"]
	case sel == "before":
		s = stmt["wallSelectBefore"]
	case sel == "after":
		s = stmt["wallSelectAfter"]
	default:
		return posts, gp.APIerror{"Invalid selector"}
	}
	rows, err := s.Query(netId, index, count)
	defer rows.Close()
	log.Println("DB hit: getPosts netId(post.id, post.by, post.time, post.texts)")
	if err != nil {
		return
	}
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
		post.By, err = GetUser(by)
		if err != nil {
			return posts, err
		}
		post.CommentCount = GetCommentCount(post.Id)
		post.Images, err = GetPostImages(post.Id)
		if err != nil {
			return
		}
		post.LikeCount, err = LikeCount(post.Id)
		if err != nil {
			return
		}
		posts = append(posts, post)
	}
	return
}

func GetPostImages(postId gp.PostId) (images []string, err error) {
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

func AddPostImage(postId gp.PostId, url string) (err error) {
	_, err = stmt["imageInsert"].Exec(postId, url)
	return
}

func CreateComment(postId gp.PostId, userId gp.UserId, text string) (commId gp.CommentId, err error) {
	s := stmt["commentInsert"]
	if res, err := s.Exec(postId, userId, text); err == nil {
		cId, err := res.LastInsertId()
		commId = gp.CommentId(cId)
		return commId, err
	} else {
		return 0, err
	}
}

func GetComments(postId gp.PostId, start int64, count int) (comments []gp.Comment, err error) {
	s := stmt["commentSelect"]
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
		comment.By, err = GetUser(by)
		if err != nil {
			log.Printf("error getting user %d %v", by, err)
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func GetCommentCount(id gp.PostId) (count int) {
	s := stmt["commentCountSelect"]
	err := s.QueryRow(id).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

//GetPost returns the post postId or an error if it doesn't exist.
//TODO: This could return without an embedded user or images array
func GetPost(postId gp.PostId) (post gp.Post, err error) {
	s := stmt["postSelect"]
	post.Id = postId
	var by gp.UserId
	var t string
	err = s.QueryRow(postId).Scan(&by, &t, &post.Text)
	if err != nil {
		return
	}
	post.By, err = GetUser(by)
	if err != nil {
		return
	}
	post.Time, err = time.Parse(mysqlTime, t)
	if err != nil {
		return
	}
	post.Images, err = GetPostImages(postId)
	return
}

/********************************************************************
		Message
********************************************************************/

func AddMessage(convId gp.ConversationId, userId gp.UserId, text string) (id gp.MessageId, err error) {
	log.Printf("Adding message to db: %d, %d %s", convId, userId, text)
	s := stmt["messageInsert"]
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
func GetMessages(convId gp.ConversationId, index int64, sel string, count int) (messages []gp.Message, err error) {
	var s *sql.Stmt
	switch {
	case sel == "after":
		s = stmt["messageSelectAfter"]
	case sel == "before":
		s = stmt["messageSelectBefore"]
	case sel == "start":
		s = stmt["messageSelect"]
	}
	rows, err := s.Query(convId, index, count)
	log.Println("DB hit: getMessages convid, start (message.id, message.by, message.text, message.time, message.seen)")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var message gp.Message
		var timeString string
		var by gp.UserId
		err = rows.Scan(&message.Id, &by, &message.Text, &timeString, &message.Seen)
		if err != nil {
			log.Printf("%v", err)
		}
		message.Time, err = time.Parse(mysqlTime, timeString)
		if err != nil {
			log.Printf("%v", err)
		}
		message.By, err = GetUser(by)
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
func MarkRead(id gp.UserId, convId gp.ConversationId, upTo gp.MessageId) (err error) {
	_, err = stmt["messagesRead"].Exec(convId, upTo, id)
	return
}

/********************************************************************
		Token
********************************************************************/

func TokenExists(id gp.UserId, token string) bool {
	var expiry string
	s := stmt["tokenSelect"]
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

func AddToken(token gp.Token) (err error) {
	s := stmt["tokenInsert"]
	_, err = s.Exec(token.UserId, token.Token, token.Expiry)
	return
}

/********************************************************************
		Contact
********************************************************************/

func AddContact(adder gp.UserId, addee gp.UserId) (err error) {
	log.Println("DB hit: addContact")
	s := stmt["contactInsert"]
	_, err = s.Exec(adder, addee)
	return
}

//GetContacts retrieves all the contacts for user.
//TODO: This could return contacts which doesn't embed a user
func GetContacts(user gp.UserId) (contacts []gp.Contact, err error) {
	s := stmt["contactSelect"]
	rows, err := s.Query(user, user)
	log.Println("DB hit: GetContacts")
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
			contact.User, err = GetUser(addee)
			if err != nil {
				return
			}
			contact.YouConfirmed = true
			contact.TheyConfirmed = confirmed
		case addee == user:
			contact.User, err = GetUser(adder)
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

func UpdateContact(user gp.UserId, contact gp.UserId) (err error) {
	s := stmt["contactUpdate"]
	_, err = s.Exec(user, contact)
	return
}

func ContactRequestExists(adder gp.UserId, addee gp.UserId) (exists bool, err error) {
	err = stmt["contactExists"].QueryRow(adder, addee).Scan(&exists)
	return
}
/********************************************************************
		Device
********************************************************************/

func AddDevice(user gp.UserId, deviceType string, deviceId string) (err error) {
	s := stmt["deviceInsert"]
	_, err = s.Exec(user, deviceType, deviceId)
	return
}

func GetDevices(user gp.UserId) (devices []gp.Device, err error) {
	s := stmt["deviceSelect"]
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

func DeleteDevice(user gp.UserId, device string) (err error) {
	s := stmt["deviceDelete"]
	_, err = s.Exec(user, device)
	return
}

/********************************************************************
		Upload
********************************************************************/

func AddUpload(user gp.UserId, url string) (err error) {
	_, err = stmt["userUpload"].Exec(user, url)
	return
}

func UploadExists(user gp.UserId, url string) (exists bool, err error) {
	err = stmt["uploadExists"].QueryRow(user, url).Scan(&exists)
	return
}

/********************************************************************
		Notification
********************************************************************/

func GetUserNotifications(id gp.UserId) (notifications []interface{}, err error) {
	s := stmt["notificationSelect"]
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
		notification.By, err = GetUser(by)
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

func MarkNotificationsSeen(user gp.UserId, upTo gp.NotificationId) (err error) {
	_, err = stmt["notificationUpdate"].Exec(user, upTo)
	return
}

func CreateNotification(ntype string, by gp.UserId, recipient gp.UserId, isPN bool, post gp.PostId) (notification interface{}, err error) {
	var res sql.Result
	if isPN {
		s := stmt["postNotificationInsert"]
		res, err = s.Exec(ntype, by, recipient, post)
	} else {
		s := stmt["notificationInsert"]
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
		n.By, err = GetUser(by)
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

func CreateLike(user gp.UserId, post gp.PostId) (err error) {
	_, err = stmt["addLike"].Exec(post, user)
	// Suppress duplicate entry errors
	if err != nil {
		if strings.HasPrefix(err.Error(), "Error 1062") {
			err = nil
		}
	}
	return
}

func RemoveLike(user gp.UserId, post gp.PostId) (err error) {
	_, err = stmt["delLike"].Exec(post, user)
	return
}

func GetLikes(post gp.PostId) (likes []gp.Like, err error) {
	rows, err := stmt["likeSelect"].Query(post)
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

func HasLiked(user gp.UserId, post gp.PostId) (liked bool, err error) {
	err = stmt["likeExists"].QueryRow(post, user).Scan(&liked)
	return
}

func LikeCount(post gp.PostId) (count int, err error) {
	err = stmt["likeCount"].QueryRow(post).Scan(&count)
	return
}
