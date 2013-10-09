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
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type UserId uint64
type NetworkId uint64
type MessageId uint64
type PostId uint64
type CommentId uint64
type ConversationId uint64

type User struct {
	Id   UserId `json:"id"`
	Name string `json:"username"`
}

type Profile struct {
	User
	Desc    string  `json:"tagline"`
	Avatar  string  `json:"profile_image"`
	Network Network `json:"network"`
	Course  string  `json:"course"`
}

type Network struct {
	Id   NetworkId `json:"id"`
	Name string    `json:"name"`
}

type Message struct {
	Id   MessageId `json:"id"`
	By   User      `json:"by"`
	Text string    `json:"text"`
	Time time.Time `json:"timestamp"`
	Seen bool      `json:"seen"`
}

type RedisMessage struct {
	Message
	Conversation ConversationId `json:"conversation_id"`
}

type Token struct {
	UserId UserId    `json:"id"`
	Token  string    `json:"value"`
	Expiry time.Time `json:"expiry"`
}

type Post struct {
	Id     PostId    `json:"id"`
	By     User      `json:"by"`
	Time   time.Time `json:"timestamp"`
	Text   string    `json:"text"`
	Images []string  `json:"images"`
}

type PostSmall struct {
	Post
	CommentCount int `json:"comments"`
	LikeCount    int `json:"likes"`
	HateCount    int `json:"hates"`
}

type PostFull struct {
	Post
	Comments  []Comment  `json:"comments"`
	LikeHates []LikeHate `json:"like_hate"`
}

type Comment struct {
	Id   CommentId `json:"id"`
	Post PostId    `json:"-"`
	By   User      `json:"by"`
	Time time.Time `json:"timestamp"`
	Text string    `json:"text"`
}

type LikeHate struct {
	Value  bool // true is like, false is hate
	UserID UserId
}

type Rule struct {
	NetworkID NetworkId
	Type      string
	Value     string
}

type Conversation struct {
	Id           ConversationId `json:"id"`
	Participants []User         `json:"participants"`
}

type ConversationSmall struct {
	Conversation
	LastActivity time.Time `json:"-"`
	LastMessage  *Message  `json:"mostRecentMessage"`
}

type ConversationAndMessages struct {
	Conversation
	Messages []Message `json:"messages"`
}

type Config struct {
	UrlBase                 string
	Port                    string
	LoginOverride           bool
	RedisProto              string
	RedisAddress            string
	MysqlMaxConnectionCount int
	MysqlUser               string
	MysqlPass               string
	MysqlHost               string
	MysqlPort               string
}

func (c *Config) ConnectionString() string {
	return c.MysqlUser + ":" + c.MysqlPass + "@tcp(" + c.MysqlHost + ":" + c.MysqlPort + ")/gleepost?charset=utf8"
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
	MysqlTime = "2006-01-02 15:04:05"
)

var (
	pool       *redis.Pool
	config     *Config
	configLock = new(sync.RWMutex)
)

func loadConfig(fail bool) {
	file, err := ioutil.ReadFile("conf.json")
	if err != nil {
		log.Println("Opening config failed: ", err)
		if fail {
			os.Exit(1)
		}
	}

	c := new(Config)
	if err = json.Unmarshal(file, c); err != nil {
		log.Println("Parsing config failed: ", err)
		if fail {
			os.Exit(1)
		}
	}
	configLock.Lock()
	config = c
	configLock.Unlock()
}

func GetConfig() *Config {
	configLock.RLock()
	defer configLock.RUnlock()
	return config
}

func configInit() {
	loadConfig(true)
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGUSR2)
	go func() {
		for {
			<-s
			loadConfig(false)
			log.Println("Reloaded")
		}
	}()
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	configInit()
	conf := GetConfig()
	db, err := sql.Open("mysql", conf.ConnectionString())
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	db.SetMaxIdleConns(conf.MysqlMaxConnectionCount)
	prepare(db)
	go keepalive(db)
	pool = redis.NewPool(RedisDial, 100)
	http.HandleFunc(conf.UrlBase+"/login", loginHandler)
	http.HandleFunc(conf.UrlBase+"/register", registerHandler)
	http.HandleFunc(conf.UrlBase+"/newconversation", newConversationHandler)
	http.HandleFunc(conf.UrlBase+"/newgroupconversation", newGroupConversationHandler)
	http.HandleFunc(conf.UrlBase+"/conversations", conversationHandler)
	http.HandleFunc(conf.UrlBase+"/conversations/", anotherConversationHandler)
	http.HandleFunc(conf.UrlBase+"/posts", postHandler)
	http.HandleFunc(conf.UrlBase+"/posts/", anotherPostHandler)
	http.HandleFunc(conf.UrlBase+"/user/", userHandler)
	http.HandleFunc(conf.UrlBase+"/longpoll", longPollHandler)
	http.ListenAndServe(":"+conf.Port, nil)
}

