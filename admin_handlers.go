package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

//MissingParameterNetwork is the error you'll get if you don't give a network when you're manually creating a user.
//{"error":"Missing parameter: network"}
var MissingParameterNetwork = gp.APIerror{Reason: "Missing parameter: network"}

func newVersionNotificationHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		if api.IsAdmin(userID) {
			count, err := api.MassNotification(r.FormValue("message"), r.FormValue("version"), r.FormValue("type"))
			if err != nil {
				log.Println(err)
				jsonResponse(w, err, 500)
			} else {
				jsonResponse(w, count, 200)
			}
		} else {
			jsonResponse(w, &lib.ENOTALLOWED, 403)
		}
	}
}

func mm(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		if api.IsAdmin(userID) {
			err := api.Massmail()
			if err != nil {
				jsonResponse(w, err, 500)
			} else {
				w.WriteHeader(204)
			}

		} else {
			jsonResponse(w, &lib.ENOTALLOWED, 403)
		}
	}
	jsonResponse(w, err, 200)
}

func postUsers(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		if api.IsAdmin(userID) {
			_netID, err := strconv.ParseUint(r.FormValue("network"), 10, 64)
			if err != nil {
				jsonResponse(w, MissingParameterNetwork, 400)
				return
			}
			netID := gp.NetworkID(_netID)
			verified, _ := strconv.ParseBool(r.FormValue("verified"))
			err = api.CreateUserSpecial(r.FormValue("first"), r.FormValue("last"), r.FormValue("email"), r.FormValue("pass"), verified, netID)
			if err != nil {
				jsonResponse(w, err, 500)
				return
			}
			w.WriteHeader(204)
		} else {
			jsonResponse(w, &lib.ENOTALLOWED, 403)
		}
	}
}
