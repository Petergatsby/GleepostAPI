package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

func init() {
	base.Handle("/stats/user/{id:[0-9]+}/posts/{type}/{period}/{start}/{finish}", timeHandler(api, http.HandlerFunc(postsStatsHandler))).Methods("GET")
	base.Handle("/stats/posts/{id:[0-9]+}/{type}/{period}/{start}/{finish}", timeHandler(api, http.HandlerFunc(individualPostStats))).Methods("GET")
}

func postsStatsHandler(w http.ResponseWriter, r *http.Request) {
	_, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "GET":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		vars := mux.Vars(r)
		var bucket time.Duration
		switch {
		case vars["period"] == "hour":
			bucket = time.Duration(time.Hour)
		case vars["period"] == "day":
			bucket = time.Duration(24 * time.Hour)
		case vars["period"] == "week":
			bucket = time.Duration(168 * time.Hour)
		default:
			bucket = time.Duration(24 * time.Hour)
		}
		start, err := time.Parse(time.RFC3339, vars["start"])
		if err != nil {
			log.Println("Error parsing start time:", err)
			log.Println("Defaulting to a year ago.")
			start = time.Now().UTC().AddDate(-1, 0, 0)
		}
		finish, err := time.Parse(time.RFC3339, vars["finish"])
		if err != nil {
			log.Println("Error parsing end time:", err)
			log.Println("Defaulting to now.")
			finish = time.Now().UTC()
		}
		if finish.Before(start) {
			finish = time.Now().UTC()
		}
		_other, err := strconv.ParseUint(vars["id"], 10, 64)
		if err != nil {
			jsonResponse(w, err, 404)
		}
		otherID := gp.UserID(_other)
		stat := lib.Stat(vars["type"])
		var stats *lib.View
		if stat == lib.OVERVIEW {
			stats, err = api.AggregateStatsForUser(otherID, start, finish, bucket)
		} else {
			stats, err = api.AggregateStatsForUser(otherID, start, finish, bucket, stat)
		}
		if err != nil {
			jsonResponse(w, err, 500)
			return
		}
		jsonResponse(w, stats, 200)
	}
}

func individualPostStats(w http.ResponseWriter, r *http.Request) {
	_, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "GET":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		vars := mux.Vars(r)
		var bucket time.Duration
		switch {
		case vars["period"] == "hour":
			bucket = time.Duration(time.Hour)
		case vars["period"] == "day":
			bucket = time.Duration(24 * time.Hour)
		case vars["period"] == "week":
			bucket = time.Duration(168 * time.Hour)
		default:
			bucket = time.Duration(24 * time.Hour)
		}
		start, err := time.Parse(time.RFC3339, vars["start"])
		if err != nil {
			log.Println("Error parsing start time:", err)
			log.Println("Defaulting to a year ago.")
			start = time.Now().UTC().AddDate(-1, 0, 0)
		}
		finish, err := time.Parse(time.RFC3339, vars["finish"])
		if err != nil {
			log.Println("Error parsing end time:", err)
			log.Println("Defaulting to now.")
			finish = time.Now().UTC()
		}
		if finish.Before(start) {
			finish = time.Now().UTC()
		}
		_post, err := strconv.ParseUint(vars["id"], 10, 64)
		if err != nil {
			jsonResponse(w, err, 404)
		}
		postID := gp.PostID(_post)
		stat := lib.Stat(vars["type"])
		var stats *lib.View
		if stat == lib.OVERVIEW {
			stats, err = api.AggregateStatsForPost(postID, start, finish, bucket)
		} else {
			stats, err = api.AggregateStatsForPost(postID, start, finish, bucket, stat)
		}
		if err != nil {
			jsonResponse(w, err, 500)
			return
		}
		jsonResponse(w, stats, 200)
	}
}
