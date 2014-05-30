package gp

import (
	"time"
)

type UserID uint64
type NetworkID uint64
type MessageID uint64
type PostID uint64
type CommentID uint64
type ConversationID uint64
type NotificationID uint64

const (
	OSTART = iota
	OBEFORE
	OAFTER
)

type User struct {
	ID     UserID `json:"id"`
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
	ID   NetworkID `json:"id"`
	Name string    `json:"name"`
}

type Group struct {
	Network
	Image   string `json:"image,omitempty"`
	Desc    string `json:"description,omitempty"`
	Creator *User  `json:"creator,omitempty"`
}

type Message struct {
	ID   MessageID `json:"id"`
	By   User      `json:"by"`
	Text string    `json:"text"`
	Time time.Time `json:"timestamp"`
}

type Read struct {
	UserID   UserID    `json:"user"`
	LastRead MessageID `json:"last_read"`
}

type RedisMessage struct {
	Message
	Conversation ConversationID `json:"conversation_id"`
}

type Token struct {
	UserID UserID    `json:"id"`
	Token  string    `json:"value"`
	Expiry time.Time `json:"expiry"`
}

type PostCore struct {
	ID   PostID    `json:"id"`
	By   User      `json:"by"`
	Time time.Time `json:"timestamp"`
	Text string    `json:"text"`
}

type Post struct {
	Network    NetworkID              `json:"-"`
	ID         PostID                 `json:"id"`
	By         User                   `json:"by"`
	Time       time.Time              `json:"timestamp"`
	Text       string                 `json:"text"`
	Images     []string               `json:"images"`
	Videos     []string               `json:"videos,omitempty"`
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
	ID   CommentID `json:"id"`
	Post PostID    `json:"-"`
	By   User      `json:"by"`
	Time time.Time `json:"timestamp"`
	Text string    `json:"text"`
}

type Like struct {
	UserID UserID
	Time   time.Time
}

type LikeFull struct {
	User User      `json:"by"`
	Time time.Time `json:"timestamp"`
}

type Rule struct {
	NetworkID NetworkID
	Type      string
	Value     string
}

type Conversation struct {
	ID           ConversationID `json:"id"`
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
	User UserID `json:"user"`
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Notification struct {
	ID   NotificationID `json:"id"`
	Type string         `json:"type"`
	Time time.Time      `json:"time"`
	By   User           `json:"user"`
	Seen bool           `json:"seen"`
}

type PostNotification struct {
	Notification
	Post PostID `json:"post"`
}

type GroupNotification struct {
	Notification
	Group NetworkID `json:"network"`
}

func (c *MysqlConfig) ConnectionString() string {
	return c.User + ":" + c.Pass + "@tcp(" + c.Host + ":" + c.Port + ")/gleepost?charset=utf8"
}

type APIerror struct {
	Reason string `json:"error"`
}

type Created struct {
	ID uint64 `json:"id"`
}

type NewUser struct {
	ID     UserID `json:"id"`
	Status string `json:"status"`
}

type URLCreated struct {
	URL string `json:"url"`
}

type BusyStatus struct {
	Busy bool `json:"busy"`
}

type Liked struct {
	Post  PostID `json:"post"`
	Liked bool   `json:"liked"`
}

type CategoryID uint64

type PostCategory struct {
	ID   CategoryID `json:"id"`
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

//Video contains a URL for an .mp4 and .webm encode of the same video, as well as thumbnails where available.
type Video struct {
	//uploaded marks whether this is just a local copy or refers to properly hosted files
	Uploaded bool     `json:"-"`
	ID       VideoID  `json:"id"`
	MP4      string   `json:"mp4,omitempty"`
	WebM     string   `json:"webm,omitempty"`
	Thumbs   []string `json:"thumbnails,omitempty"`
}

//VideoID is a reference to an uploaded video.
type VideoID uint64

//UploadStatus represents the status of an uploaded video.
type UploadStatus struct {
	Status string `json:"status"`
	Video
}
