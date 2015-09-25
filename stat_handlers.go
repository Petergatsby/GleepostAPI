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
	base.Handle("/stats/user/{id:[0-9]+}/posts/{type}/{period}/{start}/{finish}", timeHandler(api, authenticated(postsStatsHandler))).Methods("GET")
	base.Handle("/stats/user/{id:[0-9]+}/posts/{type}/{period}/{start}/{finish}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/stats/posts/{id:[0-9]+}/{type}/{period}/{start}/{finish}", timeHandler(api, authenticated(individualPostStats))).Methods("GET")
	base.Handle("/stats/posts/{id:[0-9]+}/{type}/{period}/{start}/{finish}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/stats/network/{id:[0-9]+}/users_online/{start}/{finish}", timeHandler(api, authenticated(usersOnlineHandler))).Methods("GET")
}

func postsStatsHandler(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := getBucketLength(vars["period"])
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

func getBucketLength(period string) (bucket time.Duration) {
	switch {
	case period == "hour":
		bucket = time.Duration(time.Hour)
	case period == "day":
		bucket = time.Duration(24 * time.Hour)
	case period == "week":
		bucket = time.Duration(168 * time.Hour)
	default:
		bucket = time.Duration(24 * time.Hour)
	}
	return bucket
}

func individualPostStats(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
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

type onlineStats struct {
	Total    int `json:"total"`
	Students int `json:"students"`
	Staff    int `json:"staff"`
	Faculty  int `json:"faculty"`
	Alumni   int `json:"alumni"`
}

func usersOnlineHandler(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
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
	_network, err := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		jsonResponse(w, err, 404)
	}
	netID := gp.NetworkID(_network)
	total, students, staff, faculty, alumni, err := api.UsersOnline(netID, start, finish)
	if err != nil {
		jsonResponse(w, err, 500)
		return
	}
	jsonResponse(w, onlineStats{Total: total, Students: students, Staff: staff, Faculty: faculty, Alumni: alumni}, 200)
}
