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
	base.Handle("/user/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(getUser))).Methods("GET")
	base.Handle("/user/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/user/{id:[0-9]+}/posts", timeHandler(api, http.HandlerFunc(getUserPosts))).Methods("GET")
	base.Handle("/user/{id:[0-9]+}/posts", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/user/{id:[0-9]+}/attending", timeHandler(api, http.HandlerFunc(getUserAttending))).Methods("GET")
	base.Handle("/user/{id:[0-9]+}/attending", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/user/{id:[0-9]+}/networks", timeHandler(api, http.HandlerFunc(getGroups))).Methods("GET")
	base.Handle("/user/{id:[0-9]+}/networks", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/user", timeHandler(api, http.HandlerFunc(postUsers))).Methods("POST")
	base.Handle("/user", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	//profile stuff
	base.Handle("/profile/profile_image", timeHandler(api, http.HandlerFunc(profileImageHandler))).Methods("POST")
	base.Handle("/profile/profile_image", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/name", timeHandler(api, http.HandlerFunc(changeNameHandler))).Methods("POST")
	base.Handle("/profile/name", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/tagline", timeHandler(api, http.HandlerFunc(postProfileTagline))).Methods("POST")
	base.Handle("/profile/tagline", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/change_pass", timeHandler(api, http.HandlerFunc(changePassHandler))).Methods("POST")
	base.Handle("/profile/change_pass", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/busy", timeHandler(api, http.HandlerFunc(busyHandler))).Methods("POST", "GET")
	base.Handle("/profile/busy", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/attending", timeHandler(api, http.HandlerFunc(userAttending))).Methods("GET")
	base.Handle("/profile/attending", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	//Approval
	base.Handle("/profile/pending", timeHandler(api, http.HandlerFunc(pendingPosts))).Methods("GET")
	base.Handle("/profile/pending", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func getUser(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
}

func getUserPosts(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
}

/*

Profile stuff

*/

func changeNameHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		firstName := r.FormValue("first")
		lastName := r.FormValue("last")
		err := api.UserSetName(userID, firstName, lastName)
		if err != nil {
			jsonResponse(w, &EBADINPUT, 400)
			return
		}
		w.WriteHeader(204)
	}
}

func busyHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
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

func profileImageHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		url := r.FormValue("url")
		err = api.UserSetProfileImage(userID, url)
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
}

func getUserAttending(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
}

func userAttending(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		events, err := api.UserAttends(userID)
		if err != nil {
			jsonResponse(w, err, 500)
		}
		jsonResponse(w, events, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

/*

Utilities - undocumented.

*/

func postProfileTagline(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		err = api.UserChangeTagline(userID, r.FormValue("tagline"))
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		w.WriteHeader(204)
	}
}

func pendingPosts(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		pending, err := api.PendingPosts(userID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, pending, 200)
	}
}
