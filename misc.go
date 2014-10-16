package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

var (
	config     *gp.Config
	configLock = new(sync.RWMutex)
	api        *lib.API
)

func init() {
	base.HandleFunc("/invite_message", inviteMessageHandler)
	base.HandleFunc("/", optionsHandler).Methods("OPTIONS")
}

//EUNSUPPORTED = 405
var EUNSUPPORTED = gp.APIerror{Reason: "Method not supported"}

//ENOTFOUNT = 404
var ENOTFOUND = gp.APIerror{Reason: "404 not found"}

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

func optionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(200)
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

func missingParamErr(param string) *gp.APIerror {
	return &gp.APIerror{Reason: "Missing parameter: " + param}
}

func init() {
	configInit()
	config = GetConfig()
	api = lib.New(*config)
	go api.FeedbackDaemon(60)
	go api.EndOldConversations()
	api.PeriodicSummary(time.Date(2014, time.April, 9, 8, 0, 0, 0, time.UTC), time.Duration(24*time.Hour))
	var futures []gp.PostFuture
	for _, f := range config.Futures {
		futures = append(futures, f.ParseDuration())
	}
	go api.KeepPostsInFuture(30*time.Minute, futures)

}

func jsonResponse(w http.ResponseWriter, resp interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	marshaled, err := json.Marshal(resp)
	if err != nil {
		marshaled, _ = json.Marshal(gp.APIerror{Reason: err.Error()})
		w.WriteHeader(500)
		w.Write(marshaled)
	} else {
		w.WriteHeader(code)
		w.Write(marshaled)
	}
}

func jsonErr(w http.ResponseWriter, err error, code int) {
	switch err.(type) {
	case gp.APIerror:
		jsonResponse(w, err, code)
	default:
		jsonResponse(w, gp.APIerror{Reason: err.Error()}, code)
	}
}
func inviteMessageHandler(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.invite_message.get")
	switch {
	case r.Method == "GET":
		resp := struct {
			Message string `json:"message"`
		}{"Check out gleepost! https://gleepost.com"}
		jsonResponse(w, resp, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}
