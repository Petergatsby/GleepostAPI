package conf

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	config     *Config
	configLock = new(sync.RWMutex)
	confPath   = flag.String("conf", "conf.json", "path to config file")
)

func init() {
	flag.Parse()
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
	file, err := ioutil.ReadFile(*confPath)
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
	Proto   string
	Address string
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
	MessagePageSize      int
	PostPageSize         int
	NotificationPageSize int
	CommentPageSize      int
	ConversationPageSize int
	Mysql                MysqlConfig
	Redis                RedisConfig
	AWS                  AWSConfig
	Pushers              []PusherConfig
	Email                EmailConfig
	Facebook             FacebookConfig
	Statsd               string
	ElasticSearch        string
}

//PusherConfig represents the configuration for sending push notifications to a particular app.
type PusherConfig struct {
	AppName string
	APNS    APNSConfig
	GCM     GCMConfig
}
