package main

import (
	"net/http"
	"strconv"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.Handle("/notifications", timeHandler(api, http.HandlerFunc(notificationHandler))).Methods("PUT", "GET")
	base.Handle("/notifications", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/notifications", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func notificationHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "PUT":
		_upTo, err := strconv.ParseUint(r.FormValue("seen"), 10, 64)
		if err != nil {
			_upTo = 0
		}
		includeSeen, _ := strconv.ParseBool(r.FormValue("include_seen"))
		mode, index := interpretPagination(r.FormValue("start"), r.FormValue("before"), r.FormValue("after"))
		notificationID := gp.NotificationID(_upTo)
		err = api.MarkNotificationsSeen(userID, notificationID)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			notifications, err := api.GetUserNotifications(userID, mode, index, includeSeen)
			if err != nil {
				jsonErr(w, err, 500)
			} else {
				jsonResponse(w, notifications, 200)
			}
		}
	case r.Method == "GET":
		includeSeen, _ := strconv.ParseBool(r.FormValue("include_seen"))
		mode, index := interpretPagination(r.FormValue("start"), r.FormValue("before"), r.FormValue("after"))
		notifications, err := api.GetUserNotifications(userID, mode, index, includeSeen)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			jsonResponse(w, notifications, 200)
		}
	}
}
