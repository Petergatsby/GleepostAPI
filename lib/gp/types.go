package gp

import (
	"time"
)

type UserId uint64
type NetworkId uint64
type MessageId uint64
type PostId uint64
type CommentId uint64
type ConversationId uint64
type NotificationId uint64

const (
	OSTART = iota
	OBEFORE
	OAFTER
)

type User struct {
	Id     UserId `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"profile_image"`
}

type Profile struct {
	User
	Desc     string `json:"tagline"`
	Network  Group  `json:"network"`
	Course   string `json:"course"`
	FullName string `json:"full_name"`
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

type Group struct {
	Network
	Image   string `json:"image,omitempty"`
	Desc    string `json:"description,omitempty"`
	Creator *User  `json:"creator,omitempty"`
}

type Message struct {
	Id   MessageId `json:"id"`
	By   User      `json:"by"`
	Text string    `json:"text"`
	Time time.Time `json:"timestamp"`
}

type Read struct {
	UserId   UserId    `json:"user"`
	LastRead MessageId `json:"last_read"`
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

type PostCore struct {
	Id   PostId    `json:"id"`
	By   User      `json:"by"`
	Time time.Time `json:"timestamp"`
	Text string    `json:"text"`
}

type Post struct {
	Network    NetworkId              `json:"-"`
	Id         PostId                 `json:"id"`
	By         User                   `json:"by"`
	Time       time.Time              `json:"timestamp"`
	Text       string                 `json:"text"`
	Images     []string               `json:"images"`
	Categories []PostCategory         `json:"categories,omitempty"`
	Attribs    map[string]interface{} `json:"attribs,omitempty"`
	Popularity int                    `json:"popularity,omitempty"`
	Attendees  int                    `json:"attendee_count,omitempty"`
	Group      *Group                 `json:"network,omitempty"`
}

type PostSmall struct {
	Post
	CommentCount int        `json:"comment_count"`
	LikeCount    int        `json:"like_count"`
	Likes        []LikeFull `json:"likes,omitempty"`
}

type PostFull struct {
	Post
	CommentCount int        `json:"comment_count"`
	LikeCount    int        `json:"like_count"`
	Comments     []Comment  `json:"comments"`
	Likes        []LikeFull `json:"likes"`
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

type LikeFull struct {
	User User      `json:"by"`
	Time time.Time `json:"timestamp"`
}

type Rule struct {
	NetworkID NetworkId
	Type      string
	Value     string
}

type Conversation struct {
	Id           ConversationId `json:"id"`
	LastActivity time.Time      `json:"lastActivity"`
	Participants []User         `json:"participants"`
	Read         []Read         `json:"read,omitempty"`
	Expiry       *Expiry        `json:"expiry,omitempty"`
}

type ConversationSmall struct {
	Conversation
	LastMessage *Message `json:"mostRecentMessage,omitempty"`
}

type ConversationAndMessages struct {
	Conversation
	Messages []Message `json:"messages"`
}

type MysqlConfig struct {
	MaxConns int
	User     string
	Pass     string
	Host     string
	Port     string
}

type RedisConfig struct {
	Proto        string
	Address      string
	MessageCache int
	PostCache    int
	CommentCache int
}

type AWSConfig struct {
	KeyId     string
	SecretKey string
}

type APNSConfig struct {
	CertFile   string
	KeyFile    string
	Production bool
}

type GCMConfig struct {
	APIKey string
}

type EmailConfig struct {
	User       string
	Pass       string
	Server     string
	Port       int
	From       string
	FromHeader string
}

type FacebookConfig struct {
	AppID     string
	AppSecret string
}

type Config struct {
	DevelopmentMode      bool
	Port                 string
	LoginOverride        bool
	RegisterOverride     bool
	MessagePageSize      int
	PostPageSize         int
	CommentPageSize      int
	ConversationPageSize int
	OnlineTimeout        int
	Expiry               int
	NewPushEnabled       bool
	Admins               int
	Mysql                MysqlConfig
	Redis                RedisConfig
	AWS                  AWSConfig
	APNS                 APNSConfig
	GCM                  GCMConfig
	Email                EmailConfig
	Facebook             FacebookConfig
}

type Device struct {
	User UserId `json:"user"`
	Type string `json:"type"`
	Id   string `json:"id"`
}

type Notification struct {
	Id   NotificationId `json:"id"`
	Type string         `json:"type"`
	Time time.Time      `json:"time"`
	By   User           `json:"user"`
	Seen bool           `json:"-"`
}

type PostNotification struct {
	Notification
	Post PostId `json:"post"`
}

type GroupNotification struct {
	Notification
	Group NetworkId `json:"network"`
}

func (c *MysqlConfig) ConnectionString() string {
	return c.User + ":" + c.Pass + "@tcp(" + c.Host + ":" + c.Port + ")/gleepost?charset=utf8"
}

type APIerror struct {
	Reason string `json:"error"`
}

type Created struct {
	Id uint64 `json:"id"`
}

type NewUser struct {
	Id     UserId `json:"id"`
	Status string `json:"status"`
}

type URLCreated struct {
	URL string `json:"url"`
}

type BusyStatus struct {
	Busy bool `json:"busy"`
}

type Liked struct {
	Post  PostId `json:"post"`
	Liked bool   `json:"liked"`
}

type CategoryId uint64

type PostCategory struct {
	Id   CategoryId `json:"id"`
	Tag  string     `json:"tag"`
	Name string     `json:"name"`
}

type Expiry struct {
	Time  time.Time `json:"time"`
	Ended bool      `json:"ended"`
}

func NewExpiry(d time.Duration) *Expiry {
	return &Expiry{Time: time.Now().Add(d), Ended: false}
}

func (e APIerror) Error() string {
	return e.Reason
}

var ENOSUCHUSER = APIerror{"No such user."}

type MsgQueue struct {
	Commands chan QueueCommand
	Messages chan []byte
}

type QueueCommand struct {
	Command string
	Value   string
}

type Event struct {
	Type     string      `json:"type"`
	Location string      `json:"location,omitempty"`
	Data     interface{} `json:"data"`
}
