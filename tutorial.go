package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Petergatsby/GleepostAPI/lib"
	"github.com/Petergatsby/GleepostAPI/lib/gp"
)

func init() {
	base.Handle("/profile/tutorial_state", timeHandler(api, authenticated(postProfileTutorialState))).Methods("POST")
	base.Handle("/profile/tutorial_state", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/profile/tutorial_state", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/campuspal_greet", timeHandler(api, authenticated(greetMe))).Methods("POST")
	base.Handle("/campuspal_greet", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/campuspal_greet", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func postProfileTutorialState(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	ts := strings.Split(r.FormValue("tutorial_state"), ",")
	err := api.SetTutorialState(userID, ts...)
	if err != nil {
		jsonErr(w, err, 500)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(204)
}

func greetMe(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	n, _ := strconv.ParseInt(r.FormValue("preset"), 10, 64)
	err := api.GreetMe(userID, int(n))
	switch {
	case err == lib.ErrInvalidPreset:
		jsonErr(w, err, 400)
	case err != nil:
		jsonErr(w, err, 500)
	default:
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(204)
	}
}
