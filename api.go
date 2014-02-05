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
	"fmt"
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
	fmt.Println(`        \/          \/     \/|__|              \/        `)
}

func main() {
	ascii()
	runtime.GOMAXPROCS(runtime.NumCPU())
	conf := GetConfig()
	server := &http.Server{
		Addr: ":" + conf.Port,
	}
	http.HandleFunc(conf.UrlBase+"/login", loginHandler)
	http.HandleFunc(conf.UrlBase+"/register", registerHandler)
	http.HandleFunc(conf.UrlBase+"/conversations/live", liveConversationHandler)
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
	http.HandleFunc(conf.UrlBase+"/profile/name", changeNameHandler)
	http.HandleFunc(conf.UrlBase+"/profile/change_pass", changePassHandler)
	http.HandleFunc(conf.UrlBase+"/profile/busy", busyHandler)
	http.HandleFunc(conf.UrlBase+"/notifications", notificationHandler)
	http.HandleFunc(conf.UrlBase+"/fblogin", facebookHandler)
	http.HandleFunc(conf.UrlBase+"/verify/", verificationHandler)
	http.HandleFunc(conf.UrlBase+"/profile/request_reset", requestResetHandler)
	http.HandleFunc(conf.UrlBase+"/profile/reset/", resetPassHandler)
	http.HandleFunc(conf.UrlBase+"/resend_verification", resendVerificationHandler)
	http.HandleFunc(conf.UrlBase+"/invite_message", inviteMessageHandler)
	http.HandleFunc(conf.UrlBase+"/live", liveHandler)
	http.Handle(conf.UrlBase+"/ws", websocket.Handler(jsonServer))
	server.ListenAndServe()
}
