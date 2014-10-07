package main

import (
	"log"
	"net/http"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.HandleFunc("/profile/facebook", facebookAssociate)
	base.HandleFunc("/fblogin", facebookHandler)
}

func facebookAssociate(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.profile.facebook.post")
	email := r.FormValue("email")
	pass := r.FormValue("pass")
	id, err := api.ValidatePass(email, pass)
	_fbToken := r.FormValue("fbtoken")
	//Is this a valid facebook token for this app?
	fbToken, errtoken := api.FBValidateToken(_fbToken, 3)
	userID, autherr := authenticate(r)
	switch {
	case r.Method != "POST":
		log.Println(r)
		jsonResponse(w, &EUNSUPPORTED, 405)
	case err != nil:
		if autherr != nil {
			jsonResponse(w, gp.APIerror{Reason: "Bad email/password"}, 400)
		} else {
			//Note to self: The existence of this branch means that a gleepost token is now a password equivalent.
			token, err := api.FacebookLogin(_fbToken)
			if err != nil {
				//This isn't associated with a gleepost account
				err = api.UserSetFB(userID, fbToken.FBUser)
				w.WriteHeader(204)
			} else {
				if token.UserID == userID {
					//The facebook account is already associated with this gleepost account
					w.WriteHeader(204)
				} else {
					jsonResponse(w, gp.APIerror{Reason: "Facebook account already associated with another gleepost account..."}, 400)
				}

			}
		}
	case errtoken != nil:
		jsonResponse(w, gp.APIerror{Reason: "Bad token"}, 400)
	default:
		token, err := api.FacebookLogin(_fbToken)
		if err != nil {
			//This isn't associated with a gleepost account
			err = api.UserSetFB(id, fbToken.FBUser)
			w.WriteHeader(204)
		} else {
			if token.UserID == id {
				//The facebook account is already associated with this gleepost account
				w.WriteHeader(204)
			} else {
				jsonResponse(w, gp.APIerror{Reason: "Facebook account already associated with another gleepost account..."}, 400)
			}

		}
	}
}

func facebookHandler(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.facebook.post")
	if r.Method == "POST" {
		_fbToken := r.FormValue("token")
		email := r.FormValue("email")
		invite := r.FormValue("invite")
		//Is this a valid facebook token for this app?
		fbToken, err := api.FBValidateToken(_fbToken, 3)
		if err != nil {
			jsonResponse(w, gp.APIerror{Reason: "Bad token"}, 400)
			return
		}
		token, err := api.FacebookLogin(_fbToken)
		//If we have an error here, that means that there is no associated gleepost user account.
		if err == nil {
			//If there's an associated user, they're verified already so there's no need to check.
			log.Println("Token: ", token)
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
			_, err := api.UserWithEmail(email)
			if err != nil {
				//There isn't already a user with this email address.
				validates, err := api.ValidateEmail(email)
				if !validates {
					jsonResponse(w, InvalidEmail, 400)
					return
				}
				if err != nil {
					jsonErr(w, err, 500)
					return
				}
				id, err := api.FacebookRegister(_fbToken, email, invite)
				if err != nil {
					jsonErr(w, err, 500)
					return
				}
				if id > 0 {
					//The invite was valid so an account has been created
					//Login
					token, err := api.CreateAndStoreToken(id)
					if err == nil {
						jsonResponse(w, token, 200)
					} else {
						jsonErr(w, err, 500)
					}
					return
				}
				log.Println("Should be unverified response")
				jsonResponse(w, Status{"unverified", email}, 201)
				return
			}
			//User has signed up already with a username+pass
			//If invite is valid, we can log in immediately
			exists, _ := api.InviteExists(email, invite)
			if exists {
				//Verify
				id, err := api.FBSetVerified(email, fbToken.FBUser)
				if err != nil {
					jsonErr(w, err, 500)
					return
				}
				//Login
				token, err := api.CreateAndStoreToken(id)
				if err == nil {
					jsonResponse(w, token, 200)
				} else {
					jsonErr(w, err, 500)
				}
				return
			}
			//otherwise, we must ask for a password
			status := Status{"registered", email}
			jsonResponse(w, status, 200)
			return
		case len(email) > 3 && (err == nil && (storedEmail == email)):
			//We already saw this user, so we don't need to re-send verification
			fallthrough
		case len(email) < 3 && (err == nil):
			//We already saw this user, so we don't need to re-send verification
			//So it should be "unverified" or "registered" as appropriate
			_, err := api.UserWithEmail(storedEmail)
			if err != nil {
				log.Println("Should be unverified response")
				jsonResponse(w, Status{"unverified", storedEmail}, 201)
				return
			}
			status := Status{"registered", storedEmail}
			jsonResponse(w, status, 200)
			return
		case len(email) < 3 && (err != nil):
			jsonResponse(w, gp.APIerror{Reason: "Email required"}, 400)
		}
	} else {
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}
