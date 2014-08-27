//package GleepostAPI is a simple REST API for gleepost.com
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"code.google.com/p/go.net/websocket"
	"github.com/draaglom/GleepostAPI/lib/gp"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
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

//GetConfig returns a pointer to the current API configuration.
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
	out, err := exec.Command("git", "describe", "--tags").Output()
	if err != nil {
		log.Println(err)
	}
	fmt.Println(`  ________.__                                       __   `)
	fmt.Println(` /  _____/|  |   ____   ____ ______   ____  _______/  |_ `)
	fmt.Println(`/   \  ___|  | _/ __ \_/ __ \\____ \ /  _ \/  ___/\   __\`)
	fmt.Println(`\    \_\  \  |_\  ___/\  ___/|  |_> >  <_> )___ \  |  |  `)
	fmt.Println(` \______  /____/\___  >\___  >   __/ \____/____  > |__|  `)
	fmt.Printf(`        \/          \/     \/|__|              \/ %s`, out)
	fmt.Printf("\n")
}

func main() {
	ascii()
	runtime.GOMAXPROCS(runtime.NumCPU())
	conf := GetConfig()
	r := mux.NewRouter()
	base := r.PathPrefix("/api/{version}").Subrouter()
	base.HandleFunc("/login", loginHandler)
	base.HandleFunc("/register", registerHandler)
	base.HandleFunc("/conversations/live", liveConversationHandler)
	base.HandleFunc("/conversations/read_all", readAll).Methods("POST")
	base.HandleFunc("/conversations/read_all/", readAll).Methods("POST")
	base.HandleFunc("/conversations", getConversations).Methods("GET")
	base.HandleFunc("/conversations", postConversations).Methods("POST")
	base.HandleFunc("/conversations/{id:[0-9]+}", getSpecificConversation).Methods("GET")
	base.HandleFunc("/conversations/{id:[0-9]+}/", getSpecificConversation).Methods("GET")
	base.HandleFunc("/conversations/{id:[0-9]+}", putSpecificConversation).Methods("PUT")
	base.HandleFunc("/conversations/{id:[0-9]+}", deleteSpecificConversation).Methods("DELETE")
	base.HandleFunc("/conversations/{id:[0-9]+}/messages", getMessages).Methods("GET")
	base.HandleFunc("/conversations/{id:[0-9]+}/messages", postMessages).Methods("POST")
	base.HandleFunc("/conversations/{id:[0-9]+}/messages", putMessages).Methods("PUT")
	base.HandleFunc("/networks/{network:[0-9]+}/posts", getPosts).Methods("GET")
	base.HandleFunc("/networks/{network:[0-9]+}/posts", postPosts).Methods("POST")
	base.HandleFunc("/networks/{network:[0-9]+}", getNetwork).Methods("GET")
	base.HandleFunc("/networks/{network:[0-9]+}", putNetwork).Methods("PUT")
	base.HandleFunc("/networks/{network:[0-9]+}/users", postNetworkUsers).Methods("POST")
	base.HandleFunc("/networks/{network:[0-9]+}/users", getNetworkUsers).Methods("GET")
	base.HandleFunc("/networks", postNetworks).Methods("POST")
	base.HandleFunc("/posts", getPosts).Methods("GET")
	base.HandleFunc("/posts", postPosts).Methods("POST")
	base.HandleFunc("/posts/{id:[0-9]+}/comments", getComments).Methods("GET")
	base.HandleFunc("/posts/{id:[0-9]+}/comments", postComments).Methods("POST")
	base.HandleFunc("/posts/{id:[0-9]+}", getPost).Methods("GET")
	base.HandleFunc("/posts/{id:[0-9]+}/", getPost).Methods("GET")
	base.HandleFunc("/posts/{id:[0-9]+}/", deletePost).Methods("DELETE")
	base.HandleFunc("/posts/{id:[0-9]+}", deletePost).Methods("DELETE")
	base.HandleFunc("/posts/{id:[0-9]+}/images", postImages).Methods("POST")
	base.HandleFunc("/posts/{id:[0-9]+}/videos", postVideos).Methods("POST")
	base.HandleFunc("/posts/{id:[0-9]+}/likes", postLikes).Methods("POST")
	base.HandleFunc("/posts/{id:[0-9]+}/attendees", getAttendees).Methods("GET")
	base.HandleFunc("/posts/{id:[0-9]+}/attendees", putAttendees).Methods("PUT")
	base.HandleFunc("/posts/{id:[0-9]+}/attending", attendHandler)
	base.HandleFunc("/user/{id:[0-9]+}", getUser).Methods("GET")
	base.HandleFunc("/user/{id:[0-9]+}/", getUser).Methods("GET")
	base.HandleFunc("/user/{id:[0-9]+}/posts", getUserPosts).Methods("GET")
	base.HandleFunc("/user/{id:[0-9]+}/unread", unread)
	base.HandleFunc("/user/{id:[0-9]+}/total_live", totalLiveConversations)
	base.HandleFunc("/user/", postUsers)
	base.HandleFunc("/user", postUsers)
	base.HandleFunc("/longpoll", longPollHandler)
	base.HandleFunc("/contacts", contactsHandler)
	base.HandleFunc("/contacts/{id:[0-9]+}", contactHandler)
	base.HandleFunc("/contacts/{id:[0-9]+}/", contactHandler)
	base.HandleFunc("/devices/{id}", deleteDevice)
	base.HandleFunc("/devices/{id}/", deleteDevice)
	base.HandleFunc("/devices", postDevice)
	base.HandleFunc("/upload", uploadHandler)
	base.HandleFunc("/upload/{id}", getUpload)

	base.HandleFunc("/videos", postVideoUpload).Methods("POST")
	base.HandleFunc("/videos/{id}", getVideos).Methods("GET")

	base.HandleFunc("/profile/profile_image", profileImageHandler)
	base.HandleFunc("/profile/name", changeNameHandler)
	base.HandleFunc("/profile/change_pass", changePassHandler)
	base.HandleFunc("/profile/busy", busyHandler)
	base.HandleFunc("/profile/facebook", facebookAssociate)
	base.HandleFunc("/profile/attending", userAttending)
	base.HandleFunc("/profile/networks", getGroups)
	base.HandleFunc("/profile/networks/posts", getGroupPosts).Methods("GET")
	base.HandleFunc("/profile/networks/{network:[0-9]+}", deleteUserNetwork).Methods("DELETE")
	base.HandleFunc("/notifications", notificationHandler)
	base.HandleFunc("/fblogin", facebookHandler)
	base.HandleFunc("/verify/{token:[a-fA-F0-9]+}", verificationHandler)
	base.HandleFunc("/profile/request_reset", requestResetHandler)
	base.HandleFunc("/profile/reset/{id:[0-9]+}/{token}", resetPassHandler)
	base.HandleFunc("/resend_verification", resendVerificationHandler)
	base.HandleFunc("/invite_message", inviteMessageHandler)
	base.HandleFunc("/live", liveHandler)
	base.HandleFunc("/search/users/{query}", searchUsers).Methods("GET")
	base.HandleFunc("/search/groups/{query}", searchGroups).Methods("GET")
	base.HandleFunc("/admin/massmail", mm).Methods("POST")
	base.HandleFunc("/admin/masspush", newVersionNotificationHandler).Methods("POST")
	base.HandleFunc("/admin/posts/duplicate", postDuplicate).Methods("POST")
	base.HandleFunc("/admin/posts/copy_attribs", copyAttribs).Methods("POST")
	base.HandleFunc("/stats/user/{id:[0-9]+}/posts/{type}/{period}/{start}/{finish}", postsStatsHandler).Methods("GET")
	base.HandleFunc("/stats/posts/{id:[0-9]+}/{type}/{period}/{start}/{finish}", individualPostStats).Methods("GET")
	base.HandleFunc("/user/{id:[0-9]+}/posts", getUserPosts).Methods("GET")
	base.HandleFunc("/reports", postReports).Methods("POST")
	base.Handle("/ws", websocket.Handler(jsonServer))

	server := &http.Server{
		Addr:    ":" + conf.Port,
		Handler: r,
	}
	server.ListenAndServe()
}
