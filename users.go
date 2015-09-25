package main

import (
	"net/http"
	"strconv"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

//EBADINPUT = You didn't supply a name for your account
var EBADINPUT = gp.APIerror{Reason: "Missing parameter: first / last"}

func init() {
	base.Handle("/user/{id:[0-9]+}", timeHandler(api, authenticated(getUser))).Methods("GET")
	base.Handle("/user/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/user/{id:[0-9]+}/posts", timeHandler(api, authenticated(getUserPosts))).Methods("GET")
	base.Handle("/user/{id:[0-9]+}/posts", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/user/{id:[0-9]+}/attending", timeHandler(api, authenticated(getUserAttending))).Methods("GET")
	base.Handle("/user/{id:[0-9]+}/attending", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/user/{id:[0-9]+}/networks", timeHandler(api, authenticated(getGroups))).Methods("GET")
	base.Handle("/user/{id:[0-9]+}/networks", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/user", timeHandler(api, authenticated(postUsers))).Methods("POST")
	base.Handle("/user", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	//profile stuff
	base.Handle("/profile/profile_image", timeHandler(api, authenticated(profileImageHandler))).Methods("POST")
	base.Handle("/profile/profile_image", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/name", timeHandler(api, authenticated(changeNameHandler))).Methods("POST")
	base.Handle("/profile/name", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/tagline", timeHandler(api, authenticated(postProfileTagline))).Methods("POST")
	base.Handle("/profile/tagline", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/change_pass", timeHandler(api, authenticated(changePassHandler))).Methods("POST")
	base.Handle("/profile/change_pass", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/busy", timeHandler(api, authenticated(busyHandler))).Methods("POST", "GET")
	base.Handle("/profile/busy", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/attending", timeHandler(api, authenticated(userAttending))).Methods("GET")
	base.Handle("/profile/attending", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	//Approval
	base.Handle("/profile/pending", timeHandler(api, authenticated(pendingPosts))).Methods("GET")
	base.Handle("/profile/pending", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func getUser(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_otherID, _ := strconv.ParseUint(vars["id"], 10, 64)
	otherID := gp.UserID(_otherID)
	user, err := api.UserGetProfile(userID, otherID)
	if err != nil {
		switch {
		case err == gp.ENOSUCHUSER:
			jsonErr(w, err, 404)
		case err == lib.ENOTALLOWED:
			jsonErr(w, err, 403)
		default:
			jsonErr(w, err, 500)
		}
	} else {
		jsonResponse(w, user, 200)
	}
}

func getUserPosts(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	otherID := gp.UserID(_id)
	mode, index := interpretPagination(r.FormValue("start"), r.FormValue("before"), r.FormValue("after"))
	posts, err := api.GetUserPosts(otherID, userID, mode, index, api.Config.PostPageSize, r.FormValue("filter"))
	if err != nil {
		jsonErr(w, err, 500)
		return
	}
	jsonResponse(w, posts, 200)
}

/*

Profile stuff

*/

func changeNameHandler(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	firstName := r.FormValue("first")
	lastName := r.FormValue("last")
	err := api.UserSetName(userID, firstName, lastName)
	if err != nil {
		jsonResponse(w, &EBADINPUT, 400)
		return
	}
	w.WriteHeader(204)
}

func busyHandler(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "POST":
		status, err := strconv.ParseBool(r.FormValue("status"))
		if err != nil {
			jsonResponse(w, gp.APIerror{Reason: "Bad input"}, 400)
		}
		err = api.SetBusyStatus(userID, status)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			jsonResponse(w, &gp.BusyStatus{Busy: status}, 200)
		}
	case r.Method == "GET":
		status, err := api.BusyStatus(userID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, &gp.BusyStatus{Busy: status}, 200)
	}
}

func profileImageHandler(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	err := api.UserSetProfileImage(userID, url)
	switch {
	case err == lib.NoSuchUpload:
		jsonResponse(w, err, 400)
	case err != nil:
		jsonErr(w, err, 500)
	default:
		user, err := api.UserGetProfile(userID, userID)
		if err != nil {
			jsonErr(w, err, 500)
		}
		jsonResponse(w, user, 200)
	}
}

func getUserAttending(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	otherID := gp.UserID(_id)
	category := r.FormValue("filter")
	mode, index := interpretPagination(r.FormValue("start"), r.FormValue("before"), r.FormValue("after"))
	events, err := api.UserEvents(userID, otherID, category, mode, index, 20)
	if err != nil {
		jsonResponse(w, err, 500)
		return
	}
	jsonResponse(w, events, 200)
}

func userAttending(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	events, err := api.UserAttends(userID)
	if err != nil {
		jsonResponse(w, err, 500)
	}
	jsonResponse(w, events, 200)
}

/*

Utilities - undocumented.

*/

func postProfileTagline(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	err := api.UserChangeTagline(userID, r.FormValue("tagline"))
	if err != nil {
		jsonErr(w, err, 500)
		return
	}
	w.WriteHeader(204)
}

func pendingPosts(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	pending, err := api.PendingPosts(userID)
	if err != nil {
		jsonErr(w, err, 500)
		return
	}
	jsonResponse(w, pending, 200)
}
