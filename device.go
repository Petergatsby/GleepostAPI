package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

func init() {
	base.Handle("/devices/{id}", timeHandler(api, authenticated(deleteDevice))).Methods("DELETE")
	base.Handle("/devices/{id}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/devices", timeHandler(api, authenticated(postDevice))).Methods("POST")
	base.Handle("/devices", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func postDevice(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	deviceType := r.FormValue("type")
	deviceID := r.FormValue("device_id")
	application := r.FormValue("application")
	if application == "" {
		application = "gleepost"
	}
	log.Println("Device:", deviceType, deviceID)
	device, err := api.AddDevice(userID, deviceType, deviceID, application)
	log.Println(device, err)
	if err != nil {
		go api.Statsd.Count(1, "gleepost.devices.post.500")
		jsonErr(w, err, 500)
	} else {
		go api.Statsd.Count(1, "gleepost.devices.post.201")
		jsonResponse(w, device, 201)
	}
}

func deleteDevice(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.devices.%s.delete", vars["id"])
	w.Header().Set("Content-Type", "application/json")
	log.Println("Delete device hit")
	err := api.DeleteDevice(userID, vars["id"])
	if err != nil {
		go api.Statsd.Count(1, url+".500")
		jsonErr(w, err, 500)
		return
	}
	go api.Statsd.Count(1, url+".204")
	w.WriteHeader(204)
	return
}
