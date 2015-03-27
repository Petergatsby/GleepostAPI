package main

import (
	"encoding/json"
	"net/http"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.Handle("/views/posts", timeHandler(api, http.HandlerFunc(postPostViews))).Methods("POST")
	base.Handle("/views/posts", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/views/posts", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func postPostViews(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		var views []gp.PostView
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&views)
		if err != nil {
			jsonErr(w, err, 400)
		}
		for i := range views {
			views[i].User = userID
		}
		go api.RecordViews(views...)
		w.WriteHeader(204)
	}
}
