package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.HandleFunc("/views/posts", postPostViews).Methods("POST")
}

func postPostViews(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.views.posts.post")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		views := make([]gp.PostView, 0)
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