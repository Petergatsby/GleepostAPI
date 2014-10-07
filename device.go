package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func init() {
	base.HandleFunc("/devices/{id}", deleteDevice)
	base.HandleFunc("/devices/{id}/", deleteDevice)
	base.HandleFunc("/devices", postDevice)
}

func postDevice(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.devices.post")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		deviceType := r.FormValue("type")
		deviceID := r.FormValue("device_id")
		log.Println("Device:", deviceType, deviceID)
		device, err := api.AddDevice(userID, deviceType, deviceID)
		log.Println(device, err)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			jsonResponse(w, device, 201)
		}
	case r.Method == "GET":
		//implement getting tokens
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func deleteDevice(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.devices.delete")
	w.Header().Set("Content-Type", "application/json")
	log.Println("Delete device hit")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, EBADTOKEN, 400)
		log.Println("Bad auth")
	case r.Method == "DELETE":
		vars := mux.Vars(r)
		err := api.DeleteDevice(userID, vars["id"])
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		w.WriteHeader(204)
		return
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}

}
