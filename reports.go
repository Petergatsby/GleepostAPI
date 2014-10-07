package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.HandleFunc("/reports", postReports).Methods("POST")
}

func postReports(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.reports.post")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_postID, _ := strconv.ParseUint(r.FormValue("post"), 10, 64)
		postID := gp.PostID(_postID)
		reason := r.FormValue("reason")
		err = api.ReportPost(userID, postID, reason)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
				return
			}
			jsonErr(w, err, 500)
			return
		}
		w.WriteHeader(204)
	}
}
