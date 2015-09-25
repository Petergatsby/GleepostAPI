package main

import (
	"encoding/json"
	"net/http"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.Handle("/views/posts", timeHandler(api, authenticated(postPostViews))).Methods("POST")
	base.Handle("/views/posts", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/views/posts", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func postPostViews(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	var views []gp.PostView
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&views)
	if err != nil {
		jsonErr(w, err, 400)
	}
	for i := range views {
		views[i].User = userID
	}
	go api.Viewer.RecordViews(views)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(204)
}
