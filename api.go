package main

import (
	"code.google.com/p/go.crypto/bcrypt"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"log"
	"math/big"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type User struct {
	Id   uint64
	Name string
}

type Message struct {
	Sender  string    `json:"sender"`
	Time    time.Time `json:"time"`
	Text    string    `json:"text"`
	TopicID uint64    `json:"topic"`
}

type Token struct {
	UserId uint64    `json:"id"`
	Token  string    `json:"value"`
	Expiry time.Time `json:"expiry"`
}

type Topic struct {
	TopicID      uint64     `json:"id"`
	Time         time.Time  `json:"time"`
	Messages     []*Message `json:"messages"`
	Text         string     `json:"text"`
	Participants []uint64   `json:"users"`
	IsPost       bool
}

type Post struct {
	Topic     Topic      `json:"topic"`
	LikeHates []LikeHate `json:"like_hate"`
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
	Id	int64	`json:"id"`
	Participants []User `json:"participants"`
}

const (
	ruleSelect         = "SELECT network_id, rule_type, rule_value FROM net_rules"
	createUser         = "INSERT INTO users(name, password, email) VALUES (?,?,?)"
	ConnectionString   = "gp:PnOaw3XzP6Tlq6fWvvVv@tcp(localhost:3306)/gleepost?charset=utf8"
	PassSelect         = "SELECT id, password FROM users WHERE name = ?"
	messageInsert      = "INSERT INTO new_messages(`by`, conversation_id, text) VALUES (?,?,?)"
	randomSelect      = "SELECT id, name FROM users ORDER BY RAND()"
	conversationInsert      = "INSERT INTO conversations (initiator) VALUES (?)"
	userSelect      = "SELECT id, name FROM users WHERE id=?"
	participantInsert      = "INSERT INTO conversation_participants (conversation_id, participant_id) VALUES (?,?)"
	MaxConnectionCount = 100
	UrlBase            = "/api/v0.6"
)

var (
	messages          = make([]*Message, 10)
	tokens            = make([]Token, 10)
	posts             = make([]*Post, 10)
	topics            = make([]*Topic, 10)
	ruleStatement     *sql.Stmt
	registerStatement *sql.Stmt
	passStatement     *sql.Stmt
	messageStatement  *sql.Stmt
	randomStatement	  *sql.Stmt
	userStatement	  *sql.Stmt
	conversationStatement	  *sql.Stmt
	participantStatement	  *sql.Stmt
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	db, err := sql.Open("mysql", ConnectionString)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	db.SetMaxIdleConns(MaxConnectionCount)
	ruleStatement, err = db.Prepare(ruleSelect)
	if err != nil {
		log.Fatal(err)
	}
	registerStatement, err = db.Prepare(createUser)
	if err != nil {
		log.Fatal(err)
	}
	passStatement, err = db.Prepare(PassSelect)
	if err != nil {
		log.Fatal(err)
	}
	messageStatement, err = db.Prepare(messageInsert)
	if err != nil {
		log.Fatal(err)
	}
	randomStatement, err = db.Prepare(randomSelect)
	if err != nil {
		log.Fatal(err)
	}
	conversationStatement, err = db.Prepare(conversationInsert)
	if err != nil {
		log.Fatal(err)
	}
	userStatement, err = db.Prepare(userSelect)
	if err != nil {
		log.Fatal(err)
	}
	participantStatement, err = db.Prepare(participantInsert)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc(UrlBase+"/login", loginHandler)
	http.HandleFunc(UrlBase+"/register", registerHandler)
	http.HandleFunc(UrlBase+"/messages", messageHandler)
	http.HandleFunc(UrlBase+"/newconversation", newConversationHandler)
	http.ListenAndServe("dev.gleepost.com:8080", nil)
}

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

func validateEmail(email string) bool {
	if !looksLikeEmail(email) {
		return (false)
	} else {
		rows, err := ruleStatement.Query()
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

func registerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		http.Error(w, "{\"error\":\"Must be a POST request!\"}", 405)
	} else {
		user := r.FormValue("user")
		pass := r.FormValue("pass")
		email := r.FormValue("email")
		if len(user) == 0 {
			errMsg := "Missing parameter: user"
			w.Write([]byte("{\"success\":false, \"error\":\"" + errMsg + "\"}"))
		} else if len(pass) == 0 {
			errMsg := "Missing parameter: pass"
			w.Write([]byte("{\"success\":false, \"error\":\"" + errMsg + "\"}"))
		} else if len(email) == 0 {
			errMsg := "Missing parameter: email"
			w.Write([]byte("{\"success\":false, \"error\":\"" + errMsg + "\"}"))
		} else if !validateEmail(email) {
			errMsg := "Invalid email"
			w.Write([]byte("{\"success\":false, \"error\":\"" + errMsg + "\"}"))
		} else {

			hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
			if err != nil {
				errMsg := "Password hashing failure"
				w.Write([]byte("{\"success\":false, \"error\":\"" + errMsg + "\"}"))
			} else {
				_, err := registerStatement.Exec(user, hash, email)
				if err != nil {
					if strings.HasPrefix(err.Error(), "Error 1062") {
						errMsg := "Username or email already registered"
						w.Write([]byte("{\"success\":false, \"error\":\"" + errMsg + "\"}"))
					} else {
						errMsg := err.Error()
						w.Write([]byte("{\"success\":false, \"error\":\"" + errMsg + "\"}"))
					}
				} else {
					w.Write([]byte("{\"success\":true}"))
					//also send activation email!
				}
			}
		}
	}
}

func validateToken(id uint64, token string) bool {
	return (true)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "POST" {
		user := r.FormValue("user")
		pass := []byte(r.FormValue("pass"))
		hash := make([]byte, 256)
		var id uint64
		err := passStatement.QueryRow(user).Scan(&id, &hash)
		if err != nil {
			/*
				if (err.(sql.ErrNoRows)) {
					w.Write([]byte("{\"success\":false}"))
				} else {
					w.Write([]byte("{\"success\":false}"))
				}*/
			w.Write([]byte("{\"success\":false}"))
		} else {
			err := bcrypt.CompareHashAndPassword(hash, pass)
			if err != nil {
				w.Write([]byte("{\"success\":false}"))
			} else {
				token := createToken(id)
				tokenJSON, _ := json.Marshal(token)
				tokens = append(tokens, token)
				w.Write([]byte("{\"success\":true, \"token\":"))
				w.Write(tokenJSON)
				w.Write([]byte("}"))
			}
		}
	} else {
		http.Error(w, "{\"error\":\"Must be a POST request!\"}", 405)
	}
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "GET" {
		id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
		token := r.FormValue("token")
		if !validateToken(id, token) {
			errMsg := "Invalid credentials"
			w.Write([]byte("{\"success\":false, \"error\":\"" + errMsg + "\"}"))
		} else {
			//
		}
	} else if r.Method == "POST" {
		id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
		token := r.FormValue("token")
		if !validateToken(id, token) {
			errMsg := "Invalid credentials"
			w.Write([]byte("{\"success\":false, \"error\":\"" + errMsg + "\"}"))
		}
	}
}

func createConversation(id uint64, nParticipants int) Conversation {
	r, _ := conversationStatement.Exec(id)
	conversation := Conversation{}
	conversation.Id, _ = r.LastInsertId()
	participants := make([]User, 0, 10)
	user := getUser(id)
	participants = append(participants, user)
	nParticipants--

	rows, err := randomStatement.Query()
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
	for _, u := range(participants) {
		_,err := participantStatement.Exec(conversation.Id, u.Id)
		if err != nil {
			log.Fatalf("Error adding user to conversation: %v", err)
		}
	}
	conversation.Participants = participants
	return(conversation)
}

func getUser(id uint64) User {
	user := User{}
	err := userStatement.QueryRow(id).Scan(&user.Id, &user.Name)
	if err != nil {
		log.Fatalf("Error getting user: %v", err)
	} else {
		//
	}
	return(user)
}

func newConversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		http.Error(w, "{\"error\":\"Must be a POST request!\"}", 405)
	} else {
		id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
		token := r.FormValue("token")
		if !validateToken(id, token) {
			errMsg := "Invalid credentials"
			w.Write([]byte("{\"success\":false, \"error\":\"" + errMsg + "\"}"))
		} else {
			conversation := createConversation(id, 2)
			conversationJSON, _ := json.Marshal(conversation)
			w.Write([]byte("{\"success\":true, \"conversation\":"))
			w.Write(conversationJSON)
			w.Write([]byte("}"))
		}
	}
}

func messageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		http.Error(w, "{\"error\":\"Must be a POST request!\"}", 405)
	} else {
		user := r.FormValue("user")
		topic := r.FormValue("topic")
		topicID, _ := strconv.ParseUint(r.FormValue("topicid"), 10, 16)
		message := r.FormValue("message")
		if len(user) > 0 && len(topic) > 0 && len(message) > 0 {
			m := Message{user, time.Now(), message, topicID}
			messages = append(messages, &m)
			w.Write([]byte("{\"success\":true}"))
		} else {
			w.Write([]byte("{\"success\":false}"))
		}
	}
}

func createPostHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		http.Error(w, "{\"error\":\"Must be a POST request!\"}", 405)
	} else {
		text := r.FormValue("text")
		if len(text) > 0 {
			//Create a topic yo
			w.Write([]byte("{\"success\":true}"))
		} else {
			w.Write([]byte("{\"success\":false}"))
		}
	}
}

func createTopicHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		http.Error(w, "{\"error\":\"Must be a POST request!\"}", 405)
	} else {
		text := r.FormValue("text")

		usrs := make([]uint64, 0, 100)
		err := json.Unmarshal([]byte(r.FormValue("to")), usrs)
		if err != nil {
			//malformed json lol
		} else {
			if len(text) > 0 {
				//Create a topic yo
				bigid, _ := rand.Int(rand.Reader, big.NewInt(int64(^uint(0)>>1)))
				id := bigid.Uint64()
				t := time.Now().UTC()
				messages := make([]*Message, 0, 100)
				topic := Topic{id, t, messages, text, usrs, false}
				topics = append(topics, &topic)
				w.Write([]byte("{\"success\":true}"))
			} else {
				w.Write([]byte("{\"success\":false}"))
			}
		}
	}
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: get /profile listing topics by time new/old
}

func recvHandler(w http.ResponseWriter, r *http.Request) {
	topicstring := r.URL.Path[6:]
	topicID, _ := strconv.ParseUint(topicstring, 10, 16)
	history := make([]*Message, 0, 100)
	for _, m := range messages {
		if m.TopicID == topicID {
			history = append(history, m)
		}
	}
	resp, _ := json.Marshal(history)
	w.Write(resp)
}
