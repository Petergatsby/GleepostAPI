package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

//EBADINPUT = You didn't supply a name for your account
var EBADINPUT = gp.APIerror{Reason: "Missing parameter: first / last"}

func init() {
	base.HandleFunc("/user/{id:[0-9]+}", getUser).Methods("GET")
	base.HandleFunc("/user/{id:[0-9]+}/", getUser).Methods("GET")
	base.HandleFunc("/user/{id:[0-9]+}/posts", getUserPosts).Methods("GET")
	base.HandleFunc("/user/{id:[0-9]+}/attending", getUserAttending).Methods("GET")
	base.HandleFunc("/user/{id:[0-9]+}/networks", getGroups).Methods("GET")
	base.HandleFunc("/user/{id:[0-9]+}/unread", unread)
	base.HandleFunc("/user/{id:[0-9]+}/total_live", totalLiveConversations)
	base.HandleFunc("/user/", postUsers)
	base.HandleFunc("/user", postUsers)
	//profile stuff
	base.HandleFunc("/profile/profile_image", profileImageHandler)
	base.HandleFunc("/profile/name", changeNameHandler)
	base.HandleFunc("/profile/tagline", postProfileTagline).Methods("POST")
	base.HandleFunc("/profile/change_pass", changePassHandler)
	base.HandleFunc("/profile/busy", busyHandler)
	base.HandleFunc("/profile/attending", userAttending)
	//notifications
	base.HandleFunc("/notifications", notificationHandler)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.users.*.get")
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
	defer api.Time(time.Now(), "gleepost.users.*.posts.get")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		otherID := gp.UserID(_id)
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
		if err != nil {
			start = 0
		}
		before, err := strconv.ParseInt(r.FormValue("before"), 10, 64)
		if err != nil {
			before = 0
		}
		after, err := strconv.ParseInt(r.FormValue("after"), 10, 64)
		if err != nil {
			after = 0
		}
		var index int64
		var mode int
		switch {
		case after > 0:
			mode = gp.OAFTER
			index = after
		case before > 0:
			mode = gp.OBEFORE
			index = before
		default:
			mode = gp.OSTART
			index = start
		}
		if err != nil {
			after = 0
		}
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
	defer api.Time(time.Now(), "gleepost.profile.name.post")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		firstName := r.FormValue("first")
		lastName := r.FormValue("last")
		err := api.SetUserName(userID, firstName, lastName)
		if err != nil {
			jsonResponse(w, &EBADINPUT, 400)
			return
		}
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func busyHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		defer api.Time(time.Now(), "gleepost.profile.busy.post")
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
		defer api.Time(time.Now(), "gleepost.profile.busy.get")
		status, err := api.BusyStatus(userID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, &gp.BusyStatus{Busy: status}, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func profileImageHandler(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.profile.profile_image.post")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		url := r.FormValue("url")
		exists, err := api.UserUploadExists(userID, url)
		if err != nil {
			jsonErr(w, err, 400)
			return
		}
		if !exists {
			jsonResponse(w, NoSuchUpload, 400)
		} else {
			err = api.SetProfileImage(userID, url)
			if err != nil {
				jsonErr(w, err, 500)
			} else {
				user, err := api.UserGetProfile(userID, userID)
				if err != nil {
					jsonErr(w, err, 500)
				}
				jsonResponse(w, user, 200)
			}
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func notificationHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "PUT":
		defer api.Time(time.Now(), "gleepost.notifications.put")
		_upTo, err := strconv.ParseUint(r.FormValue("seen"), 10, 64)
		if err != nil {
			_upTo = 0
		}
		includeSeen, _ := strconv.ParseBool(r.FormValue("include_seen"))
		notificationID := gp.NotificationID(_upTo)
		err = api.MarkNotificationsSeen(userID, notificationID)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			notifications, err := api.GetUserNotifications(userID, includeSeen)
			if err != nil {
				jsonErr(w, err, 500)
			} else {
				jsonResponse(w, notifications, 200)
			}
		}
	case r.Method == "GET":
		defer api.Time(time.Now(), "gleepost.notifications.get")
		includeSeen, _ := strconv.ParseBool(r.FormValue("include_seen"))
		notifications, err := api.GetUserNotifications(userID, includeSeen)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			jsonResponse(w, notifications, 200)
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func getUserAttending(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.user.*.attending.get")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		otherID := gp.UserID(_id)
		category := r.FormValue("filter")
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
		if err != nil {
			start = 0
		}
		before, err := strconv.ParseInt(r.FormValue("before"), 10, 64)
		if err != nil {
			before = 0
		}
		after, err := strconv.ParseInt(r.FormValue("after"), 10, 64)
		if err != nil {
			after = 0
		}
		var mode int
		var index int64
		switch {
		case after > 0:
			mode = gp.OAFTER
			index = after
		case before > 0:
			mode = gp.OBEFORE
			index = before
		default:
			mode = gp.OSTART
			index = start
		}
		events, err := api.UserEvents(userID, otherID, category, mode, index, 20)
		if err != nil {
			jsonResponse(w, err, 500)
		}
		jsonResponse(w, events, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func userAttending(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.profile.attending.get")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		events, err := api.UserAttends(userID)
		if err != nil {
			jsonResponse(w, err, 500)
		}
		if len(events) == 0 {
			// this is an ugly hack. But I can't immediately
			// think of a neater way to fix this
			// (json.Marshal(empty slice) returns null rather than
			// empty array ([]) which it obviously should
			jsonResponse(w, []string{}, 200)
			return
		}
		jsonResponse(w, events, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

/*

Utilities - undocumented.

*/

func unread(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.users.*.conversations.unread.get")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case userID != 2:
		jsonResponse(w, gp.APIerror{Reason: "Not allowed"}, 403)
	case r.Method == "GET":
		vars := mux.Vars(r)
		_uid, _ := strconv.ParseInt(vars["id"], 10, 64)
		uid := gp.UserID(_uid)
		count, err := api.UnreadMessageCount(uid)
		if err != nil {
			jsonResponse(w, err, 500)
		}
		jsonResponse(w, count, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func totalLiveConversations(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.users.*.conversations.live.get")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case userID != 2:
		jsonResponse(w, gp.APIerror{Reason: "Not allowed"}, 403)
	case r.Method == "GET":
		vars := mux.Vars(r)
		_uid, _ := strconv.ParseInt(vars["id"], 10, 64)
		uid := gp.UserID(_uid)
		count, err := api.TotalLiveConversations(uid)
		if err != nil {
			jsonResponse(w, err, 500)
		}
		jsonResponse(w, count, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func postProfileTagline(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.profile.tagline.post")
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
