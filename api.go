package main

import (
	"code.google.com/p/go.crypto/bcrypt"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"log"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type User struct {
	Id   uint64 `json:"id"`
	Name string `json:"username"`
}

type Profile struct {
	User
	Desc string `json:"tagline"`
}

type Network struct {
	Id   uint64
	Name string
}

type Message struct {
	Id   uint64    `json:"id"`
	By   User      `json:"by"`
	Text string    `json:"text"`
	Time time.Time `json:"timestamp"`
	Seen bool      `json:"seen"`
}

type RedisMessage struct {
	Message
	Conversation uint64 `json:"conversation_id"`
}

type Token struct {
	UserId uint64    `json:"id"`
	Token  string    `json:"value"`
	Expiry time.Time `json:"expiry"`
}

type Post struct {
	Id        uint64     `json:"id"`
	By        User       `json:"by"`
	Time      time.Time  `json:"timestamp"`
	Text      string     `json:"text"`
	Comments  []Comment  `json:"comments"`
	LikeHates []LikeHate `json:"like_hate"`
}

type PostSmall struct {
	Id           uint64    `json:"id"`
	By           User      `json:"by"`
	Time         time.Time `json:"timestamp"`
	Text         string    `json:"text"`
	CommentCount int       `json:"comments"`
	LikeCount    int       `json:"likes"`
	HateCount    int       `json:"hates"`
}

type Comment struct {
	Id   uint64    `json:"id"`
	Post uint64    `json:"-"`
	By   User      `json:"by"`
	Time time.Time `json:"timestamp"`
	Text string    `json:"text"`
}

type LikeHate struct {
	Value  bool // true is like, false is hate
	UserID uint64
}

type Rule struct {
	NetworkID int
	Type      string
	Value     string
}

type Conversation struct {
	Id           int64    `json:"id"`
	Participants []User   `json:"participants"`
	LastMessage  *Message `json:"mostRecentMessage"`
}

type ConversationAndMessages struct {
	Id           int64     `json:"id"`
	Participants []User    `json:"participants"`
	Messages     []Message `json:"messages"`
}

type APIerror struct {
	Reason string `json:"error"`
}

func (e APIerror) Error() string {
	return e.Reason
}

func jsonResp(w http.ResponseWriter, resp []byte, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(resp)
}

const (
	ruleSelect         = "SELECT network_id, rule_type, rule_value FROM net_rules"
	createUser         = "INSERT INTO users(name, password, email) VALUES (?,?,?)"
	ConnectionString   = "gp:PnOaw3XzP6Tlq6fWvvVv@tcp(localhost:3306)/gleepost?charset=utf8"
	PassSelect         = "SELECT id, password FROM users WHERE name = ?"
	randomSelect       = "SELECT id, name FROM users ORDER BY RAND()"
	conversationInsert = "INSERT INTO conversations (initiator, last_mod) VALUES (?, NOW())"
	userSelect         = "SELECT id, name FROM users WHERE id=?"
	participantInsert  = "INSERT INTO conversation_participants (conversation_id, participant_id) VALUES (?,?)"
	postInsert         = "INSERT INTO wall_posts(`by`, `text`, network_id) VALUES (?,?,?)"
	wallSelect         = "SELECT id, `by`, time, text FROM wall_posts WHERE network_id = ? ORDER BY time DESC LIMIT 20"
	networkSelect      = "SELECT user_network.network_id, network.name FROM user_network INNER JOIN network ON user_network.network_id = network.id WHERE user_id = ?"
	conversationSelect = "SELECT conversation_participants.conversation_id FROM conversation_participants JOIN conversations ON conversation_participants.conversation_id = conversations.id WHERE participant_id = ? ORDER BY conversations.last_mod DESC LIMIT ?, 20"
	participantSelect  = "SELECT participant_id, users.name FROM conversation_participants JOIN users ON conversation_participants.participant_id = users.id WHERE conversation_id=?"
	messageInsert      = "INSERT INTO chat_messages (conversation_id, `from`, `text`) VALUES (?,?,?)"
	messageSelect      = "SELECT id, `from`, text, timestamp, seen FROM chat_messages WHERE conversation_id = ? ORDER BY timestamp DESC LIMIT ?, 20"
	tokenInsert        = "INSERT INTO tokens (user_id, token, expiry) VALUES (?, ?, ?)"
	tokenSelect        = "SELECT expiry FROM tokens WHERE user_id = ? AND token = ?"
	conversationUpdate = "UPDATE conversations SET last_mod = NOW() WHERE id = ?"
	commentInsert      = "INSERT INTO post_comments (post_id, `by`, text) VALUES (?, ?, ?)"
	commentSelect      = "SELECT id, `by`, text, timestamp FROM post_comments WHERE post_id = ? ORDER BY timestamp DESC LIMIT ?, 20"
	lastMessageSelect  = "SELECT id, `from`, text, timestamp, seen FROM chat_messages WHERE conversation_id = ? ORDER BY timestamp DESC LIMIT 1"
	commentCountSelect = "SELECT COUNT(*) FROM post_comments WHERE post_id = ?"
	profileSelect      = "SELECT name, `desc` FROM users WHERE id = ?"
	MaxConnectionCount = 100
	UrlBase            = "/api/v0.9"
	LoginOverride      = false
	MysqlTime          = "2006-01-02 15:04:05"
	RedisProto         = "tcp"
	RedisAddress       = "146.185.138.53:6379"
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
	tokenInsertStmt        *sql.Stmt
	tokenSelectStmt        *sql.Stmt
	conversationUpdateStmt *sql.Stmt
	commentInsertStmt      *sql.Stmt
	commentSelectStmt      *sql.Stmt
	lastMessageSelectStmt  *sql.Stmt
	commentCountSelectStmt *sql.Stmt
	profileSelectStmt      *sql.Stmt
	pool                   *redis.Pool
)

func keepalive(db *sql.DB) {
	tick := time.Tick(1 * time.Hour)
	for {
		<-tick
		err := db.Ping()
		if err != nil {
			log.Print(err)
			db, err = sql.Open("mysql", ConnectionString)
			if err != nil {
				log.Fatalf("Error opening database: %v", err)
			}
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	db, err := sql.Open("mysql", ConnectionString)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	db.SetMaxIdleConns(MaxConnectionCount)
	ruleStmt, err = db.Prepare(ruleSelect)
	if err != nil {
		log.Fatal(err)
	}
	registerStmt, err = db.Prepare(createUser)
	if err != nil {
		log.Fatal(err)
	}
	passStmt, err = db.Prepare(PassSelect)
	if err != nil {
		log.Fatal(err)
	}
	randomStmt, err = db.Prepare(randomSelect)
	if err != nil {
		log.Fatal(err)
	}
	conversationStmt, err = db.Prepare(conversationInsert)
	if err != nil {
		log.Fatal(err)
	}
	userStmt, err = db.Prepare(userSelect)
	if err != nil {
		log.Fatal(err)
	}
	participantStmt, err = db.Prepare(participantInsert)
	if err != nil {
		log.Fatal(err)
	}
	postStmt, err = db.Prepare(postInsert)
	if err != nil {
		log.Fatal(err)
	}
	wallSelectStmt, err = db.Prepare(wallSelect)
	if err != nil {
		log.Fatal(err)
	}
	networkStmt, err = db.Prepare(networkSelect)
	if err != nil {
		log.Fatal(err)
	}
	conversationSelectStmt, err = db.Prepare(conversationSelect)
	if err != nil {
		log.Fatal(err)
	}
	participantSelectStmt, err = db.Prepare(participantSelect)
	if err != nil {
		log.Fatal(err)
	}
	messageInsertStmt, err = db.Prepare(messageInsert)
	if err != nil {
		log.Fatal(err)
	}
	messageSelectStmt, err = db.Prepare(messageSelect)
	if err != nil {
		log.Fatal(err)
	}
	tokenInsertStmt, err = db.Prepare(tokenInsert)
	if err != nil {
		log.Fatal(err)
	}
	tokenSelectStmt, err = db.Prepare(tokenSelect)
	if err != nil {
		log.Fatal(err)
	}
	conversationUpdateStmt, err = db.Prepare(conversationUpdate)
	if err != nil {
		log.Fatal(err)
	}
	commentInsertStmt, err = db.Prepare(commentInsert)
	if err != nil {
		log.Fatal(err)
	}
	commentSelectStmt, err = db.Prepare(commentSelect)
	if err != nil {
		log.Fatal(err)
	}
	lastMessageSelectStmt, err = db.Prepare(lastMessageSelect)
	if err != nil {
		log.Fatal(err)
	}
	commentCountSelectStmt, err = db.Prepare(commentCountSelect)
	if err != nil {
		log.Fatal(err)
	}
	profileSelectStmt, err = db.Prepare(profileSelect)
	if err != nil {
		log.Fatal(err)
	}
	go keepalive(db)
	pool = redis.NewPool(RedisDial, 100)
	http.HandleFunc(UrlBase+"/login", loginHandler)
	http.HandleFunc(UrlBase+"/register", registerHandler)
	http.HandleFunc(UrlBase+"/newconversation", newConversationHandler)
	http.HandleFunc(UrlBase+"/newgroupconversation", newGroupConversationHandler)
	http.HandleFunc(UrlBase+"/conversations", conversationHandler)
	http.HandleFunc(UrlBase+"/conversations/", anotherConversationHandler)
	http.HandleFunc(UrlBase+"/posts", postHandler)
	http.HandleFunc(UrlBase+"/posts/", anotherPostHandler)
	http.HandleFunc(UrlBase+"/user/", userHandler)
	http.HandleFunc(UrlBase+"/longpoll", longPollHandler)
	http.ListenAndServe(":8080", nil)
}

/********************************************************************
Top-level functions
********************************************************************/

func createToken(userid uint64) Token {
	hash := sha256.New()
	random := make([]byte, 10)
	_, err := io.ReadFull(rand.Reader, random)
	if err == nil {
		hash.Write(random)
		digest := hex.EncodeToString(hash.Sum(nil))
		expiry := time.Now().Add(time.Duration(24) * time.Hour).UTC()
		token := Token{userid, digest, expiry}
		return (token)
	} else {
		return (Token{userid, "foo", time.Now().UTC()})
	}
}

func looksLikeEmail(email string) bool {
	rx := "<?\\S+@\\S+?>?"
	regex, _ := regexp.Compile(rx)
	if !regex.MatchString(email) {
		return (false)
	} else {
		return (true)
	}
}

func getLastMessage(id uint64) (message Message, err error) {
	message, err = redisGetLastMessage(id)
	if err != nil {
		message, err = dbGetLastMessage(id)
	}
	return
}

func validateToken(id uint64, token string) bool {
	if LoginOverride {
		return (true)
	} else if redisTokenExists(id, token) {
		return (true)
	} else {
		var expiry string
		err := tokenSelectStmt.QueryRow(id, token).Scan(&expiry)
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
}

func validatePass(user string, pass string) (id uint64, err error) {
	hash := make([]byte, 256)
	passBytes := []byte(pass)
	err = passStmt.QueryRow(user).Scan(&id, &hash)
	if err != nil {
		return 0, err
	} else {
		err := bcrypt.CompareHashAndPassword(hash, passBytes)
		if err != nil {
			return 0, err
		} else {
			return id, nil
		}
	}
}

func createAndStoreToken(id uint64) (Token, error) {
	token := createToken(id)
	_, err := tokenInsertStmt.Exec(token.UserId, token.Token, token.Expiry)
	redisPutToken(token)
	if err != nil {
		return token, err
	} else {
		return token, nil
	}
}

func getUser(id uint64) (user User, err error) {
	/* Hits the cache then the db
	only I'm not 100% confident yet with what
	happens when you attempt to get a redis key
	that doesn't exist in redigo! */
	user, err = redisGetUser(id)
	if err != nil {
		user, err = dbGetUser(id)
		redisSetUser(user)
	}
	return user, err
}

/********************************************************************
Database functions
********************************************************************/

func updateConversation(id uint64) {
	_, err := conversationUpdateStmt.Exec(id)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func getCommentCount(id uint64) int {
	var count int
	err := commentCountSelectStmt.QueryRow(id).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

func dbGetLastMessage(id uint64) (message Message, err error) {
	var timeString string
	var by uint64
	err = lastMessageSelectStmt.QueryRow(id).Scan(&message.Id, &by, &message.Text, &timeString, &message.Seen)
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

func validateEmail(email string) bool {
	if !looksLikeEmail(email) {
		return (false)
	} else {
		rows, err := ruleStmt.Query()
		if err != nil {
			log.Fatalf("Error preparing statement: %v", err)
		}
		defer rows.Close()
		for rows.Next() {
			rule := new(Rule)
			if err = rows.Scan(&rule.NetworkID, &rule.Type, &rule.Value); err != nil {
				log.Fatalf("Error getting rule: %v", err)
			}
			if rule.Type == "email" && strings.HasSuffix(email, rule.Value) {
				return (true)
			}
		}
		return (false)
	}
}

func registerUser(user string, pass string, email string) (uint64, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		return 0, err
	} else {
		res, err := registerStmt.Exec(user, hash, email)
		if err != nil && strings.HasPrefix(err.Error(), "Error 1062") { //Note to self:There must be a better way?
			return 0, APIerror{"Username or email address already taken"}
		} else if err != nil {
			return 0, err
		} else {
			id, _ := res.LastInsertId()
			return uint64(id), nil
		}
	}
}

func getUserNetworks(id uint64) []Network {
	rows, err := networkStmt.Query(id)
	nets := make([]Network, 0, 5)
	if err != nil {
		log.Fatalf("Error querying db: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var network Network
		err = rows.Scan(&network.Id, &network.Name)
		if err != nil {
			log.Fatalf("Error scanning row: %v", err)
		} else {
			nets = append(nets, network)
		}
	}
	return (nets)
}

func getParticipants(conv int64) []User {
	rows, err := participantSelectStmt.Query(conv)
	if err != nil {
		log.Fatalf("Error getting participant: %v", err)
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

func getMessages(convId uint64, offset int64) []Message {
	rows, err := messageSelectStmt.Query(convId, offset)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer rows.Close()
	messages := make([]Message, 0, 20)
	for rows.Next() {
		var message Message
		var timeString string
		var by uint64
		err := rows.Scan(&message.Id, &by, &message.Text, &timeString, &message.Seen)
		if err != nil {
			log.Fatalf("%v", err)
		}
		message.Time, err = time.Parse(MysqlTime, timeString)
		if err != nil {
			log.Fatalf("%v", err)
		}
		message.By, err = getUser(by)
		if err != nil {
			//should only happen if a message is from a non-existent user
			//(or the db is fucked :))
			log.Println(err)
		}
		messages = append(messages, message)
	}
	return (messages)
}

func getConversations(user_id uint64, start int64) ([]Conversation, error) {
	conversations := make([]Conversation, 0, 20)
	rows, err := conversationSelectStmt.Query(user_id, start)
	if err != nil {
		return conversations, err
	}
	defer rows.Close()
	for rows.Next() {
		var conv Conversation
		err = rows.Scan(&conv.Id)
		if err != nil {
			return conversations, err
		}
		conv.Participants = getParticipants(conv.Id)
		LastMessage, err := getLastMessage(uint64(conv.Id))
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		conversations = append(conversations, conv)
	}
	return conversations, nil
}

func getComments(id uint64, offset int64) ([]Comment, error) {
	rows, err := commentSelectStmt.Query(id, offset)
	comments := make([]Comment, 0, 20)
	if err != nil {
		return comments, err
	}
	defer rows.Close()
	for rows.Next() {
		var comment Comment
		comment.Post = id
		var timeString string
		var by uint64
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

func createConversation(id uint64, nParticipants int) Conversation {
	r, _ := conversationStmt.Exec(id)
	conversation := Conversation{}
	conversation.Id, _ = r.LastInsertId()
	participants := make([]User, 0, 10)
	user, err := getUser(id)
	if err != nil {
		log.Printf("error getting user: %d %v", id, err)
	}
	participants = append(participants, user)
	nParticipants--

	rows, err := randomStmt.Query()
	if err != nil {
		log.Fatalf("Error preparing statement: %v", err)
	}
	defer rows.Close()
	for nParticipants > 0 {
		rows.Next()
		if err = rows.Scan(&user.Id, &user.Name); err != nil {
			log.Fatalf("Error getting user: %v", err)
		} else {
			participants = append(participants, user)
			nParticipants--
		}
	}
	for _, u := range participants {
		_, err := participantStmt.Exec(conversation.Id, u.Id)
		if err != nil {
			log.Fatalf("Error adding user to conversation: %v", err)
		}
	}
	conversation.Participants = participants
	return (conversation)
}

func dbGetUser(id uint64) (user User, err error) {
	err = userStmt.QueryRow(id).Scan(&user.Id, &user.Name)
	if err != nil {
		return user, err
	} else {
		return user, nil
	}
}

func getPosts(net_id uint64) ([]PostSmall, error) {
	rows, err := wallSelectStmt.Query(net_id)
	posts := make([]PostSmall, 0, 20)
	if err != nil {
		return posts, err
	}
	defer rows.Close()
	for rows.Next() {
		var post PostSmall
		var t string
		var by uint64
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
		posts = append(posts, post)
	}
	return posts, nil
}

func getProfile(id uint64) (user Profile, err error) {
	err = profileSelectStmt.QueryRow(id).Scan(&user.User.Name, &user.Desc)
	user.User.Id = id
	return user, err
}


/********************************************************************
redis functions
********************************************************************/

func redisPublish(recipients []User, msg RedisMessage) {
	conn := pool.Get()
	defer conn.Close()
	JSONmsg, _ := json.Marshal(msg)
	for _, user := range recipients {
		conn.Send("PUBLISH", user.Id, JSONmsg)
	}
	conn.Flush()
}

func RedisDial() (redis.Conn, error) {
	conn, err := redis.Dial(RedisProto, RedisAddress)
	return conn, err
}

func redisGetLastMessage(id uint64) (message Message, err error) {
	conn := pool.Get()
	defer conn.Close()
	BaseKey := "conversations:" + strconv.FormatUint(id, 10) + ":lastmessage:"
	reply, err := redis.Values(conn.Do("MGET", BaseKey+"id", BaseKey+"by", BaseKey+"text", BaseKey+"time", BaseKey+"seen"))
	if err != nil {
		//should reach this if there is no last message
		log.Printf("error getting message in redis %v", err)
		return message, err
	}
	var postId int64
	var by int64
	var timeString string
	if _, err = redis.Scan(reply, &postId, &by, &message.Text, &timeString, &message.Seen); err != nil {
		return message, err
	}
	message.Id = uint64(postId)
	if by != 0 {
		message.By, err = getUser(uint64(by))
		if err != nil {
			log.Printf("error getting user %d %v", by, err)
		}
	}
	message.Time, err = time.Parse(time.RFC3339, timeString)
	return message, err
}

func redisSetLastMessage(convId uint64, message Message) {
	conn := pool.Get()
	defer conn.Close()
	BaseKey := "conversations:" + strconv.FormatUint(convId, 10) + ":lastmessage:"
	conn.Send("SET", BaseKey+"id", strconv.FormatUint(message.Id, 10))
	conn.Send("SET", BaseKey+"by", strconv.FormatUint(message.By.Id, 10))
	conn.Send("SET", BaseKey+"text", message.Text)
	conn.Send("SET", BaseKey+"time", message.Time.Format(time.RFC3339))
	conn.Send("SET", BaseKey+"seen", strconv.FormatBool(message.Seen))
	conn.Flush()
}

func redisSetUser(user User) {
	conn := pool.Get()
	defer conn.Close()
	BaseKey := "users:" + strconv.FormatUint(user.Id, 10) + ":"
	conn.Send("SET", BaseKey+"name", user.Name)
	conn.Flush()
}

func redisPutToken(token Token) {
	/* Set a session token in redis.
		We use the token value as part of the redis key
	        so that a user may have more than one concurrent session
		(eg: signed in on the web and mobile at once */
	conn := pool.Get()
	defer conn.Close()
	expiry := int(token.Expiry.Sub(time.Now()).Seconds())
	key := "users:" + strconv.FormatUint(token.UserId, 10) + ":token:" + token.Token
	conn.Send("SETEX", key, expiry, token.Expiry)
	conn.Flush()
}

func redisTokenExists(id uint64, token string) bool {
	conn := pool.Get()
	defer conn.Close()
	key := "users:" + strconv.FormatUint(id, 10) + ":token:" + token
	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false
	}
	return exists
}

func redisGetUser(id uint64) (user User, err error) {
	conn := pool.Get()
	defer conn.Close()
	user.Name, err = redis.String(conn.Do("GET", "users:"+strconv.FormatUint(id, 10)+":name"))
	if err != nil {
		return user, err
	}
	user.Id = id
	return user, nil
}

/*********************************************************************************

Begin http handlers!

*********************************************************************************/


func registerHandler(w http.ResponseWriter, r *http.Request) {
	/* POST /register
	requires parameters: user, pass, email 
        example responses:
        HTTP 201
	{"id":2397}
	HTTP 400
	{"error":"Invalid email"}
	*/

	//Note to self: maybe check cache for user before trying to register
	user := r.FormValue("user")
	pass := r.FormValue("pass")
	email := r.FormValue("email")
	switch {
	case r.Method != "POST":
		errorJSON, _ := json.Marshal(APIerror{"Must be a POST request!"})
		jsonResp(w, errorJSON, 405)
	case len(user) == 0:
		//Note to future self : would be neater if
		//we returned _all_ errors not just the first
		errorJSON, _ := json.Marshal(APIerror{"Missing parameter: user"})
		jsonResp(w, errorJSON, 400)
	case len(pass) == 0:
		errorJSON, _ := json.Marshal(APIerror{"Missing parameter: pass"})
		jsonResp(w, errorJSON, 400)
	case len(email) == 0:
		errorJSON, _ := json.Marshal(APIerror{"Missing parameter: email"})
		jsonResp(w, errorJSON, 400)
	case !validateEmail(email):
		errorJSON, _ := json.Marshal(APIerror{"Invalid Email"})
		jsonResp(w, errorJSON, 400)
	default:
		id, err := registerUser(user, pass, email)
		if err != nil {
			_, ok := err.(APIerror)
			if ok { //Duplicate user/email
				errorJSON, _ := json.Marshal(err)
				jsonResp(w, errorJSON, 400)
			} else {
				errorJSON, _ := json.Marshal(APIerror{err.Error()})
				jsonResp(w, errorJSON, 500)
			}
		} else {
			resp := []byte("{\"id\":" + strconv.FormatUint(id, 10) + "}")
			jsonResp(w, resp, 201)
		}
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	/* POST /login
	requires parameters: user, pass
	example responses:
	HTTP 200  
        {
            "id":2397,
            "value":"552e5a9687ec04418b3b4da61a8b062dbaf5c7937f068341f36a4b4fcbd4ed45",
            "expiry":"2013-09-25T14:43:17.664646892Z"
        }
	HTTP 400  
	{"error":"Bad username/password"}
	*/
	user := r.FormValue("user")
	pass := r.FormValue("pass")
	id, err := validatePass(user, pass)
	switch {
	case r.Method != "POST":
		errorJSON, _ := json.Marshal(APIerror{"Must be a POST request!"})
		jsonResp(w, errorJSON, 405)
	case err == nil:
		token, err := createAndStoreToken(id)
		if err == nil {
			tokenJSON, _ := json.Marshal(token)
			w.Write(tokenJSON)
		} else {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		}
	default:
		errorJSON, _ := json.Marshal(APIerror{"Bad username/password"})
		jsonResp(w, errorJSON, 400)
	}
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	/* GET /posts
	   requires parameters: id, token
	   
           POST /posts
	   requires parameters: id, token

        */
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	token := r.FormValue("token")
	switch {
	case !validateToken(id, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case r.Method == "GET":
		networks := getUserNetworks(id)
		posts, err := getPosts(networks[0].Id)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		}
		postsJSON, err := json.Marshal(posts)
		if err != nil {
			log.Printf("Something went wrong with json parsing: %v", err)
		}
		w.Write(postsJSON)
	case r.Method == "POST":
		id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
		text := r.FormValue("text")
		networks := getUserNetworks(id)
		res, err := postStmt.Exec(id, text, networks[0].Id)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			postId, _ := res.LastInsertId()
			w.Write([]byte("{\"id\":" + strconv.FormatInt(postId, 10) + "}"))
		}
	default:
		errorJSON, _ := json.Marshal(APIerror{"Must be a POST or GET request"})
		jsonResp(w, errorJSON, 405)
	}
}

func newConversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	token := r.FormValue("token")
	switch {
	case r.Method != "POST":
		errorJSON, _ := json.Marshal(APIerror{"Must be a POST request"})
		jsonResp(w, errorJSON, 405)
	case !validateToken(id, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	default:
		conversation := createConversation(id, 2)
		conversationJSON, _ := json.Marshal(conversation)
		w.Write(conversationJSON)
	}
}

func newGroupConversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	token := r.FormValue("token")
	switch {
	case r.Method != "POST":
		errorJSON, _ := json.Marshal(APIerror{"Must be a POST request"})
		jsonResp(w, errorJSON, 405)
	case !validateToken(id, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	default:
		conversation := createConversation(id, 4)
		conversationJSON, _ := json.Marshal(conversation)
		w.Write(conversationJSON)
	}
}

func conversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	token := r.FormValue("token")
	switch {
	case r.Method != "GET":
		errorJSON, _ := json.Marshal(APIerror{"Must be a GET request"})
		jsonResp(w, errorJSON, 405)
	case !validateToken(id, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	default:
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			start = 0
		}
		conversations, err := getConversations(id, start)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			conversationsJSON, _ := json.Marshal(conversations)
			w.Write(conversationsJSON)
		}
	}
}

func anotherConversationHandler(w http.ResponseWriter, r *http.Request) { //lol
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	token := r.FormValue("token")
	regex, _ := regexp.Compile("conversations/(\\d+)/messages/?$")
	convIdString := regex.FindStringSubmatch(r.URL.Path)
	regex2, _ := regexp.Compile("conversations/(\\d+)/?$")
	convIdString2 := regex2.FindStringSubmatch(r.URL.Path)
	switch {
	case !validateToken(id, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case convIdString != nil && r.Method == "GET":
		convId, _ := strconv.ParseUint(convIdString[1], 10, 16)
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			start = 0
		}
		messages := getMessages(convId, start)
		messagesJSON, _ := json.Marshal(messages)
		w.Write(messagesJSON)
	case convIdString != nil && r.Method == "POST":
		convId, _ := strconv.ParseUint(convIdString[1], 10, 16)
		text := r.FormValue("text")
		res, err := messageInsertStmt.Exec(convId, id, text)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		}
		messageId, _ := res.LastInsertId()
		participants := getParticipants(int64(convId))
		user, err := getUser(id)
		if err != nil {
			//Should only happen if the conversation has non-existent
			//participants. Or the db has just died.
			log.Println(err)
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		}
		msgSmall := Message{uint64(messageId), user, text, time.Now().UTC(), false}
		redisSetLastMessage(convId, msgSmall)
		msg := RedisMessage{msgSmall, convId}
		go redisPublish(participants, msg)
		go updateConversation(convId)
		w.Write([]byte("{\"id\":" + strconv.FormatInt(messageId, 10) + "}"))
	case convIdString != nil: //Unsuported method
		errorJSON, _ := json.Marshal(APIerror{"Must be a GET or POST request"})
		jsonResp(w, errorJSON, 405)
	case convIdString2 != nil && r.Method != "GET":
		errorJSON, _ := json.Marshal(APIerror{"Must be a GET request"})
		jsonResp(w, errorJSON, 405)
	case convIdString2 != nil:
		convId, _ := strconv.ParseInt(convIdString2[1], 10, 16)
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			start = 0
		}
		var conv ConversationAndMessages
		conv.Id = convId
		conv.Participants = getParticipants(conv.Id)
		conv.Messages = getMessages(uint64(convId), start)
		conversationJSON, _ := json.Marshal(conv)
		w.Write(conversationJSON)
	default:
		errorJSON, _ := json.Marshal(APIerror{"404 not found"})
		jsonResp(w, errorJSON, 404)
	}
}

func anotherPostHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	token := r.FormValue("token")
	regexComments, _ := regexp.Compile("posts/(\\d+)/comments/?$")
	regexNoComments, _ := regexp.Compile("posts/(\\d+)/?$")
	commIdStringA := regexComments.FindStringSubmatch(r.URL.Path)
	commIdStringB := regexNoComments.FindStringSubmatch(r.URL.Path)
	switch {
	case !validateToken(id, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case commIdStringA != nil && r.Method == "GET":
		commId, _ := strconv.ParseUint(commIdStringA[1], 10, 16)
		offset, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			offset = 0
		}
		comments, err := getComments(commId, offset)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			jsonComments, _ := json.Marshal(comments)
			w.Write(jsonComments)
		}
	case commIdStringA != nil && r.Method == "POST":
		commId, _ := strconv.ParseUint(commIdStringA[1], 10, 16)
		text := r.FormValue("text")
		res, err := commentInsertStmt.Exec(commId, id, text)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			commentId, _ := res.LastInsertId()
			w.Write([]byte("{\"id\":" + strconv.FormatInt(commentId, 10) + "}"))
		}
	case commIdStringB != nil && r.Method == "GET":
		commId, _ := strconv.ParseUint(commIdStringB[1], 10, 16)
		log.Printf("%d", commId)
		//implement getting a specific post
	default:
		errorJSON, _ := json.Marshal(APIerror{"Method not supported"})
		jsonResp(w, errorJSON, 405)
	}
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	token := r.FormValue("token")
	regexUser, _ := regexp.Compile("user/(\\d+)/?$")
	userIdString := regexUser.FindStringSubmatch(r.URL.Path)
	switch {
	case !validateToken(id, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case r.Method != "GET":
		errorJSON, _ := json.Marshal(APIerror{"Method not supported"})
		jsonResp(w, errorJSON, 405)
	case userIdString != nil:
		userId, _ := strconv.ParseUint(userIdString[1], 10, 16)
		user, err := getProfile(userId)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		}
		userjson, _ := json.Marshal(user)
		w.Write(userjson)
	default:
		errorJSON, _ := json.Marshal(APIerror{"User not found"})
		jsonResp(w, errorJSON, 404)
	}
}

func longPollHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	token := r.FormValue("token")
	if !validateToken(id, token) {
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	} else {
		conn := pool.Get()
		defer conn.Close()
		psc := redis.PubSubConn{conn}
		psc.Subscribe(id)
		for {
			switch n := psc.Receive().(type) {
			case redis.Message:
				w.Write([]byte(n.Data))
				return
			case redis.Subscription:
				fmt.Printf("%s: %s %d\n", n.Channel, n.Kind, n.Count)
			}
		}
	}
}
