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
	r.HandleFunc("/api/{version}/login", loginHandler)
	r.HandleFunc("/api/{version}/register", registerHandler)
	r.HandleFunc("/api/{version}/conversations/live", liveConversationHandler)
	r.HandleFunc("/api/{version}/conversations/read_all", readAll).Methods("POST")
	r.HandleFunc("/api/{version}/conversations/read_all/", readAll).Methods("POST")
	r.HandleFunc("/api/{version}/conversations", getConversations).Methods("GET")
	r.HandleFunc("/api/{version}/conversations", postConversations).Methods("POST")
	r.HandleFunc("/api/{version}/conversations/{id:[0-9]+}", getSpecificConversation).Methods("GET")
	r.HandleFunc("/api/{version}/conversations/{id:[0-9]+}/", getSpecificConversation).Methods("GET")
	r.HandleFunc("/api/{version}/conversations/{id:[0-9]+}", putSpecificConversation).Methods("PUT")
	r.HandleFunc("/api/{version}/conversations/{id:[0-9]+}", deleteSpecificConversation).Methods("DELETE")
	r.HandleFunc("/api/{version}/conversations/{id:[0-9]+}/messages", getMessages).Methods("GET")
	r.HandleFunc("/api/{version}/conversations/{id:[0-9]+}/messages", postMessages).Methods("POST")
	r.HandleFunc("/api/{version}/conversations/{id:[0-9]+}/messages", putMessages).Methods("PUT")
	r.HandleFunc("/api/{version}/networks/{network:[0-9]+}/posts", getPosts).Methods("GET")
	r.HandleFunc("/api/{version}/networks/{network:[0-9]+}/posts", postPosts).Methods("POST")
	r.HandleFunc("/api/{version}/networks/{network:[0-9]+}", getNetwork).Methods("GET")
	r.HandleFunc("/api/{version}/networks/{network:[0-9]+}", putNetwork).Methods("PUT")
	r.HandleFunc("/api/{version}/networks/{network:[0-9]+}/users", postNetworkUsers).Methods("POST")
	r.HandleFunc("/api/{version}/networks/{network:[0-9]+}/users", getNetworkUsers).Methods("GET")
	r.HandleFunc("/api/{version}/networks", postNetworks).Methods("POST")
	r.HandleFunc("/api/{version}/posts", getPosts).Methods("GET")
	r.HandleFunc("/api/{version}/posts", postPosts).Methods("POST")
	r.HandleFunc("/api/{version}/posts/{id:[0-9]+}/comments", getComments).Methods("GET")
	r.HandleFunc("/api/{version}/posts/{id:[0-9]+}/comments", postComments).Methods("POST")
	r.HandleFunc("/api/{version}/posts/{id:[0-9]+}", getPost).Methods("GET")
	r.HandleFunc("/api/{version}/posts/{id:[0-9]+}/", getPost).Methods("GET")
	r.HandleFunc("/api/{version}/posts/{id:[0-9]+}/", deletePost).Methods("DELETE")
	r.HandleFunc("/api/{version}/posts/{id:[0-9]+}", deletePost).Methods("DELETE")
	r.HandleFunc("/api/{version}/posts/{id:[0-9]+}/images", postImages).Methods("POST")
	r.HandleFunc("/api/{version}/posts/{id:[0-9]+}/likes", postLikes).Methods("POST")
	r.HandleFunc("/api/{version}/posts/{id:[0-9+}/attendees", getAttendees).Methods("GET")
	r.HandleFunc("/api/{version}/posts/{id:[0-9+}/attendees", putAttendees).Methods("PUT")
	r.HandleFunc("/api/{version}/posts/{id:[0-9]+}/attending", attendHandler)
	r.HandleFunc("/api/{version}/user/{id:[0-9]+}", getUser).Methods("GET")
	r.HandleFunc("/api/{version}/user/{id:[0-9]+}/", getUser).Methods("GET")
	r.HandleFunc("/api/{version}/user/{id:[0-9]+}/posts", getUserPosts).Methods("GET")
	r.HandleFunc("/api/{version}/user/{id:[0-9]+}/unread", unread)
	r.HandleFunc("/api/{version}/user/{id:[0-9]+}/total_live", totalLiveConversations)
	r.HandleFunc("/api/{version}/user/", postUsers)
	r.HandleFunc("/api/{version}/user", postUsers)
	r.HandleFunc("/api/{version}/longpoll", longPollHandler)
	r.HandleFunc("/api/{version}/contacts", contactsHandler)
	r.HandleFunc("/api/{version}/contacts/{id:[0-9]+}", contactHandler)
	r.HandleFunc("/api/{version}/contacts/{id:[0-9]+}/", contactHandler)
	r.HandleFunc("/api/{version}/devices/{id}", deleteDevice)
	r.HandleFunc("/api/{version}/devices/{id}/", deleteDevice)
	r.HandleFunc("/api/{version}/devices", postDevice)
	r.HandleFunc("/api/{version}/upload", uploadHandler)
	r.HandleFunc("/api/{version}/profile/profile_image", profileImageHandler)
	r.HandleFunc("/api/{version}/profile/name", changeNameHandler)
	r.HandleFunc("/api/{version}/profile/change_pass", changePassHandler)
	r.HandleFunc("/api/{version}/profile/busy", busyHandler)
	r.HandleFunc("/api/{version}/profile/facebook", facebookAssociate)
	r.HandleFunc("/api/{version}/profile/attending", userAttending)
	r.HandleFunc("/api/{version}/profile/networks", getGroups)
	r.HandleFunc("/api/{version}/profile/networks/posts", getGroupPosts).Methods("GET")
	r.HandleFunc("/api/{version}/profile/networks/{network:[0-9]+}", deleteUserNetwork).Methods("DELETE")
	r.HandleFunc("/api/{version}/notifications", notificationHandler)
	r.HandleFunc("/api/{version}/fblogin", facebookHandler)
	r.HandleFunc("/api/{version}/verify/{token:[a-fA-F0-9]+}", verificationHandler)
	r.HandleFunc("/api/{version}/profile/request_reset", requestResetHandler)
	r.HandleFunc("/api/{version}/profile/reset/{id:[0-9]+}/{token}", resetPassHandler)
	r.HandleFunc("/api/{version}/resend_verification", resendVerificationHandler)
	r.HandleFunc("/api/{version}/invite_message", inviteMessageHandler)
	r.HandleFunc("/api/{version}/live", liveHandler)
	r.HandleFunc("/api/{version}/search/users/{query}", searchUsers).Methods("GET")
	r.HandleFunc("/api/{version}/admin/massmail", mm).Methods("POST")
	r.HandleFunc("/api/{version}/admin/masspush", newVersionNotificationHandler).Methods("POST")
	r.HandleFunc("/api/{version}/stats/user/{id:[0-9]+}/posts/{type}/{period}/{start}/{finish}", postsStatsHandler).Methods("GET")
	r.HandleFunc("/api/{version}/user/{id:[0-9]+}/posts", getUserPosts).Methods("GET")
	r.Handle("/api/{version}/ws", websocket.Handler(jsonServer))

	server := &http.Server{
		Addr:    ":" + conf.Port,
		Handler: r,
	}
	server.ListenAndServe()
}
