//package GleepostAPI is a simple REST API for gleepost.com
package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"github.com/draaglom/GleepostAPI/lib/gp"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
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

func ascii() {
	fmt.Println(`  ________.__                                       __   `)
	fmt.Println(` /  _____/|  |   ____   ____ ______   ____  _______/  |_ `)
	fmt.Println(`/   \  ___|  | _/ __ \_/ __ \\____ \ /  _ \/  ___/\   __\`)
	fmt.Println(`\    \_\  \  |_\  ___/\  ___/|  |_> >  <_> )___ \  |  |  `)
	fmt.Println(` \______  /____/\___  >\___  >   __/ \____/____  > |__|  `)
	fmt.Printf(`        \/          \/     \/|__|              \/ %s`, api.Config.UrlBase)
	fmt.Printf("\n")
}

func main() {
	ascii()
	runtime.GOMAXPROCS(runtime.NumCPU())
	conf := GetConfig()
	r := mux.NewRouter()
	r.HandleFunc(conf.UrlBase+"/login", loginHandler)
	r.HandleFunc(conf.UrlBase+"/register", registerHandler)
	r.HandleFunc(conf.UrlBase+"/conversations/live", liveConversationHandler)
	r.HandleFunc(conf.UrlBase+"/conversations", getConversations).Methods("GET")
	r.HandleFunc(conf.UrlBase+"/conversations", postConversations).Methods("POST")
	r.HandleFunc(conf.UrlBase+"/conversations/{id:[0-9]+}", getSpecificConversation).Methods("GET")
	r.HandleFunc(conf.UrlBase+"/conversations/{id:[0-9]+}", putSpecificConversation).Methods("PUT")
	r.HandleFunc(conf.UrlBase+"/conversations/{id:[0-9]+}", deleteSpecificConversation).Methods("DELETE")
	r.HandleFunc(conf.UrlBase+"/conversations/{id:[0-9]+}/messages", getMessages).Methods("GET")
	r.HandleFunc(conf.UrlBase+"/conversations/{id:[0-9]+}/messages", postMessages).Methods("POST")
	r.HandleFunc(conf.UrlBase+"/conversations/{id:[0-9]+}/messages", putMessages).Methods("PUT")
	r.HandleFunc(conf.UrlBase+"/posts", getPosts).Methods("GET")
	r.HandleFunc(conf.UrlBase+"/posts", postPosts).Methods("POST")
	r.HandleFunc(conf.UrlBase+"/posts/{id:[0-9]+}/comments", getComments).Methods("GET")
	r.HandleFunc(conf.UrlBase+"/posts/{id:[0-9]+}/comments", postComments).Methods("POST")
	r.HandleFunc(conf.UrlBase+"/posts/{id:[0-9]+}", getPost).Methods("GET")
	r.HandleFunc(conf.UrlBase+"/posts/{id:[0-9]+}/images", postImages).Methods("POST")
	r.HandleFunc(conf.UrlBase+"/posts/{id:[0-9]+}/likes", postLikes).Methods("POST")
	r.HandleFunc(conf.UrlBase+"/posts/{id:[0-9]+}/attending", attendHandler)
	r.HandleFunc(conf.UrlBase+"/user/{id:[0-9]+}", getUser).Methods("GET")
	r.HandleFunc(conf.UrlBase+"/user/{id:[0-9]+}/posts", getUserPosts).Methods("GET")
	r.HandleFunc(conf.UrlBase+"/longpoll", longPollHandler)
	r.HandleFunc(conf.UrlBase+"/contacts", contactsHandler)
	r.HandleFunc(conf.UrlBase+"/contacts/{id:[0-9]+}", contactHandler)
	r.HandleFunc(conf.UrlBase+"/devices/{id}", deleteDevice)
	r.HandleFunc(conf.UrlBase+"/devices", postDevice)
	r.HandleFunc(conf.UrlBase+"/upload", uploadHandler)
	r.HandleFunc(conf.UrlBase+"/profile/profile_image", profileImageHandler)
	r.HandleFunc(conf.UrlBase+"/profile/name", changeNameHandler)
	r.HandleFunc(conf.UrlBase+"/profile/change_pass", changePassHandler)
	r.HandleFunc(conf.UrlBase+"/profile/busy", busyHandler)
	r.HandleFunc(conf.UrlBase+"/notifications", notificationHandler)
	r.HandleFunc(conf.UrlBase+"/fblogin", facebookHandler)
	r.HandleFunc(conf.UrlBase+"/verify/{token:[a-fA-F0-9]+}", verificationHandler)
	r.HandleFunc(conf.UrlBase+"/profile/request_reset", requestResetHandler)
	r.HandleFunc(conf.UrlBase+"/profile/reset/{id:[0-9]+}/{token}", resetPassHandler)
	r.HandleFunc(conf.UrlBase+"/resend_verification", resendVerificationHandler)
	r.HandleFunc(conf.UrlBase+"/invite_message", inviteMessageHandler)
	r.HandleFunc(conf.UrlBase+"/live", liveHandler)
	r.Handle(conf.UrlBase+"/ws", websocket.Handler(jsonServer))
	server := &http.Server{
		Addr:    ":" + conf.Port,
		Handler: r,
	}
	server.ListenAndServe()
}
