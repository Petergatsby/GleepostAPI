package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

//InvalidEmail = Your email isn't in our approved list
var InvalidEmail = gp.APIerror{Reason: "Invalid Email"}

//EBADTOKEN = Your token was missing or invalid
var EBADTOKEN = gp.APIerror{Reason: "Invalid credentials"}

func init() {
	base.Handle("/login", timeHandler(api, http.HandlerFunc(loginHandler)))
	base.Handle("/register", timeHandler(api, http.HandlerFunc(registerHandler)))
	base.Handle("/verify/{token:[a-fA-F0-9]+}", timeHandler(api, http.HandlerFunc(verificationHandler)))
	base.Handle("/profile/request_reset", timeHandler(api, http.HandlerFunc(requestResetHandler)))
	base.Handle("/profile/reset/{id:[0-9]+}/{token}", timeHandler(api, http.HandlerFunc(resetPassHandler)))
	base.Handle("/resend_verification", timeHandler(api, http.HandlerFunc(resendVerificationHandler)))
}

//Note to self: validateToken should probably return an error at some point
func authenticate(r *http.Request) (userID gp.UserID, err error) {
	defer api.Time(time.Now(), "gleepost.auth.authenticate")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	userID = gp.UserID(id)
	token := r.FormValue("token")
	if len(token) == 0 {
		credentialsFromHeader := strings.Split(r.Header.Get("X-GP-Auth"), "-")
		id, _ = strconv.ParseUint(credentialsFromHeader[0], 10, 64)
		userID = gp.UserID(id)
		if len(credentialsFromHeader) == 2 {
			token = credentialsFromHeader[1]
		}
	}
	success := api.ValidateToken(userID, token)
	if success {
		go api.Count(1, "gleepost.auth.authenticate.fail")
		return userID, nil
	}
	go api.Count(1, "gleepost.auth.authenticate.success")
	return 0, &EBADTOKEN
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	//Note to self: maybe check cache for user before trying to register
	pass := r.FormValue("pass")
	email := r.FormValue("email")
	first := r.FormValue("first")
	last := r.FormValue("last")
	invite := r.FormValue("invite")
	switch {
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
		//Note to future self : would be neater if
		//we returned _all_ errors not just the first
	case len(first) < 2:
		jsonResponse(w, missingParamErr("first"), 400)
	case len(last) < 1:
		jsonResponse(w, missingParamErr("last"), 400)
	case len(pass) == 0:
		jsonResponse(w, missingParamErr("pass"), 400)
	case len(email) == 0:
		jsonResponse(w, missingParamErr("email"), 400)
	default:
		validates, err := api.ValidateEmail(email)
		if err != nil {
			jsonErr(w, err, 500)
			go api.Count(1, "gleepost.auth.register.500")
			return
		}
		if !validates {
			jsonResponse(w, InvalidEmail, 400)
			go api.Count(1, "gleepost.auth.register.400")
			return
		}
		created, err := api.RegisterUser(pass, email, first, last, invite)
		if err != nil {
			_, ok := err.(gp.APIerror)
			if ok { //Duplicate user/email or password too short
				go api.Count(1, "gleepost.auth.register.400")
				jsonResponse(w, err, 400)
			} else {
				go api.Count(1, "gleepost.auth.register.500")
				jsonErr(w, err, 500)
			}
		} else {
			go api.Count(1, "gleepost.auth.register.201")
			jsonResponse(w, created, 201)
		}
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	pass := r.FormValue("pass")
	switch {
	case r.Method != "POST":
		go api.Count(1, "gleepost.auth.login.405")
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		token, verificationStatus, err := api.AttemptLogin(email, pass)
		switch {
		case err != nil && err == lib.BadLogin:
			go api.Count(1, "gleepost.auth.login.400")
			jsonResponse(w, err, 400)
		case verificationStatus.Status != "":
			go api.Count(1, "gleepost.auth.login.403")
			jsonResponse(w, verificationStatus, 403)
		case err == nil:
			go api.Count(1, "gleepost.auth.login.200")
			jsonResponse(w, token, 200)
		default:
			go api.Count(1, "gleepost.auth.login.500")
			jsonErr(w, err, 500)
		}
	}
}

func changePassHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, "gleepost.profile.change_pass.post.400")
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		oldPass := r.FormValue("old")
		newPass := r.FormValue("new")
		err := api.ChangePass(userID, oldPass, newPass)
		if err != nil {
			go api.Count(1, "gleepost.profile.change_pass.post.400")
			//Assuming that most errors will be bad input for now
			jsonErr(w, err, 400)
			return
		}
		go api.Count(1, "gleepost.profile.change_pass.post.204")
		w.WriteHeader(204)
	default:
		go api.Count(1, "gleepost.profile.change_pass.post.405")
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func verificationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		vars := mux.Vars(r)
		err := api.Verify(vars["token"])
		if err != nil {
			log.Println(err)
			go api.Count(1, "gleepost.verify.post.400")
			jsonResponse(w, gp.APIerror{Reason: "Bad verification token"}, 400)
			return
		}
		go api.Count(1, "gleepost.verify.post.200")
		jsonResponse(w, struct {
			Verified bool `json:"verified"`
		}{true}, 200)
		return
	}
	go api.Count(1, "gleepost.verify.post.405")
	jsonResponse(w, &EUNSUPPORTED, 405)
}

func requestResetHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "POST":
		email := r.FormValue("email")
		err := api.RequestReset(email)
		if err != nil {
			go api.Count(1, "gleepost.profile.request_reset.post.400")
			jsonErr(w, err, 400)
			return
		}
		go api.Count(1, "gleepost.profile.request_reset.post.204")
		w.WriteHeader(204)
		return
	default:
		go api.Count(1, "gleepost.profile.request_reset.post.405")
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func resetPassHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "POST":
		vars := mux.Vars(r)
		id, err := strconv.ParseUint(vars["id"], 10, 64)
		if err != nil {
			go api.Count(1, "gleepost.profile.reset.post.400")
			jsonErr(w, err, 400)
			return
		}
		userID := gp.UserID(id)
		pass := r.FormValue("pass")
		err = api.ResetPass(userID, vars["token"], pass)
		if err != nil {
			go api.Count(1, "gleepost.profile.reset.post.400")
			jsonErr(w, err, 400)
			return
		}
		go api.Count(1, "gleepost.profile.reset.post.204")
		w.WriteHeader(204)
		return
	default:
		go api.Count(1, "gleepost.profile.reset.post.405")
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func resendVerificationHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "POST":
		email := r.FormValue("email")
		userID, err := api.UserWithEmail(email)
		if err != nil {
			fbid, err := api.FBUserWithEmail(email)
			if err == nil {
				go api.Count(1, "gleepost.resend_verification.post.400")
				jsonErr(w, err, 400)
				return
			}
			api.FBissueVerification(fbid)
		} else {
			user, err := api.GetUser(userID)
			if err != nil {
				go api.Count(1, "gleepost.resend_verification.post.500")
				jsonErr(w, err, 500)
				return
			}
			api.GenerateAndSendVerification(userID, user.Name, email)
		}
		go api.Count(1, "gleepost.resend_verification.post.204")
		w.WriteHeader(204)
	default:
		go api.Count(1, "gleepost.resend_verification.post.405")
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}
