package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func init() {
	base.Handle("/devices/{id}", timeHandler(api, http.HandlerFunc(deleteDevice))).Methods("DELETE")
	base.Handle("/devices/{id}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/devices", timeHandler(api, http.HandlerFunc(postDevice))).Methods("POST")
	base.Handle("/devices", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func postDevice(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, "gleepost.devices.post.400")
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
			go api.Count(1, "gleepost.devices.post.500")
			jsonErr(w, err, 500)
		} else {
			go api.Count(1, "gleepost.devices.post.201")
			jsonResponse(w, device, 201)
		}
	}
}

func deleteDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.devices.%s.delete", vars["id"])
	w.Header().Set("Content-Type", "application/json")
	log.Println("Delete device hit")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, EBADTOKEN, 400)
	default:
		err := api.DeleteDevice(userID, vars["id"])
		if err != nil {
			go api.Count(1, url+".500")
			jsonErr(w, err, 500)
			return
		}
		go api.Count(1, url+".204")
		w.WriteHeader(204)
		return
	}

}
