//package GleepostAPI is a simple REST API for gleepost.com
package main

import (
	"code.google.com/p/go.net/websocket"
	"github.com/draaglom/GleepostAPI/lib/gp"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	_ "net/http/pprof"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	conf := gp.GetConfig()
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
	http.HandleFunc(conf.UrlBase+"/profile/busy", busyHandler)
	http.HandleFunc(conf.UrlBase+"/notifications", notificationHandler)
	http.HandleFunc(conf.UrlBase+"/fblogin", facebookHandler)
	http.HandleFunc(conf.UrlBase+"/verify/", verificationHandler)
	http.Handle(conf.UrlBase+"/ws", websocket.Handler(jsonServer))
	server.ListenAndServe()
}
