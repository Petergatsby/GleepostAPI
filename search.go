package main

import (
	"net/http"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

func init() {
	base.Handle("/search/users/{query}", timeHandler(api, http.HandlerFunc(searchUsers))).Methods("GET")
	base.Handle("/search/users/{query}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/search/groups/{query}", timeHandler(api, http.HandlerFunc(searchGroups))).Methods("GET")
	base.Handle("/search/groups/{query}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func searchUsers(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
}

func searchGroups(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		query := vars["query"]
		groups, err := api.UserSearchGroups(userID, query)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, groups, 200)
	}
}
