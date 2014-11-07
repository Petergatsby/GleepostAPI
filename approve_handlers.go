package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.HandleFunc("/approve/access", permissionHandler).Methods("GET")
	base.HandleFunc("/approve/level", getApproveSettings).Methods("GET")
	base.HandleFunc("/approve/level", postApproveSettings).Methods("POST")
	base.HandleFunc("/approve/pending", getApprovePending).Methods("GET")
	base.HandleFunc("/approve/approved", postApproveApproved).Methods("POST")
	base.HandleFunc("/approve/approved", getApproveApproved).Methods("GET")
	base.HandleFunc("/approve/rejected", postApproveRejected).Methods("POST")
}

func permissionHandler(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "approve.access.get")
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
	defer api.Time(time.Now(), "approve.level.get")
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
	defer api.Time(time.Now(), "approve.level.post")
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
	defer api.Time(time.Now(), "approve.pending.get")
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
	defer api.Time(time.Now(), "approve.approved.post")
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
	defer api.Time(time.Now(), "approve.approved.get")
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
		approved, err := api.GetNetworkApproved(userID, nets[0].ID)
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
	defer api.Time(time.Now(), "approve.rejected.post")
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
	defer api.Time(time.Now(), "approve.rejected.get")
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
		rejected, err := api.GetNetworkRejected(userID, nets[0].ID)
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
