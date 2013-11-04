package main

import (
	"database/sql"
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
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
	db.SetMaxIdleConns(conf.Mysql.MaxConns)
	err = prepare(db)
	if err != nil {
		log.Fatal(err)
	}
	go keepalive(db)
	server := &http.Server{
		Addr:         ":" + conf.Port,
		ReadTimeout:  70 * time.Second,
		WriteTimeout: 70 * time.Second,
	}
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
	http.HandleFunc(conf.UrlBase+"/contacts", contactsHandler)
	http.HandleFunc(conf.UrlBase+"/contacts/", anotherContactsHandler)
	http.HandleFunc(conf.UrlBase+"/devices", deviceHandler)
	http.HandleFunc(conf.UrlBase+"/upload", uploadHandler)
	http.HandleFunc(conf.UrlBase+"/profile/profile_image", profileImageHandler)
	http.HandleFunc(conf.UrlBase+"/profile/busy", busyHandler)
	http.HandleFunc(conf.UrlBase+"/notifications", notificationHandler)
	server.ListenAndServe()
}
