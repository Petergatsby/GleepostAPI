package main

import (
	"net/http"
	"time"
)

func init() {
	base.HandleFunc("/approve/access", permissionHandler).Methods("GET")
}

type permission struct {
	ApproveAccess bool `json:"access"`
	LevelChange   bool `json:"settings"`
}

func permissionHandler(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "approve.access.get")
	_, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		jsonResponse(w, permission{ApproveAccess: true, LevelChange: false}, 200)
	}
}