/********************************************************************
Top-level functions
********************************************************************/

func createToken(userId UserId) Token {
	hash := sha256.New()
	random := make([]byte, 10)
	_, err := io.ReadFull(rand.Reader, random)
	if err == nil {
		hash.Write(random)
		digest := hex.EncodeToString(hash.Sum(nil))
		expiry := time.Now().Add(time.Duration(24) * time.Hour).UTC()
		token := Token{userId, digest, expiry}
		return (token)
	} else {
		return (Token{userId, "foo", time.Now().UTC()})
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

func getLastMessage(id ConversationId) (message Message, err error) {
	message, err = redisGetLastMessage(id)
	if err != nil {
		// Last message is not in redis
		count, err := redisGetConversationMessageCount(id)
		if err != nil {
			//and the number of messages that exist is not in redis
			message, err = dbGetLastMessage(id)
			if err != nil {
				//this won't wipe the cache since if we're here it's already missing
				redisSetConversationMessageCount(id, 0)
			}
		} else {
			//and the number of messages we should have is in redis
			if count != 0 { // this number currently is probably completely wrong!
				// but it should be correct in zero vs non-zero terms
				message, err = dbGetLastMessage(id)
				redisSetLastMessage(id, message)
			}
		}
	}
	return
}

func validateToken(id UserId, token string) bool {
	conf := GetConfig()
	if conf.LoginOverride {
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

func validatePass(user string, pass string) (id UserId, err error) {
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

func createAndStoreToken(id UserId) (Token, error) {
	token := createToken(id)
	_, err := tokenInsertStmt.Exec(token.UserId, token.Token, token.Expiry)
	redisPutToken(token)
	if err != nil {
		return token, err
	} else {
		return token, nil
	}
}

func getUser(id UserId) (user User, err error) {
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

func getCommentCount(id PostId) (count int) {
	count, err := redisGetCommentCount(id)
	if err != nil {
		count = dbGetCommentCount(id)
		go redisSetCommentCount(id, count)
	}
	return count
}

func createComment(postId PostId, userId UserId, text string) (commId CommentId, err error) {
	commId, err = dbCreateComment(postId, userId, text)
	if err == nil {
		err = redisIncCommentCount(postId)
	}
	return commId, err
}

func getUserNetworks(id UserId) (nets []Network) {
	nets, err := redisGetUserNetwork(id)
	if err != nil {
		nets = dbGetUserNetworks(id)
		redisSetUserNetwork(id, nets[0])
	}
	return (nets)
}

func getParticipants(convId ConversationId) []User {
	participants, err := redisGetConversationParticipants(convId)
	if err != nil {
		participants = dbGetParticipants(convId)
		go redisSetConversationParticipants(convId, participants)
	}
	return participants
}

func getMessages(convId ConversationId, offset int64) []Message {
	messages, err := redisGetMessages(convId, offset)
	if err != nil {
		messages = dbGetMessages(convId, offset)
		go redisAddMessages(convId, messages)
	}
	return messages
}

func getConversations(user_id UserId, start int64) (conversations []ConversationSmall, err error) {
	conversations, err = redisGetConversations(user_id, start)
	if err != nil {
		conversations, err = dbGetConversations(user_id, start)
		if err == nil {
			for _, conv := range conversations {
				go redisAddConversation(conv)
			}
		}
		return
	}
	return
}

func getMessage(msgId MessageId) (message Message, err error) {
	message, err = redisGetMessage(msgId)
	return message, err
}

func updateConversation(id ConversationId) (err error) {
	err = dbUpdateConversation(id)
	if err != nil {
		return err
	}
	go redisUpdateConversation(id)
	return nil
}

func addMessage(convId ConversationId, userId UserId, text string) (messageId MessageId, err error) {
	messageId, err = dbAddMessage(convId, userId, text)
	if err != nil {
		return
	}
	user, err := getUser(userId)
	if err != nil {
		return
	}
	msgSmall := Message{MessageId(messageId), user, text, time.Now().UTC(), false}
	go redisSetLastMessage(convId, msgSmall)
	msg := RedisMessage{msgSmall, convId}
	go redisPublish(msg)
	go redisIncConversationMessageCount(convId)
	go updateConversation(convId)
	return
}

func getFullConversation(convId ConversationId, start int64) (conv ConversationAndMessages) {
	conv.Conversation.Id = convId
	conv.Conversation.Participants = getParticipants(convId)
	conv.Messages = getMessages(convId, start)
	return
}

func getPostImages(postId PostId) (images []string) {
	images, _ = dbGetPostImages(postId)
	return
}

func getProfile(id UserId) (user Profile, err error) {
	user, err = dbGetProfile(id)
	return
}

func awaitOneMessage(userId UserId) []byte {
	conn := pool.Get()
	defer conn.Close()
	psc := redis.PubSubConn{Conn:conn}
	psc.Subscribe(userId)
	for {
		switch n := psc.Receive().(type) {
		case redis.Message:
			return n.Data
		case redis.Subscription:
			fmt.Printf("%s: %s %d\n", n.Channel, n.Kind, n.Count)
		}
	}
}

func addPost(userId UserId, text string) (postId PostId, err error) {
	return dbAddPost(userId, text)
}

func getPosts(netId NetworkId, start int64) (posts []PostSmall, err error) {
	posts, err = redisGetNetworkPosts(netId, start)
	if err != nil {
		posts, err = dbGetPosts(netId)
	}
	return
}

/********************************************************************
Database functions
********************************************************************/

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

func validateEmail(email string) bool {
	if !looksLikeEmail(email) {
		return (false)
	} else {
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
}

func registerUser(user string, pass string, email string) (UserId, error) {
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
			return UserId(id), nil
		}
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

func dbGetMessages(convId ConversationId, offset int64) []Message {
	rows, err := messageSelectStmt.Query(convId, offset)
	log.Println("DB hit: getMessages convid, offset (message.id, message.by, message.text, message.time, message.seen)")
	if err != nil {
		log.Printf("%v", err)
	}
	defer rows.Close()
	messages := make([]Message, 0, 20)
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
	return (messages)
}

func dbGetConversations(user_id UserId, start int64) (conversations []ConversationSmall, err error) {
	rows, err := conversationSelectStmt.Query(user_id, start)
	log.Println("DB hit: getConversations user_id, offset (conversation.id)")
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

func getComments(id PostId, offset int64) ([]Comment, error) {
	rows, err := commentSelectStmt.Query(id, offset)
	log.Println("DB hit: getComments postid, offset(comment.id, comment.by, comment.text, comment.time)")
	comments := make([]Comment, 0, 20)
	if err != nil {
		return comments, err
	}
	defer rows.Close()
	for rows.Next() {
		var comment Comment
		comment.Post = id
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

func createConversation(id UserId, nParticipants int) Conversation {
	r, _ := conversationStmt.Exec(id)
	conversation := Conversation{}
	cId, _ := r.LastInsertId()
	conversation.Id = ConversationId(cId)
	participants := make([]User, 0, 10)
	user, err := getUser(id)
	if err != nil {
		log.Printf("error getting user: %d %v", id, err)
	}
	participants = append(participants, user)
	nParticipants--

	rows, err := randomStmt.Query()
	log.Println("DB hit: createConversation (user.Name, user.Id)")
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
	}
	defer rows.Close()
	for nParticipants > 0 {
		rows.Next()
		if err = rows.Scan(&user.Id, &user.Name); err != nil {
			log.Printf("Error getting user: %v", err)
		} else {
			participants = append(participants, user)
			nParticipants--
		}
	}
	for _, u := range participants {
		_, err := participantStmt.Exec(conversation.Id, u.Id)
		if err != nil {
			log.Printf("Error adding user to conversation: %v", err)
		}
	}
	conversation.Participants = participants
	return (conversation)
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

func dbGetPosts(net_id NetworkId) (posts []PostSmall, err error) {
	rows, err := wallSelectStmt.Query(net_id)
	defer rows.Close()
	log.Println("DB hit: getPosts net_id(post.id, post.by, post.time, post.texts)")
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

/********************************************************************
redis functions
********************************************************************/

func redisAddPost(post PostSmall) {
	conn := pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("posts:%d", post.Id)
	conn.Send("MSET", baseKey + ":by", post.By, baseKey + ":time", post.Time.Format(time.RFC3339), baseKey + ":text", post.Text)
	conn.Flush()
}

func redisGetPost(postId PostId) (post PostSmall, err error) {
	conn := pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("posts:%d", postId)
	values, err := redis.Values(conn.Do("MGET", baseKey + ":by", baseKey + ":time", baseKey + ":text"))
	if err != nil {
		return post, err
	}
	var by UserId
	var t string
	if _, err = redis.Scan(values, &by, &t, &post.Post.Text); err != nil {
		return post, err
	}
	post.Post.Id = postId
	post.Post.By, err = getUser(by)
	if err != nil {
		return post, err
	}
	post.Post.Time, _ = time.Parse(time.RFC3339, t)
	post.Post.Images = getPostImages(postId)
	post.CommentCount = getCommentCount(postId)
	return post, nil
}

func redisGetNetworkPosts(id NetworkId, start int64) (posts []PostSmall, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("networks:%d:posts", id)
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, start+19))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return posts, redis.Error("No posts for this network in redis.")
	}
	for len(values) > 0 {
		curr := -1
		values, err = redis.Scan(values, &curr)
		if err != nil {
			return
		}
		if curr == -1 {
			return
		}
		postId := PostId(curr)
		post, err := redisGetPost(postId)
		if err != nil {
			return posts, err
		}
		posts = append(posts, post)
	}
	return
}

func redisUpdateConversation(id ConversationId) {
	conn := pool.Get()
	defer conn.Close()
	participants := getParticipants(id)
	for _, user := range participants {
		key := "users:" + strconv.FormatUint(uint64(user.Id), 10) + ":conversations"
		//nb: this means that the last activity time for a conversation will
		//differ slightly from the db to the cache (and even from user to user)
		//but I think this is okay because it's only for ordering purposes
		//(the actual last message timestamp will be consistent)
		conn.Send("ZADD", key, time.Now().Unix(), id)
	}
	conn.Flush()
}

func redisPublish(msg RedisMessage) {
	conn := pool.Get()
	defer conn.Close()
	participants := getParticipants(msg.Conversation)
	JSONmsg, _ := json.Marshal(msg)
	for _, user := range participants {
		conn.Send("PUBLISH", user.Id, JSONmsg)
	}
	conn.Flush()
}

func RedisDial() (redis.Conn, error) {
	conf := GetConfig()
	conn, err := redis.Dial(conf.RedisProto, conf.RedisAddress)
	return conn, err
}

func redisGetCommentCount(id PostId) (count int, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := "posts:" + strconv.FormatUint(uint64(id), 10) + ":comment_count"
	count, err = redis.Int(conn.Do("GET", key))
	if err != nil {
		return 0, err
	} else {
		return count, nil
	}
}

func redisSetCommentCount(id PostId, count int) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("posts:%d:comment_count", id)
	conn.Send("SET", key, count)
	conn.Flush()
}

func redisGetUserNetwork(userId UserId) (networks []Network, err error) {
	/* Part 1 of the transition to one network per user (why did I ever allow more :| */
	//this returns a slice of 1 network to keep compatible with dbGetNetworks
	conn := pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("users:%d:network", userId)
	reply, err := redis.Values(conn.Do("MGET", baseKey+":id", baseKey+":name"))
	if err != nil {
		return networks, err
	}
	net := Network{}
	if _, err = redis.Scan(reply, &net.Id, &net.Name); err != nil {
		return networks, err
	} else if net.Id == 0 {
		//there must be a neater way?
		err = redis.Error("Cache miss")
		return networks, err
	}
	networks = append(networks, net)
	return networks, nil
}

func redisSetUserNetwork(userId UserId, network Network) {
	conn := pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("users:%d:network", userId)
	conn.Send("MSET", baseKey+":id", network.Id, baseKey+":name", network.Name)
	conn.Flush()
}

func redisIncCommentCount(id PostId) (err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("posts:%d:comment_count", id)
	conn.Send("INCR", key)
	conn.Flush()
	return nil
}

func redisGetLastMessage(id ConversationId) (message Message, err error) {
	conn := pool.Get()
	defer conn.Close()
	BaseKey := fmt.Sprintf("conversations:%d:lastmessage:", id)
	reply, err := redis.Values(conn.Do("MGET", BaseKey+"id", BaseKey+"by", BaseKey+"text", BaseKey+"time", BaseKey+"seen"))
	if err != nil {
		//should reach this if there is no last message
		log.Printf("error getting message in redis %v", err)
		return message, err
	}
	var by UserId
	var timeString string
	if _, err = redis.Scan(reply, &message.Id, &by, &message.Text, &timeString, &message.Seen); err != nil {
		return message, err
	}
	if by != 0 {
		message.By, err = getUser(by)
		if err != nil {
			log.Printf("error getting user %d %v", by, err)
		}
	}
	message.Time, err = time.Parse(time.RFC3339, timeString)
	return message, err
}

func redisSetLastMessage(convId ConversationId, message Message) {
	conn := pool.Get()
	defer conn.Close()
	BaseKey := fmt.Sprintf("conversations:%d:lastmessage:", convId)
	conn.Send("SET", BaseKey+"id", message.Id)
	conn.Send("SET", BaseKey+"by", message.By.Id)
	conn.Send("SET", BaseKey+"text", message.Text)
	conn.Send("SET", BaseKey+"time", message.Time.Format(time.RFC3339))
	conn.Send("SET", BaseKey+"seen", strconv.FormatBool(message.Seen))
	conn.Flush()
}

func redisGetConversationMessageCount(convId ConversationId) (count int, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messagecount", convId)
	count, err = redis.Int(conn.Do("GET", key))
	if err != nil {
		return 0, err
	}
	return count, nil
}

func redisSetConversationMessageCount(convId ConversationId, count int) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messagecount", convId)
	conn.Send("SET", key, count)
	conn.Flush()
}

func redisIncConversationMessageCount(convId ConversationId) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messagecount", convId)
	conn.Send("INCR", key)
	conn.Flush()
}

func redisSetConversationParticipants(convId ConversationId, participants []User) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:participants", convId)
	for _, user := range participants {
		conn.Send("HSET", key, user.Id, user.Name)
	}
	conn.Flush()
}

func redisGetConversationParticipants(convId ConversationId) (participants []User, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:participants", convId)
	values, err := redis.Values(conn.Do("HGETALL", key))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return participants, redis.Error("Nothing in redis")
	}
	for len(values) > 0 {
		user := User{}
		values, err = redis.Scan(values, &user.Id, &user.Name)
		if err != nil {
			return
		}
		participants = append(participants, user)
	}
	return
}

func redisSetUser(user User) {
	conn := pool.Get()
	defer conn.Close()
	BaseKey := fmt.Sprintf("users:%d", user.Id)
	conn.Send("SET", BaseKey+":name", user.Name)
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
	key := fmt.Sprintf("users:%d:token:%s", token.UserId, token.Token)
	conn.Send("SETEX", key, expiry, token.Expiry)
	conn.Flush()
}

func redisTokenExists(id UserId, token string) bool {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:token:%s", id, token)
	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false
	}
	return exists
}

func redisGetUser(id UserId) (user User, err error) {
	conn := pool.Get()
	defer conn.Close()
	user.Name, err = redis.String(conn.Do("GET", fmt.Sprintf("users:%d:name", id)))
	if err != nil {
		return user, err
	}
	user.Id = id
	return user, nil
}

func redisGetConversations(id UserId, start int64) (conversations []ConversationSmall, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:conversations", id)
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, start+19))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return conversations, redis.Error("No conversations for this user in redis.")
	}
	for len(values) > 0 {
		curr := -1
		values, err = redis.Scan(values, &curr)
		if err != nil {
			return
		}
		if curr == -1 {
			return
		}
		conv := ConversationSmall{}
		conv.Id = ConversationId(curr)
		conv.Conversation.Participants = getParticipants(conv.Id)
		LastMessage, err := getLastMessage(conv.Id)
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		conversations = append(conversations, conv)
	}
	return
}

