package main

import (
	"net/http"
	"strconv"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.Handle("/approve/access", timeHandler(api, authenticated(permissionHandler))).Methods("GET")
	base.Handle("/approve/access", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/approve/level", timeHandler(api, authenticated(getApproveSettings))).Methods("GET")
	base.Handle("/approve/level", timeHandler(api, authenticated(postApproveSettings))).Methods("POST")
	base.Handle("/approve/level", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/approve/pending", timeHandler(api, authenticated(getApprovePending))).Methods("GET")
	base.Handle("/approve/pending", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/approve/approved", timeHandler(api, authenticated(postApproveApproved))).Methods("POST")
	base.Handle("/approve/approved", timeHandler(api, authenticated(getApproveApproved))).Methods("GET")
	base.Handle("/approve/approved", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/approve/rejected", timeHandler(api, authenticated(postApproveRejected))).Methods("POST")
	base.Handle("/approve/rejected", timeHandler(api, authenticated(getApproveRejected))).Methods("GET")
	base.Handle("/approve/rejected", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func permissionHandler(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	access, err := api.ApproveAccess(userID)
	if err != nil {
		jsonErr(w, err, 500)
		return
	}
	jsonResponse(w, access, 200)
}

func getApproveSettings(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	level, err := api.ApproveLevel(userID)
	if err != nil {
		jsonErr(w, err, 500)
		return
	}
	jsonResponse(w, level, 200)
}

func postApproveSettings(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	_lev := r.FormValue("level")
	level, _ := strconv.Atoi(_lev)
	err := api.SetApproveLevel(userID, level)
	switch {
	case err == nil:
		level, err := api.ApproveLevel(userID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, level, 200)
	case err == &lib.ENOTALLOWED:
		jsonErr(w, err, 403)
	default:
		jsonErr(w, err, 500)
	}
}

func getApprovePending(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	pending, err := api.UserGetPending(userID)
	switch {
	case err == nil:
		jsonResponse(w, pending, 200)
	case err == &lib.ENOTALLOWED:
		jsonErr(w, err, 403)
	default:
		jsonErr(w, err, 500)
	}
}

func postApproveApproved(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	_postID, _ := strconv.ParseUint(r.FormValue("post"), 10, 64)
	postID := gp.PostID(_postID)
	reason := r.FormValue("reason")
	err := api.ApprovePost(userID, postID, reason)
	switch {
	case err == nil:
		w.WriteHeader(204)
	case err == &lib.ENOTALLOWED:
		jsonErr(w, err, 403)
	default:
		jsonErr(w, err, 500)
	}
}

func getApproveApproved(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	mode, index := interpretPagination(r)
	approved, err := api.UserGetApproved(userID, mode, index, api.Config.PostPageSize)
	switch {
	case err == nil:
		jsonResponse(w, approved, 200)
	case err == &lib.ENOTALLOWED:
		jsonErr(w, err, 403)
	default:
		jsonErr(w, err, 500)
	}
}

func interpretPagination(r *http.Request) (mode int, index int64) {
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
	centre, err := strconv.ParseInt(r.FormValue("centre"), 10, 64)
	if err != nil {
		centre = 0
	}
	switch {
	case after > 0:
		mode = lib.ChronologicallyAfterID
		index = after
	case before > 0:
		mode = lib.ChronologicallyBeforeID
		index = before
	case centre > 0:
		mode = lib.CentredOnID
		index = centre
	default:
		mode = lib.ByOffsetDescending
		index = start
	}
	return
}

func postApproveRejected(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	_postID, _ := strconv.ParseUint(r.FormValue("post"), 10, 64)
	postID := gp.PostID(_postID)
	reason := r.FormValue("reason")
	err := api.RejectPost(userID, postID, reason)
	switch {
	case err == nil:
		w.WriteHeader(204)
	case err == &lib.ENOTALLOWED:
		jsonErr(w, err, 403)
	default:
		jsonErr(w, err, 500)
	}
}

func getApproveRejected(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	mode, index := interpretPagination(r)
	rejected, err := api.UserGetRejected(userID, mode, index, api.Config.PostPageSize)
	switch {
	case err == nil:
		jsonResponse(w, rejected, 200)
	case err == &lib.ENOTALLOWED:
		jsonErr(w, err, 403)
	default:
		jsonErr(w, err, 500)
	}
}
