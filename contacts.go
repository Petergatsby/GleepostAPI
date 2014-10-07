package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

func init() {
	base.HandleFunc("/contacts", contactsHandler)
	base.HandleFunc("/contacts/{id:[0-9]+}", contactHandler)
	base.HandleFunc("/contacts/{id:[0-9]+}/", contactHandler)
}

func contactsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		defer api.Time(time.Now(), "gleepost.contacts.get")
		contacts, err := api.GetContacts(userID)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			if len(contacts) == 0 {
				jsonResponse(w, []string{}, 200)
			} else {
				jsonResponse(w, contacts, 200)
			}
		}
	case r.Method == "POST":
		defer api.Time(time.Now(), "gleepost.contacts.post")
		_otherID, _ := strconv.ParseUint(r.FormValue("user"), 10, 64)
		otherID := gp.UserID(_otherID)
		contact, err := api.AddContact(userID, otherID)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			jsonResponse(w, contact, 201)
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func contactHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	vars := mux.Vars(r)
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	contactID := gp.UserID(_id)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "PUT":
		defer api.Time(time.Now(), "gleepost.contacts.*.put")
		accepted, err := strconv.ParseBool(r.FormValue("accepted"))
		if err != nil {
			accepted = false
		}
		if accepted {
			contact, err := api.AcceptContact(userID, contactID)
			if err != nil {
				jsonErr(w, err, 500)
			} else {
				jsonResponse(w, contact, 200)
			}
		} else {
			jsonResponse(w, missingParamErr("accepted"), 400)
		}
	case r.Method == "DELETE":
		defer api.Time(time.Now(), "gleepost.contacts.*.delete")
		//Implement refusing requests
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}