func redisAddConversation(conv ConversationSmall) {
	conn := pool.Get()
	defer conn.Close()
	for _, participant := range conv.Participants {
		key := fmt.Sprintf("users:%d:conversations", participant.Id)
		conn.Send("ZADD", key, conv.LastActivity.Unix(), conv.Id)
	}
	conn.Flush()
}

func redisAddMessages(convId ConversationId, messages []Message) {
	//expecting messages ordered b
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convId)
	for _, message := range messages {
		conn.Send("ZADD", key, message.Time.Unix(), message.Id)
		go redisSetMessage(message)
	}
	conn.Flush()
}

func redisSetMessage(message Message) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("messages:%d", message.Id)
	conn.Send("MSET", key+":by", message.By.Id, key+":text", message.Text, key+":time", message.Time.Format(time.RFC3339), key+":seen", message.Seen)
	conn.Flush()
}

func redisGetMessages(convId ConversationId, start int64) (messages []Message, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convId)
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, start+19))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return messages, redis.Error("No messages for this conversation in redis.")
	}
	for len(values) > 0 {
		curr := -1
		values, err = redis.Scan(values, &curr)
		if err != nil {
			return
		}
		if curr == -1 {
			return
		}
		if curr != 0 {
			message, errGettingMessage := getMessage(MessageId(curr))
			if errGettingMessage != nil {
				return messages, errGettingMessage
			} else {
				go redisSetMessage(message)
			}
			messages = append(messages, message)
		}
	}
	return
}

