package main

import (
	"github.com/draaglom/GleepostAPI/lib"
	"net/http"
)

func newVersionNotificationHandler(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		if api.IsAdmin(userId) {
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
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		if api.IsAdmin(userId) {
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

