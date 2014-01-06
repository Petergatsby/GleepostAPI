//package GleepostAPI is a simple REST API for gleepost.com
package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"github.com/draaglom/GleepostAPI/lib/gp"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func loadConfig(fail bool) {
	file, err := ioutil.ReadFile("conf.json")
	if err != nil {
		log.Println("Opening config failed: ", err)
		if fail {
			os.Exit(1)
		}
	}

	c := new(gp.Config)
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

func GetConfig() *gp.Config {
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
	conf := GetConfig()
	server := &http.Server{
		Addr: ":" + conf.Port,
	}
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
	http.HandleFunc(conf.UrlBase+"/devices/", deleteDeviceHandler)
	http.HandleFunc(conf.UrlBase+"/devices", deviceHandler)
	http.HandleFunc(conf.UrlBase+"/upload", uploadHandler)
	http.HandleFunc(conf.UrlBase+"/profile/profile_image", profileImageHandler)
	http.HandleFunc(conf.UrlBase+"/profile/change_pass", changePassHandler)
	http.HandleFunc(conf.UrlBase+"/profile/busy", busyHandler)
	http.HandleFunc(conf.UrlBase+"/notifications", notificationHandler)
	http.HandleFunc(conf.UrlBase+"/fblogin", facebookHandler)
	http.HandleFunc(conf.UrlBase+"/verify/", verificationHandler)
	http.HandleFunc(conf.UrlBase+"/profile/request_reset", requestResetHandler)
	http.HandleFunc(conf.UrlBase+"/profile/reset/", resetPassHandler)
	http.Handle(conf.UrlBase+"/ws", websocket.Handler(jsonServer))
	server.ListenAndServe()
}