func redisGetMessage(msgId MessageId) (message Message, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("messages:%d", msgId)
	reply, err := redis.Values(conn.Do("MGET", key+":by", key+":text", key+":timestamp", key+":seen"))
	if err != nil {
		return message, err
	}
	message.Id = msgId
	var timeString string
	var by UserId
	if _, err = redis.Scan(reply, &by, &message.Text, &timeString, &message.Seen); err != nil {
		return message, err
	}
	if by != 0 {
		message.By, err = getUser(by)
		if err != nil {
			log.Printf("error getting user %d %v", by, err)
		}
	}
	message.Time, err = time.Parse(time.RFC3339, timeString)
	return message, err
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
			resp := []byte(fmt.Sprintf("{\"id\":%d}", id))
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
			jsonResp(w, tokenJSON, 200)
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
	userId := UserId(id)
	token := r.FormValue("token")
	switch {
	case !validateToken(userId, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case r.Method == "GET":
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
		if err != nil {
			start = 0
		}
		networks := getUserNetworks(userId)
		posts, err := getPosts(networks[0].Id, start)
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
		text := r.FormValue("text")
		postId, err := addPost(userId, text)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			w.Write([]byte(fmt.Sprintf("{\"id\":%d}", postId)))
		}
	default:
		errorJSON, _ := json.Marshal(APIerror{"Must be a POST or GET request"})
		jsonResp(w, errorJSON, 405)
	}
}

func newConversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	userId := UserId(id)
	token := r.FormValue("token")
	switch {
	case r.Method != "POST":
		errorJSON, _ := json.Marshal(APIerror{"Must be a POST request"})
		jsonResp(w, errorJSON, 405)
	case !validateToken(userId, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	default:
		conversation := createConversation(userId, 2)
		conversationJSON, _ := json.Marshal(conversation)
		w.Write(conversationJSON)
	}
}

func newGroupConversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	token := r.FormValue("token")
	userId := UserId(id)
	switch {
	case r.Method != "POST":
		errorJSON, _ := json.Marshal(APIerror{"Must be a POST request"})
		jsonResp(w, errorJSON, 405)
	case !validateToken(userId, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	default:
		conversation := createConversation(userId, 4)
		conversationJSON, _ := json.Marshal(conversation)
		w.Write(conversationJSON)
	}
}

func conversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	token := r.FormValue("token")
	userId := UserId(id)
	switch {
	case r.Method != "GET":
		errorJSON, _ := json.Marshal(APIerror{"Must be a GET request"})
		jsonResp(w, errorJSON, 405)
	case !validateToken(userId, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	default:
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			start = 0
		}
		conversations, err := getConversations(userId, start)
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
	userId := UserId(id)
	token := r.FormValue("token")
	regex, _ := regexp.Compile("conversations/(\\d+)/messages/?$")
	convIdString := regex.FindStringSubmatch(r.URL.Path)
	regex2, _ := regexp.Compile("conversations/(\\d+)/?$")
	convIdString2 := regex2.FindStringSubmatch(r.URL.Path)
	switch {
	case !validateToken(userId, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case convIdString != nil && r.Method == "GET":
		_convId, _ := strconv.ParseUint(convIdString[1], 10, 16)
		convId := ConversationId(_convId)
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			start = 0
		}
		messages := getMessages(convId, start)
		messagesJSON, _ := json.Marshal(messages)
		w.Write(messagesJSON)
	case convIdString != nil && r.Method == "POST":
		_convId, _ := strconv.ParseUint(convIdString[1], 10, 16)
		convId := ConversationId(_convId)
		text := r.FormValue("text")
		messageId, err := addMessage(convId, userId, text)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		}
		w.Write([]byte(fmt.Sprintf("{\"id\":%d}", messageId)))
	case convIdString != nil: //Unsuported method
		errorJSON, _ := json.Marshal(APIerror{"Must be a GET or POST request"})
		jsonResp(w, errorJSON, 405)
	case convIdString2 != nil && r.Method != "GET":
		errorJSON, _ := json.Marshal(APIerror{"Must be a GET request"})
		jsonResp(w, errorJSON, 405)
	case convIdString2 != nil:
		_convId, _ := strconv.ParseInt(convIdString2[1], 10, 16)
		convId := ConversationId(_convId)
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			start = 0
		}
		conv := getFullConversation(convId, start)
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
	userId := UserId(id)
	regexComments, _ := regexp.Compile("posts/(\\d+)/comments/?$")
	regexNoComments, _ := regexp.Compile("posts/(\\d+)/?$")
	commIdStringA := regexComments.FindStringSubmatch(r.URL.Path)
	commIdStringB := regexNoComments.FindStringSubmatch(r.URL.Path)
	switch {
	case !validateToken(userId, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case commIdStringA != nil && r.Method == "GET":
		_id, _ := strconv.ParseUint(commIdStringA[1], 10, 16)
		postId := PostId(_id)
		offset, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			offset = 0
		}
		comments, err := getComments(postId, offset)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			jsonComments, _ := json.Marshal(comments)
			w.Write(jsonComments)
		}
	case commIdStringA != nil && r.Method == "POST":
		_id, _ := strconv.ParseUint(commIdStringA[1], 10, 16)
		postId := PostId(_id)
		text := r.FormValue("text")
		commentId, err := createComment(postId, userId, text)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			w.Write([]byte(fmt.Sprintf("{\"id\":%d}", commentId)))
		}
	case commIdStringB != nil && r.Method == "GET":
		_id, _ := strconv.ParseUint(commIdStringB[1], 10, 16)
		postId := PostId(_id)
		log.Printf("%d", postId)
		//implement getting a specific post
	default:
		errorJSON, _ := json.Marshal(APIerror{"Method not supported"})
		jsonResp(w, errorJSON, 405)
	}
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	userId := UserId(id)
	token := r.FormValue("token")
	regexUser, _ := regexp.Compile("user/(\\d+)/?$")
	userIdString := regexUser.FindStringSubmatch(r.URL.Path)
	switch {
	case !validateToken(userId, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case r.Method != "GET":
		errorJSON, _ := json.Marshal(APIerror{"Method not supported"})
		jsonResp(w, errorJSON, 405)
	case userIdString != nil:
		u, _ := strconv.ParseUint(userIdString[1], 10, 16)
		profileId := UserId(u)
		user, err := getProfile(profileId)
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
	userId := UserId(id)
	token := r.FormValue("token")
	switch {
	case !validateToken(userId, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case r.Method != "GET":
		errorJSON, _ := json.Marshal(APIerror{"Method not supported"})
		jsonResp(w, errorJSON, 405)
	default:
		//awaitOneMessage will block until a message arrives over redis
		message := awaitOneMessage(userId)
		w.Write(message)
	}
}
