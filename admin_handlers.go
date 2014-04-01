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
			api.MassNotification(r.FormValue("message"), r.FormValue("version"), r.FormValue("type"))
		} else {
			jsonResponse(w, &lib.ENOTALLOWED, 403)
		}
	}
}
