package main

import (
	"net/http"
	"strconv"

	"github.com/Petergatsby/GleepostAPI/lib"
	"github.com/Petergatsby/GleepostAPI/lib/gp"
)

func init() {
	base.Handle("/reports", timeHandler(api, authenticated(postReports))).Methods("POST")
}

func postReports(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	_postID, _ := strconv.ParseUint(r.FormValue("post"), 10, 64)
	postID := gp.PostID(_postID)
	reason := r.FormValue("reason")
	err := api.ReportPost(userID, postID, reason)
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
