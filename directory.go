package main

import (
	"net/http"
	"strconv"

	"github.com/Petergatsby/GleepostAPI/lib/dir/stanford"
	"github.com/Petergatsby/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

func init() {
	base.Handle("/directory/{university}/{query}", timeHandler(api, authenticated(searchDirectory))).Methods("GET")
	base.Handle("/directory/{university}/{query}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

//NotYetImplemented means this university's directory search doesn't exist yet.
var NotYetImplemented = gp.APIerror{Reason: "This directory is not yet searchable"}

func searchDirectory(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if vars["university"] != "stanford" {
		jsonResponse(w, NotYetImplemented, 404)
		return
	}
	cached, _ := strconv.ParseBool(r.FormValue("cache"))
	dir := stanford.Dir{ElasticSearch: api.Config.ElasticSearch}
	var results []stanford.Member
	var err error
	if cached {
		results, err = dir.CacheQuery(vars["query"], stanford.Everyone)
	} else {
		results, err = dir.Query(vars["query"], stanford.Everyone)
	}
	if err != nil {
		jsonResponse(w, err, 502)
		return
	}
	jsonResponse(w, results, 200)
}
