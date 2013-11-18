package gp

import (
	"time"
	"io/ioutil"
	"sync"
	"syscall"
	"encoding/json"
	"os"
	"os/signal"
	"log"
)

type UserId uint64
type NetworkId uint64
type MessageId uint64
type PostId uint64
type CommentId uint64
type ConversationId uint64
type NotificationId uint64

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
	Comments []Comment  `json:"comments"`
	Likes    []LikeFull `json:"likes"`
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
	Proto   string
	Address string
}

type AWSConfig struct {
	KeyId     string
	SecretKey string
}

type APNSConfig struct {
	CertFile string
	KeyFile  string
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
	AppID      string
	AppSecret  string
}

type Config struct {
	UrlBase              string
	Port                 string
	LoginOverride        bool
	RegisterOverride     bool
	MessageCache         int
	PostCache            int
	CommentCache         int
	MessagePageSize      int
	PostPageSize         int
	CommentPageSize      int
	ConversationPageSize int
	OnlineTimeout        int
	Expiry               int
	Mysql                MysqlConfig
	Redis                RedisConfig
	AWS                  AWSConfig
	APNS                 APNSConfig
	Email                EmailConfig
	Facebook	     FacebookConfig
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

func (c *Config) ConnectionString() string {
	return c.Mysql.User + ":" + c.Mysql.Pass + "@tcp(" + c.Mysql.Host + ":" + c.Mysql.Port + ")/gleepost?charset=utf8"
}

type APIerror struct {
	Reason string `json:"error"`
}

type Created struct {
	Id uint64 `json:"id"`
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

type Expiry struct {
	Time time.Time `json:"time"`
}

func (e APIerror) Error() string {
	return e.Reason
}

var (
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

func init() {
	configInit()
}
