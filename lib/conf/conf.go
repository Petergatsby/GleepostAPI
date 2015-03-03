package conf

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

var (
	config     *Config
	configLock = new(sync.RWMutex)
)

func init() {
	configInit()
}

//GetConfig returns a pointer to the current API configuration.
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

//MysqlConfig represents the database configuration.
type MysqlConfig struct {
	MaxConns int
	User     string
	Pass     string
	Host     string
	Port     string
}

//ConnectionString returns the db/sql string for connecting to MySQL based on this config.
func (c *MysqlConfig) ConnectionString() string {
	return c.User + ":" + c.Pass + "@tcp(" + c.Host + ":" + c.Port + ")/gleepost?charset=utf8mb4"
}

//RedisConfig represents the cache configuration.
type RedisConfig struct {
	Proto        string
	Address      string
	MessageCache int //Max number of messages per conversation to cache
	PostCache    int //Max number of posts per network to cache
	CommentCache int //Max number of comments per post to cache
}

//AWSConfig contains AWS credentials.
type AWSConfig struct {
	KeyID     string
	SecretKey string
}

//APNSConfig contains Apple push credentials
type APNSConfig struct {
	CertFile   string
	KeyFile    string
	Production bool //Targeting real servers or sandbox?
}

//GCMConfig contains GCM credentials.
type GCMConfig struct {
	APIKey string
}

//EmailConfig contains SMTP credentials.
type EmailConfig struct {
	User       string
	Pass       string
	Server     string
	Port       int
	From       string
	FromHeader string
}

//FacebookConfig contains facebook credentials.
type FacebookConfig struct {
	AppID     string
	AppSecret string
}

//Config defines all the available configuration for the API.
type Config struct {
	DevelopmentMode      bool
	Port                 string
	LoginOverride        bool
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
	Pushers              []PusherConfig
	Email                EmailConfig
	Facebook             FacebookConfig
	Futures              []ConfigFuture
	Statsd               string
}

//PusherConfig represents the configuration for sending push notifications to a particular app.
type PusherConfig struct {
	AppName string
	APNS    APNSConfig
	GCM     GCMConfig
}

//PostFuture represents a commitment to keeping an event's event-time in the future by a specified duration.
type PostFuture struct {
	Post   gp.PostID     `json:"id"`
	Future time.Duration `json:"future"`
}

//ConfigFuture is PostFuture but without the duration because json can't unmarshal it apparently.
type ConfigFuture struct {
	Post   gp.PostID `json:"id"`
	Future string    `json:"future"`
}

//ParseDuration converts a ConfigFuture into a PostFuture
func (c ConfigFuture) ParseDuration() (pf PostFuture) {
	pf.Post = c.Post
	duration, err := time.ParseDuration(c.Future)
	if err != nil {
		log.Println("Error parsing duration:", err)
		return
	}
	pf.Future = duration
	return pf
}
