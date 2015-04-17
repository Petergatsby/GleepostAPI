package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"time"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

var (
	config = conf.GetConfig()
	api    = lib.New(*config)
)

func init() {
	base.Handle("/invite_message", timeHandler(api, http.HandlerFunc(inviteMessageHandler))).Methods("GET")
	base.Handle("/contact_form", timeHandler(api, http.HandlerFunc(contactFormHandler))).Methods("POST")
	base.Handle("/chasen", timeHandler(api, http.HandlerFunc(chasenHandler))).Methods("POST")
	base.Handle("/", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
}

//EUNSUPPORTED = 405
var EUNSUPPORTED = gp.APIerror{Reason: "Method not supported"}

//ENOTFOUNT = 404
var ENOTFOUND = gp.APIerror{Reason: "404 not found"}

func optionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Headers", "X-GP-Auth")
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
	resp := struct {
		Message string `json:"message"`
	}{"Check out gleepost! https://gleepost.com"}
	jsonResponse(w, resp, 200)
}

func contactFormHandler(w http.ResponseWriter, r *http.Request) {
	ip := r.Header.Get("X-Real-IP")
	err := api.ContactFormRequest(r.FormValue("name"), r.FormValue("college"), r.FormValue("email"), r.FormValue("phoneNo"), ip)
	if err != nil {
		jsonErr(w, err, 500)
	}
	jsonResponse(w, struct {
		Success bool `json:"success"`
	}{Success: true}, 200)
}

func chasenHandler(w http.ResponseWriter, r *http.Request) {
	err := api.ChasenRequest(r.FormValue("where"), r.FormValue("when"))
	if err != nil {
		jsonErr(w, err, 500)
	}
	jsonResponse(w, struct {
		Success bool `json:"success"`
	}{Success: true}, 200)
}

func goneHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, gp.APIerror{Reason: "All endpoints to do with Live conversations have been deprecated. Stop using them."}, 410)
}

func timeHandler(api *lib.API, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		metric := statsdMetricName(r)
		api.Time(start, metric)
	})
}

func authenticatedHandler(api *lib.API, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := authenticate(r)
		if err != nil {
			jsonResponse(w, &EBADTOKEN, 400)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func statsdMetricName(r *http.Request) string {
	metric := "gleepost." + strings.Replace(r.URL.Path, "/", ".", -1) + "." + strings.ToLower(r.Method)
	return metric
}

func unsupportedHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, &EUNSUPPORTED, 405)
}
