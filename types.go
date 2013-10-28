package main

import (
	"time"
)

type UserId uint64
type NetworkId uint64
type MessageId uint64
type PostId uint64
type CommentId uint64
type ConversationId uint64

type User struct {
	Id     UserId `json:"id"`
	Name   string `json:"username"`
	Avatar string `json:"profile_image"`
}

type Profile struct {
	User
	Desc    string  `json:"tagline"`
	Network Network `json:"network"`
	Course  string  `json:"course"`
}

type Contact struct {
	User
	YouConfirmed  bool `json:"you_confirmed"`
	TheyConfirmed bool `json:"they_confirmed"`
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
}

type PostFull struct {
	Post
	Comments []Comment `json:"comments"`
	Likes    []Like    `json:"likes"`
}

type Comment struct {
	Id   CommentId `json:"id"`
	Post PostId    `json:"-"`
	By   User      `json:"by"`
	Time time.Time `json:"timestamp"`
	Text string    `json:"text"`
}

type Like struct {
	UserID UserId
	Time   time.Time
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
	LastMessage  *Message  `json:"mostRecentMessage,omitempty"`
}

type ConversationAndMessages struct {
	Conversation
	Messages []Message `json:"messages"`
}

type MysqlConfig struct {
	MaxConns	   int
	User               string
	Pass               string
	Host               string
	Port               string
}

type RedisConfig struct {
	Proto              string
	Address            string
}

type AWSConfig struct {
	KeyId		string
	SecretKey	string
}

type Config struct {
	UrlBase                 string
	Port                    string
	LoginOverride           bool
	RegisterOverride	bool
	UploadPath		string
	MessageCache            int
	PostCache               int
	CommentCache            int
	MessagePageSize         int
	PostPageSize            int
	CommentPageSize         int
	Mysql			MysqlConfig
	Redis			RedisConfig
	AWS			AWSConfig
}

type Device struct {
	User UserId `json:"user"`
	Type string `json:"type"`
	Id   string `json:"id"`
}

func (c *Config) ConnectionString() string {
	return c.Mysql.User + ":" + c.Mysql.Pass + "@tcp(" + c.Mysql.Host + ":" + c.Mysql.Port + ")/gleepost?charset=utf8"
}

type APIerror struct {
	Reason string `json:"error"`
}

type Created struct {
	Id	uint64 `json:"id"`
}

func (e APIerror) Error() string {
	return e.Reason
}
