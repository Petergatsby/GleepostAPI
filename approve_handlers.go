package main

import (
	"net/http"
	"strconv"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.Handle("/approve/access", timeHandler(api, http.HandlerFunc(permissionHandler))).Methods("GET")
	base.Handle("/approve/level", timeHandler(api, http.HandlerFunc(getApproveSettings))).Methods("GET")
	base.Handle("/approve/level", timeHandler(api, http.HandlerFunc(postApproveSettings))).Methods("POST")
	base.Handle("/approve/pending", timeHandler(api, http.HandlerFunc(getApprovePending))).Methods("GET")
	base.Handle("/approve/approved", timeHandler(api, http.HandlerFunc(postApproveApproved))).Methods("POST")
	base.Handle("/approve/approved", timeHandler(api, http.HandlerFunc(getApproveApproved))).Methods("GET")
	base.Handle("/approve/rejected", timeHandler(api, http.HandlerFunc(postApproveRejected))).Methods("POST")
	base.Handle("/approve/rejected", timeHandler(api, http.HandlerFunc(getApproveRejected))).Methods("GET")
}

func permissionHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		nets, err := api.GetUserNetworks(userID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		access, err := api.ApproveAccess(userID, nets[0].ID)
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
		nets, err := api.GetUserNetworks(userID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		level, err := api.ApproveLevel(userID, nets[0].ID)
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
		nets, err := api.GetUserNetworks(userID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		_lev := r.FormValue("level")
		level, _ := strconv.Atoi(_lev)
		err = api.SetApproveLevel(userID, nets[0].ID, level)
		switch {
		case err == nil:
			level, err := api.ApproveLevel(userID, nets[0].ID)
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
		nets, err := api.GetUserNetworks(userID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		pending, err := api.GetNetworkPending(userID, nets[0].ID)
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
		nets, err := api.GetUserNetworks(userID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
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
		approved, err := api.GetNetworkApproved(userID, nets[0].ID, mode, index, api.Config.PostPageSize)
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
		nets, err := api.GetUserNetworks(userID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
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
		rejected, err := api.GetNetworkRejected(userID, nets[0].ID, mode, index, api.Config.PostPageSize)
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
