package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.HandleFunc("/admin/massmail", mm).Methods("POST")
	base.HandleFunc("/admin/masspush", newVersionNotificationHandler).Methods("POST")
	base.HandleFunc("/admin/posts/duplicate", postDuplicate).Methods("POST")
	base.HandleFunc("/admin/posts/copy_attribs", copyAttribs).Methods("POST")
}

//MissingParameterNetwork is the error you'll get if you don't give a network when you're manually creating a user.
//{"error":"Missing parameter: network"}
var MissingParameterNetwork = gp.APIerror{Reason: "Missing parameter: network"}

func newVersionNotificationHandler(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "admin.masspush")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		if api.IsAdmin(userID) {
			count, err := api.MassNotification(r.FormValue("message"), r.FormValue("version"), r.FormValue("type"))
			if err != nil {
				log.Println(err)
				jsonResponse(w, err, 500)
			} else {
				jsonResponse(w, count, 200)
			}
		} else {
			jsonResponse(w, &lib.ENOTALLOWED, 403)
		}
	}
}

func mm(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "admin.massmail")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		if api.IsAdmin(userID) {
			err := api.Massmail()
			if err != nil {
				jsonResponse(w, err, 500)
			} else {
				w.WriteHeader(204)
			}

		} else {
			jsonResponse(w, &lib.ENOTALLOWED, 403)
		}
	}
	jsonResponse(w, err, 200)
}

func postUsers(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "users.post")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		if api.IsAdmin(userID) {
			_netID, err := strconv.ParseUint(r.FormValue("network"), 10, 64)
			if err != nil {
				jsonResponse(w, MissingParameterNetwork, 400)
				return
			}
			netID := gp.NetworkID(_netID)
			verified, _ := strconv.ParseBool(r.FormValue("verified"))
			_, err = api.CreateUserSpecial(r.FormValue("first"), r.FormValue("last"), r.FormValue("email"), r.FormValue("pass"), verified, netID)
			if err != nil {
				jsonResponse(w, err, 500)
				return
			}
			w.WriteHeader(204)
		} else {
			jsonResponse(w, &lib.ENOTALLOWED, 403)
		}
	}
}

func postDuplicate(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "admin.posts.duplicate")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		if api.IsAdmin(userID) {
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
			dupes, err := api.DuplicatePosts(netID, true, postIDs...)
			if err != nil {
				jsonResponse(w, err, 500)
				return
			}
			jsonResponse(w, dupes, 201)
		} else {
			jsonResponse(w, &lib.ENOTALLOWED, 403)
		}
	}
}

func copyAttribs(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "admin.posts.copy_attribs")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		if api.IsAdmin(userID) {
			from := strings.Split(r.FormValue("from"), ",")
			var fromIDs []gp.PostID
			for _, p := range from {
				_postID, err := strconv.ParseUint(p, 10, 64)
				if err == nil {
					postID := gp.PostID(_postID)
					fromIDs = append(fromIDs, postID)
				}
			}
			to := strings.Split(r.FormValue("to"), ",")
			var toIDs []gp.PostID
			for _, p := range to {
				_postID, err := strconv.ParseUint(p, 10, 64)
				if err == nil {
					postID := gp.PostID(_postID)
					toIDs = append(toIDs, postID)
				}
			}
			err := api.MultiCopyPostAttribs(fromIDs, toIDs)
			if err != nil {
				jsonResponse(w, err, 500)
				return
			}
			w.WriteHeader(204)
		} else {
			jsonResponse(w, &lib.ENOTALLOWED, 403)
		}
	}
}
