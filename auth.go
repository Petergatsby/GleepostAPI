package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Petergatsby/GleepostAPI/lib"
	"github.com/Petergatsby/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

//EBADTOKEN = Your token was missing or invalid
var EBADTOKEN = gp.APIerror{Reason: "Invalid credentials"}

func init() {
	base.Handle("/login", timeHandler(api, http.HandlerFunc(loginHandler))).Methods("POST")
	base.Handle("/login", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/login", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/register", timeHandler(api, http.HandlerFunc(registerHandler))).Methods("POST")
	base.Handle("/register", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/verify/{token}", timeHandler(api, http.HandlerFunc(verificationHandler))).Methods("POST")
	base.Handle("/verify/{token}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/request_reset", timeHandler(api, http.HandlerFunc(requestResetHandler))).Methods("POST")
	base.Handle("/profile/request_reset", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/reset/{id:[0-9]+}/{token}", timeHandler(api, http.HandlerFunc(resetPassHandler))).Methods("POST")
	base.Handle("/profile/reset/{id:[0-9]+}/{token}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/resend_verification", timeHandler(api, http.HandlerFunc(resendVerificationHandler))).Methods("POST")
	base.Handle("/resend_verification", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

//Note to self: validateToken should probably return an error at some point
func authenticate(r *http.Request) (userID gp.UserID, err error) {
	defer api.Statsd.Time(time.Now(), "gleepost.auth.authenticate")
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
	success := api.Auth.ValidateToken(userID, token)
	if success {
		go api.Statsd.Count(1, "gleepost.auth.authenticate.fail")
		return userID, nil
	}
	go api.Statsd.Count(1, "gleepost.auth.authenticate.success")
	return 0, &EBADTOKEN
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	//Note to self: maybe check cache for user before trying to register
	pass := r.FormValue("pass")
	email := r.FormValue("email")
	first := r.FormValue("first")
	last := r.FormValue("last")
	invite := r.FormValue("invite")
	created, err := api.AttemptRegister(email, pass, first, last, invite)
	switch {
	//Note to future self : would be neater if
	//we returned _all_ errors not just the first
	case err == lib.MissingParamFirst:
		fallthrough
	case err == lib.MissingParamLast:
		fallthrough
	case err == lib.MissingParamPass:
		fallthrough
	case err == lib.MissingParamEmail:
		fallthrough
	case err == lib.ETOOWEAK:
		fallthrough
	case err == lib.UserAlreadyExists:
		fallthrough
	case err == lib.InvalidEmail:
		go api.Statsd.Count(1, "gleepost.auth.register.400")
		jsonResponse(w, err, 400)
	case err != nil:
		go api.Statsd.Count(1, "gleepost.auth.register.500")
		jsonErr(w, err, 500)
	default:
		go api.Statsd.Count(1, "gleepost.auth.register.201")
		jsonResponse(w, created, 201)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	pass := r.FormValue("pass")
	token, verificationStatus, err := api.AttemptLogin(email, pass)
	switch {
	case err != nil && err == lib.BadLogin:
		go api.Statsd.Count(1, "gleepost.auth.login.400")
		jsonResponse(w, err, 400)
	case verificationStatus.Status != "":
		go api.Statsd.Count(1, "gleepost.auth.login.403")
		jsonResponse(w, verificationStatus, 403)
	case err == nil:
		go api.Statsd.Count(1, "gleepost.auth.login.200")
		jsonResponse(w, token, 200)
	default:
		go api.Statsd.Count(1, "gleepost.auth.login.500")
		jsonErr(w, err, 500)
	}
}

func changePassHandler(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	oldPass := r.FormValue("old")
	newPass := r.FormValue("new")
	err := api.ChangePass(userID, oldPass, newPass)
	if err != nil {
		go api.Statsd.Count(1, "gleepost.profile.change_pass.post.400")
		//Assuming that most errors will be bad input for now
		jsonErr(w, err, 400)
		return
	}
	go api.Statsd.Count(1, "gleepost.profile.change_pass.post.204")
	w.WriteHeader(204)
}

func verificationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	err := api.Verify(vars["token"])
	if err != nil {
		go api.Statsd.Count(1, "gleepost.verify.post.400")
		jsonResponse(w, gp.APIerror{Reason: "Bad verification token"}, 400)
		return
	}
	go api.Statsd.Count(1, "gleepost.verify.post.200")
	jsonResponse(w, struct {
		Verified bool `json:"verified"`
	}{true}, 200)
	return
}

func requestResetHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	err := api.RequestReset(email)
	if err != nil {
		go api.Statsd.Count(1, "gleepost.profile.request_reset.post.400")
		jsonErr(w, err, 400)
		return
	}
	go api.Statsd.Count(1, "gleepost.profile.request_reset.post.204")
	w.WriteHeader(204)
	return
}

func resetPassHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		go api.Statsd.Count(1, "gleepost.profile.reset.post.400")
		jsonErr(w, err, 400)
		return
	}
	userID := gp.UserID(id)
	pass := r.FormValue("pass")
	err = api.ResetPass(userID, vars["token"], pass)
	if err != nil {
		go api.Statsd.Count(1, "gleepost.profile.reset.post.400")
		jsonErr(w, err, 400)
		return
	}
	go api.Statsd.Count(1, "gleepost.profile.reset.post.204")
	w.WriteHeader(204)
	return
}

func resendVerificationHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	err := api.AttemptResendVerification(email)
	switch {
	case err == lib.NoSuchUser:
		go api.Statsd.Count(1, "gleepost.resend_verification.post.400")
		jsonErr(w, err, 400)
	case err != nil:
		go api.Statsd.Count(1, "gleepost.resend_verification.post.500")
		jsonErr(w, err, 500)
	default:
		go api.Statsd.Count(1, "gleepost.resend_verification.post.204")
		w.WriteHeader(204)
	}
}
