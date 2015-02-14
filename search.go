package main

import (
	"net/http"
	"strings"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

func init() {
	base.Handle("/search/users/{query}", timeHandler(api, http.HandlerFunc(searchUsers))).Methods("GET")
	base.Handle("/search/groups/{query}", timeHandler(api, http.HandlerFunc(searchGroups))).Methods("GET")
}

func searchUsers(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		vars := mux.Vars(r)
		query := strings.Split(vars["query"], " ")
		for i := range query {
			query[i] = strings.TrimSpace(query[i])
		}
		users, err := api.UserSearchUsersInPrimaryNetwork(userID, query[0], strings.Join(query[1:], " "))
		if err != nil {
			e, ok := err.(*gp.APIerror)
			switch {
			case !ok:
				jsonErr(w, err, 500)
			case *e == lib.ENOTALLOWED:
				jsonResponse(w, e, 403)
			case *e == lib.ETOOSHORT:
				jsonResponse(w, e, 400)
			default:
				jsonErr(w, err, 500)
			}
			return
		}
		jsonResponse(w, users, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func searchGroups(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		vars := mux.Vars(r)
		query := vars["query"]
		groups, err := api.UserSearchGroups(userID, query)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, groups, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}
