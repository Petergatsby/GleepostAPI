package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

//Status represents a user's current signup state (You should only ever see "unverified" (you have an account pending email verification" or "registered" (this email is taken by someone else)
type Status struct {
	Status string `json:"status"`
	Email  string `json:"email"`
}

func newStatus(status, email string) *Status {
	return &Status{Status: status, Email: email}
}

//InvalidEmail = Your email isn't in our approved list
var InvalidEmail = gp.APIerror{Reason: "Invalid Email"}

//BadLogin = guess...
var BadLogin = gp.APIerror{Reason: "Bad username/password"}

//EBADTOKEN = Your token was missing or invalid
var EBADTOKEN = gp.APIerror{Reason: "Invalid credentials"}

func init() {
	base.HandleFunc("/login", loginHandler)
	base.HandleFunc("/register", registerHandler)
	base.HandleFunc("/verify/{token:[a-fA-F0-9]+}", verificationHandler)
	base.HandleFunc("/profile/request_reset", requestResetHandler)
	base.HandleFunc("/profile/reset/{id:[0-9]+}/{token}", resetPassHandler)
	base.HandleFunc("/resend_verification", resendVerificationHandler)
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
	/* POST /register
		requires parameters: user, pass, email
	        example responses:
	        HTTP 201
		{"id":2397}
		HTTP 400
		{"error":"Invalid email"}
	*/

	//Note to self: maybe check cache for user before trying to register
	defer api.Time(time.Now(), "gleepost.auth.register")
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
	/* POST /login
		requires parameters: email, pass
		example responses:
		HTTP 200
	        {
	            "id":2397,
	            "value":"552e5a9687ec04418b3b4da61a8b062dbaf5c7937f068341f36a4b4fcbd4ed45",
	            "expiry":"2013-09-25T14:43:17.664646892Z"
	        }
		HTTP 400
		{"error":"Bad username/password"}
	*/
	defer api.Time(time.Now(), "gleepost.auth.login")
	email := r.FormValue("email")
	pass := r.FormValue("pass")
	id, err := api.ValidatePass(email, pass)
	switch {
	case r.Method != "POST":
		go api.Count(1, "gleepost.auth.login.405")
		jsonResponse(w, &EUNSUPPORTED, 405)
	case err == nil:
		verified, err := api.IsVerified(id)
		switch {
		case err != nil:
			go api.Count(1, "gleepost.auth.login.500")
			jsonErr(w, err, 500)
		case !verified:
			resp := newStatus("unverified", email)
			go api.Count(1, "gleepost.auth.login.403")
			jsonResponse(w, resp, 403)
		default:
			token, err := api.CreateAndStoreToken(id)
			if err == nil {
				go api.Count(1, "gleepost.auth.login.200")
				jsonResponse(w, token, 200)
			} else {
				go api.Count(1, "gleepost.auth.login.500")
				jsonErr(w, err, 500)
			}
		}
	default:
		go api.Count(1, "gleepost.auth.login.400")
		jsonResponse(w, BadLogin, 400)
	}
}

func changePassHandler(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.profile.change_pass.post")
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
	defer api.Time(time.Now(), "gleepost.verify.post")
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
	defer api.Time(time.Now(), "gleepost.profile.request_reset.post")
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
	defer api.Time(time.Now(), "gleepost.profile.reset.post")
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
	defer api.Time(time.Now(), "gleepost.resend_verification.post")
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
