package main

import (
	"net/http"
	"time"
)

func init() {
	base.HandleFunc("/approve/access", permissionHandler).Methods("GET")
	base.HandleFunc("/approve/level", getApproveSettings).Methods("GET")
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
