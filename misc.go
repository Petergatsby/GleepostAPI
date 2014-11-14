package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"

	"time"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

var (
	api    *lib.API
	config *conf.Config
)

func init() {
	base.HandleFunc("/invite_message", inviteMessageHandler)
	base.HandleFunc("/contact_form", contactFormHandler).Methods("POST")
	base.HandleFunc("/", optionsHandler).Methods("OPTIONS")
}

//EUNSUPPORTED = 405
var EUNSUPPORTED = gp.APIerror{Reason: "Method not supported"}

//ENOTFOUNT = 404
var ENOTFOUND = gp.APIerror{Reason: "404 not found"}

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
	config = conf.GetConfig()
	api = lib.New(*config)
	go api.FeedbackDaemon(60)
	go api.EndOldConversations()
	api.PeriodicSummary(time.Date(2014, time.April, 9, 8, 0, 0, 0, time.UTC), time.Duration(24*time.Hour))
	var futures []conf.PostFuture
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

func contactFormHandler(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.contact_form.post")
	err := api.ContactFormRequest(r.FormValue("name"), r.FormValue("college"), r.FormValue("email"), r.FormValue("phoneNo"))
	if err != nil {
		jsonErr(w, err, 500)
	}
	jsonResponse(w, struct {
		Success bool `json:"success"`
	}{Success: true}, 200)
}
