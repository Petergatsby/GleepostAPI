package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.Handle("/admin/massmail", timeHandler(api, http.HandlerFunc(mm))).Methods("POST")
	base.Handle("/admin/massmail", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/admin/masspush", timeHandler(api, http.HandlerFunc(newVersionNotificationHandler))).Methods("POST")
	base.Handle("/admin/masspush", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/admin/posts/duplicate", timeHandler(api, http.HandlerFunc(postDuplicate))).Methods("POST")
	base.Handle("/admin/posts/duplicate", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/admin/prefill", timeHandler(api, http.HandlerFunc(prefillNetwork))).Methods("POST")
	base.Handle("/admin/prefill", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

//MissingParameterNetwork is the error you'll get if you don't give a network when you're manually creating a user.
//{"error":"Missing parameter: network"}
var MissingParameterNetwork = gp.APIerror{Reason: "Missing parameter: network"}

func newVersionNotificationHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
}

func mm(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		err = api.Massmail(userID)
		switch {
		case err == lib.ENOTALLOWED:
			jsonResponse(w, err, 403)
		case err != nil:
			jsonResponse(w, err, 500)
		default:
			w.WriteHeader(204)
		}
	}
	jsonResponse(w, err, 200)
}

func postUsers(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
}

func postDuplicate(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, err := strconv.ParseUint(r.FormValue("network"), 10, 64)
		if err != nil {
			jsonResponse(w, MissingParameterNetwork, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		posts := strings.Split(r.FormValue("posts"), ",")
		var postIDs []gp.PostID
		for _, p := range posts {
			_postID, err := strconv.ParseUint(p, 10, 64)
			if err == nil {
				postID := gp.PostID(_postID)
				postIDs = append(postIDs, postID)
			}
		}
		regExp := r.FormValue("regexp")
		replacement := r.FormValue("replacement")
		dupes, err := api.UserDuplicatePosts(userID, netID, true, regExp, replacement, postIDs...)
		switch {
		case err == lib.ENOTALLOWED:
			jsonResponse(w, err, 403)
		case err != nil:
			jsonResponse(w, err, 500)
		default:
			jsonResponse(w, dupes, 201)
		}
	}
}

func prefillNetwork(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
}
