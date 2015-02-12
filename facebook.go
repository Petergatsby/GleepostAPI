package main

import (
	"log"
	"net/http"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.Handle("/profile/facebook", timeHandler(api, http.HandlerFunc(facebookAssociate)))
	base.Handle("/fblogin", timeHandler(api, http.HandlerFunc(facebookHandler)))
}

func facebookAssociate(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	pass := r.FormValue("pass")
	_fbToken := r.FormValue("fbtoken")
	//Is this a valid facebook token for this app?
	fbToken, errtoken := api.FBValidateToken(_fbToken, 3)
	userID, autherr := authenticate(r)
	switch {
	case r.Method != "POST":
		go api.Count(1, "gleepost.profile.facebook.post.405")
		jsonResponse(w, &EUNSUPPORTED, 405)
	case errtoken != nil:
		go api.Count(1, "gleepost.profile.facebook.post.400")
		jsonResponse(w, gp.APIerror{Reason: "Bad token"}, 400)
	case autherr == nil:
		//Note to self: The existence of this branch means that a gleepost token is now a password equivalent.
		err := api.AssociateFB(userID, _fbToken, fbToken.FBUser)
		switch {
		case err != nil && err == lib.AlreadyAssociated:
			go api.Count(1, "gleepost.profile.facebook.post.400")
			jsonResponse(w, err, 400)
		case err != nil:
			go api.Count(1, "gleepost.profile.facebook.post.500")
			jsonResponse(w, err, 500)
		default:
			go api.Count(1, "gleepost.profile.facebook.post.204")
			w.WriteHeader(204)
		}
	default:
		err := api.AttemptAssociationWithCredentials(email, pass, _fbToken, fbToken.FBUser)
		switch {
		case err != nil && err == lib.BadLogin:
			go api.Count(1, "gleepost.profile.facebook.post.400")
			jsonResponse(w, gp.APIerror{Reason: "Bad email/password"}, 400)
		case err != nil && err == lib.AlreadyAssociated:
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
}

func facebookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		_fbToken := r.FormValue("token")
		email := r.FormValue("email")
		invite := r.FormValue("invite")
		//Is this a valid facebook token for this app?
		fbToken, err := api.FBValidateToken(_fbToken, 3)
		if err != nil {
			go api.Count(1, "gleepost.facebook.post.400")
			jsonResponse(w, gp.APIerror{Reason: "Bad token"}, 400)
			return
		}
		token, err := api.FacebookLogin(_fbToken)
		//If we have an error here, that means that there is no associated gleepost user account.
		if err == nil {
			//If there's an associated user, they're verified already so there's no need to check.
			log.Println("Token: ", token)
			go api.Count(1, "gleepost.facebook.post.201")
			jsonResponse(w, token, 201)
			return
		}
		log.Println("Error logging in with facebook, probably means there's no associated gleepost account:", err)
		//Did the user provide an email (takes precedence over stored email, because they might have typo'd the first time)
		var storedEmail string
		storedEmail, err = api.FBGetEmail(fbToken.FBUser)
		switch {
		//Has this email been seen before for this user?
		case len(email) > 3 && (err != nil || storedEmail != email):
			//Either we don't have a stored email for this user, or at least it wasn't this one.
			//(So we should check if there's an existing signed up / verified user)
			//(and if not, issue a verification email)
			//(since this is the first time they've signed up with this email)
			token, verification, err := api.FBFirstTimeWithEmail(email, _fbToken, invite, fbToken.FBUser)
			switch {
			case err == lib.InvalidEmail:
				go api.Count(1, "gleepost.facebook.post.400")
				jsonResponse(w, err, 400)
			case err != nil:
				go api.Count(1, "gleepost.facebook.post.500")
				jsonErr(w, err, 500)
				return
			case token.UserID > 0:
				go api.Count(1, "gleepost.facebook.post.200")
				jsonResponse(w, token, 200)
				return
			case verification.Status == "registered":
				//The invite wasn't valid; this means that this user is already registered but the fb user wasn't able to prove they are this gleepost user.
				go api.Count(1, "gleepost.facebook.post.200")
				jsonResponse(w, verification, 200)
			default:
				go api.Count(1, "gleepost.facebook.post.201")
				jsonResponse(w, verification, 201)
				return
			}
		case len(email) > 3 && (err == nil && (storedEmail == email)):
			//We already saw this user, so we don't need to re-send verification
			fallthrough
		case len(email) < 3 && (err == nil):
			//We already saw this user, so we don't need to re-send verification
			//So it should be "unverified" or "registered" as appropriate
			_, err := api.UserWithEmail(storedEmail)
			if err != nil {
				log.Println("Should be unverified response")
				go api.Count(1, "gleepost.facebook.post.201")
				jsonResponse(w, gp.NewStatus("unverified", storedEmail), 201)
				return
			}
			status := gp.NewStatus("registered", storedEmail)
			go api.Count(1, "gleepost.facebook.post.200")
			jsonResponse(w, status, 200)
			return
		case len(email) < 3 && (err != nil):
			go api.Count(1, "gleepost.facebook.post.400")
			jsonResponse(w, gp.APIerror{Reason: "Email required"}, 400)
		}
	} else {
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}
