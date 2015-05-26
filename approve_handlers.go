package main

import (
	"net/http"
	"strconv"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.Handle("/approve/access", timeHandler(api, http.HandlerFunc(permissionHandler))).Methods("GET")
	base.Handle("/approve/access", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/approve/level", timeHandler(api, http.HandlerFunc(getApproveSettings))).Methods("GET")
	base.Handle("/approve/level", timeHandler(api, http.HandlerFunc(postApproveSettings))).Methods("POST")
	base.Handle("/approve/level", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/approve/pending", timeHandler(api, http.HandlerFunc(getApprovePending))).Methods("GET")
	base.Handle("/approve/pending", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/approve/approved", timeHandler(api, http.HandlerFunc(postApproveApproved))).Methods("POST")
	base.Handle("/approve/approved", timeHandler(api, http.HandlerFunc(getApproveApproved))).Methods("GET")
	base.Handle("/approve/approved", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/approve/rejected", timeHandler(api, http.HandlerFunc(postApproveRejected))).Methods("POST")
	base.Handle("/approve/rejected", timeHandler(api, http.HandlerFunc(getApproveRejected))).Methods("GET")
	base.Handle("/approve/rejected", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func permissionHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		access, err := api.ApproveAccess(userID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, access, 200)
	}
}

func getApproveSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		level, err := api.ApproveLevel(userID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, level, 200)
	}
}

func postApproveSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_lev := r.FormValue("level")
		level, _ := strconv.Atoi(_lev)
		err = api.SetApproveLevel(userID, level)
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
}

func getApprovePending(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
}

func postApproveApproved(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_postID, _ := strconv.ParseUint(r.FormValue("post"), 10, 64)
		postID := gp.PostID(_postID)
		reason := r.FormValue("reason")
		err = api.ApprovePost(userID, postID, reason)
		switch {
		case err == nil:
			w.WriteHeader(204)
		case err == &lib.ENOTALLOWED:
			jsonErr(w, err, 403)
		default:
			jsonErr(w, err, 500)
		}
	}
}

func getApproveApproved(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		mode, index := interpretPagination(r.FormValue("start"), r.FormValue("before"), r.FormValue("after"))
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
}

func interpretPagination(startString, beforeString, afterString string) (mode int, index int64) {
	start, err := strconv.ParseInt(startString, 10, 64)
	if err != nil {
		start = 0
	}
	before, err := strconv.ParseInt(beforeString, 10, 64)
	if err != nil {
		before = 0
	}
	after, err := strconv.ParseInt(afterString, 10, 64)
	if err != nil {
		after = 0
	}
	switch {
	case after > 0:
		mode = lib.ChronologicallyAfterID
		index = after
	case before > 0:
		mode = lib.ChronologicallyBeforeID
		index = before
	default:
		mode = lib.ByOffsetDescending
		index = start
	}
	return
}

func postApproveRejected(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_postID, _ := strconv.ParseUint(r.FormValue("post"), 10, 64)
		postID := gp.PostID(_postID)
		reason := r.FormValue("reason")
		err = api.RejectPost(userID, postID, reason)
		switch {
		case err == nil:
			w.WriteHeader(204)
		case err == &lib.ENOTALLOWED:
			jsonErr(w, err, 403)
		default:
			jsonErr(w, err, 500)
		}
	}
}

func getApproveRejected(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		mode, index := interpretPagination(r.FormValue("start"), r.FormValue("before"), r.FormValue("after"))
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
}
