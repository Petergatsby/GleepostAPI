package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"time"
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
