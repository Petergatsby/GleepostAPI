package main

import (
	"fmt"
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
		go api.Count(1, "gleepost.devices.post.400")
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
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
	case r.Method == "GET":
		//implement getting tokens
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func deleteDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.devices.%s.delete", vars["id"])
	defer api.Time(time.Now(), url)
	w.Header().Set("Content-Type", "application/json")
	log.Println("Delete device hit")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, EBADTOKEN, 400)
	case r.Method == "DELETE":
		err := api.DeleteDevice(userID, vars["id"])
		if err != nil {
			go api.Count(1, url+".500")
			jsonErr(w, err, 500)
			return
		}
		go api.Count(1, url+".204")
		w.WriteHeader(204)
		return
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}

}
