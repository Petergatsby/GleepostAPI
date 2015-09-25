package main

import (
	"net/http"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

func init() {
	base.Handle("/search/users/{query}", timeHandler(api, authenticated(searchUsers))).Methods("GET")
	base.Handle("/search/users/{query}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/search/groups/{query}", timeHandler(api, authenticated(searchGroups))).Methods("GET")
	base.Handle("/search/groups/{query}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func searchUsers(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	query := vars["query"]
	users, err := api.UserSearchUsersInPrimaryNetwork(userID, query)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		switch {
		case !ok:
			jsonErr(w, err, 500)
		case *e == lib.ENOTALLOWED:
			jsonResponse(w, e, 403)
		default:
			jsonErr(w, err, 500)
		}
		return
	}
	jsonResponse(w, users, 200)
}

func searchGroups(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	query := vars["query"]
	filter := r.FormValue("filter")
	groups, err := api.UserSearchGroups(userID, query, filter)
	if err != nil {
		jsonErr(w, err, 500)
		return
	}
	jsonResponse(w, groups, 200)
}
