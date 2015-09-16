package main

import (
	"net/http"

	"github.com/draaglom/GleepostAPI/lib/dir/stanford"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

func init() {
	base.Handle("/directory/{university}/{query}", timeHandler(api, http.HandlerFunc(searchDirectory))).Methods("GET")
	base.Handle("/directory/{university}/{query}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

var NotYetImplemented = gp.APIerror{Reason: "This directory is not yet searchable"}

func searchDirectory(w http.ResponseWriter, r *http.Request) {
	_, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		if vars["university"] != "stanford" {
			jsonResponse(w, NotYetImplemented, 404)
			return
		}
		dir := stanford.Dir{}
		results, err := dir.Query(vars["query"])
		if err != nil {
			jsonResponse(w, err, 502)
			return
		}
		jsonResponse(w, results, 200)
	}
}
