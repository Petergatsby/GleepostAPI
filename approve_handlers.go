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
