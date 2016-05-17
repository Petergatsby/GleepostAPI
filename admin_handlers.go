package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/Petergatsby/GleepostAPI/lib"
	"github.com/Petergatsby/GleepostAPI/lib/gp"
)

func init() {
	base.Handle("/admin/massmail", timeHandler(api, authenticated(mm))).Methods("POST")
	base.Handle("/admin/massmail", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/admin/masspush", timeHandler(api, authenticated(newVersionNotificationHandler))).Methods("POST")
	base.Handle("/admin/masspush", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/admin/prefill", timeHandler(api, authenticated(prefillNetwork))).Methods("POST")
	base.Handle("/admin/prefill", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/admin/prefill", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/admin/templates", timeHandler(api, authenticated(createTemplate))).Methods("POST")
	base.Handle("/admin/templates", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

//MissingParameterNetwork is the error you'll get if you don't give a network when you're manually creating a user.
//{"error":"Missing parameter: network"}
var MissingParameterNetwork = gp.APIerror{Reason: "Missing parameter: network"}

//MissingParameterNetwork is the error you'll get if you don't give a network when you're manually creating a user.
//{"error":"Missing parameter: network"}
var MissingParameterPost = gp.APIerror{Reason: "Missing parameter: post"}

func newVersionNotificationHandler(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	count, err := api.SendUpdateNotification(userID, r.FormValue("message"), r.FormValue("version"), r.FormValue("type"))
	switch {
	case err == lib.ENOTALLOWED:
		jsonResponse(w, err, 403)
	case err != nil:
		log.Println(err)
		jsonErr(w, err, 500)
	default:
		jsonResponse(w, count, 200)
	}
}

func mm(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	err := api.Massmail(userID)
	switch {
	case err == lib.ENOTALLOWED:
		jsonResponse(w, err, 403)
	case err != nil:
		jsonResponse(w, err, 500)
	default:
		w.WriteHeader(204)
	}
}

func postUsers(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	_netID, err := strconv.ParseUint(r.FormValue("network"), 10, 64)
	if err != nil {
		jsonResponse(w, MissingParameterNetwork, 400)
		return
	}
	netID := gp.NetworkID(_netID)
	verified, _ := strconv.ParseBool(r.FormValue("verified"))
	_, err = api.UserCreateUserSpecial(userID, r.FormValue("first"), r.FormValue("last"), r.FormValue("email"), r.FormValue("pass"), verified, netID)
	switch {
	case err == lib.ENOTALLOWED:
		jsonResponse(w, err, 403)
	case err != nil:
		jsonErr(w, err, 500)
	default:
		w.WriteHeader(204)
	}
}

func prefillNetwork(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	_netID, err := strconv.ParseUint(r.FormValue("network"), 10, 64)
	if err != nil {
		jsonResponse(w, MissingParameterNetwork, 400)
		return
	}
	netID := gp.NetworkID(_netID)
	name := r.FormValue("name")
	err = api.AdminPrefillUniversity(userID, netID, name)
	switch {
	case err == lib.ENOTALLOWED:
		jsonResponse(w, err, 403)
	case err != nil:
		jsonResponse(w, err, 500)
	default:
		w.WriteHeader(204)
	}
}

func createTemplate(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	_postID, err := strconv.ParseUint(r.FormValue("post"), 10, 64)
	if err != nil {
		jsonResponse(w, MissingParameterPost, 400)
		return
	}
	postID := gp.PostID(_postID)
	id, err := api.AdminCreateTemplateFromPost(userID, postID)
	switch {
	case err == lib.ENOTALLOWED:
		jsonResponse(w, err, 403)
	case err != nil:
		jsonResponse(w, err, 500)
	default:
		jsonResponse(w, id, 201)
	}
}
