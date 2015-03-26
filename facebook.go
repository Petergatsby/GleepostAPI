package main

import (
	"net/http"

	"github.com/draaglom/GleepostAPI/lib"
)

func init() {
	base.Handle("/profile/facebook", timeHandler(api, http.HandlerFunc(facebookAssociate))).Methods("POST")
	base.Handle("/profile/facebook", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/fblogin", timeHandler(api, http.HandlerFunc(facebookHandler))).Methods("POST")
	base.Handle("/fblogin", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func facebookAssociate(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	pass := r.FormValue("pass")
	_fbToken := r.FormValue("fbtoken")
	//Is this a valid facebook token for this app?
	userID, autherr := authenticate(r)
	var err error
	switch {
	case autherr == nil:
		//Note to self: The existence of this branch means that a gleepost token is now a password equivalent.
		err = api.AssociateFB(userID, _fbToken)
	default:
		err = api.AttemptAssociationWithCredentials(email, pass, _fbToken)
	}
	switch {
	case err != nil && err == lib.AlreadyAssociated:
		fallthrough
	case err != nil && err == lib.BadFBToken:
		go api.Count(1, "gleepost.profile.facebook.post.400")
		jsonResponse(w, err, 400)
	case err != nil:
		go api.Count(1, "gleepost.profile.facebook.post.500")
		jsonResponse(w, err, 500)
	default:
		go api.Count(1, "gleepost.profile.facebook.post.204")
		w.WriteHeader(204)
	}
}

func facebookHandler(w http.ResponseWriter, r *http.Request) {
	_fbToken := r.FormValue("token")
	email := r.FormValue("email")
	invite := r.FormValue("invite")
	token, _, status, err := api.FacebookLogin(_fbToken, email, invite)
	switch {
	case err == lib.BadFBToken:
		fallthrough
	case err == lib.InvalidEmail:
		fallthrough
	case err == lib.FBNoEmail:
		go api.Count(1, "gleepost.facebook.post.400")
		jsonResponse(w, err, 400)
		return
	case err != nil:
		go api.Count(1, "gleepost.facebook.post.500")
		jsonErr(w, err, 500)
		return
	case status.Status == "registered":
		//The invite wasn't valid; this means that this user is already registered but the fb user wasn't able to prove they are this gleepost user.
		go api.Count(1, "gleepost.facebook.post.200")
		jsonResponse(w, status, 200)
	case status.Status == "unverified":
		go api.Count(1, "gleepost.facebook.post.201")
		jsonResponse(w, status, 201)
		return
	case token.UserID > 0:
		go api.Count(1, "gleepost.facebook.post.200")
		jsonResponse(w, token, 200)
		return
	case err == nil:
		//If there's an associated user, they're verified already so there's no need to check.
		go api.Count(1, "gleepost.facebook.post.201")
		jsonResponse(w, token, 201)
		return
	}
}
