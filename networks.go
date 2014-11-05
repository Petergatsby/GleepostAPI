package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

func init() {
	base.HandleFunc("/networks/{network:[0-9]+}/posts", getPosts).Methods("GET")
	base.HandleFunc("/networks/{network:[0-9]+}/posts", postPosts).Methods("POST")
	base.HandleFunc("/networks/{network:[0-9]+}", getNetwork).Methods("GET")
	base.HandleFunc("/networks/{network:[0-9]+}", putNetwork).Methods("PUT")
	base.HandleFunc("/networks/{network:[0-9]+}/users", postNetworkUsers).Methods("POST")
	base.HandleFunc("/networks/{network:[0-9]+}/users", getNetworkUsers).Methods("GET")
	base.HandleFunc("/networks/{network:[0-9]+}/admins", postNetworkAdmins).Methods("POST")
	base.HandleFunc("/networks/{network:[0-9]+}/admins", getNetworkAdmins).Methods("GET")
	base.HandleFunc("/networks/{network:[0-9]+}/admins/{user:[0-9]+}", deleteNetworkAdmins).Methods("DELETE")
	base.HandleFunc("/networks", postNetworks).Methods("POST")

	base.HandleFunc("/profile/networks", getGroups)
	base.HandleFunc("/profile/networks/posts", getGroupPosts).Methods("GET")
	base.HandleFunc("/profile/networks/{network:[0-9]+}", deleteUserNetwork).Methods("DELETE")
}

func getGroups(w http.ResponseWriter, r *http.Request) {
	t := time.Now()
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		var otherID gp.UserID
		vars := mux.Vars(r)
		_id, ok := vars["id"]
		var url string
		if !ok {
			otherID = userID
			url = "gleepost.profile.networks.get"
		} else {
			id, err := strconv.ParseUint(_id, 10, 64)
			if err != nil {
				jsonErr(w, err, 400)
				return
			}
			otherID = gp.UserID(id)
			url = fmt.Sprintf("gleepost.users.%d.networks.get", otherID)
		}
		defer api.Time(t, url)
		networks, err := api.UserGetUserGroups(userID, otherID)
		if err != nil {
			go api.Count(1, url+".500")
			jsonErr(w, err, 500)
			return
		}
		go api.Count(1, url+".200")
		jsonResponse(w, networks, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func getNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%s.get", vars["network"])
	defer api.Time(time.Now(), url)
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		_netID, err := strconv.ParseUint(vars["network"], 10, 16)
		if err != nil {
			go api.Count(1, url+".400")
			jsonErr(w, err, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		network, err := api.UserGetNetwork(userID, netID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		go api.Count(1, url+".200")
		jsonResponse(w, network, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func postNetworks(w http.ResponseWriter, r *http.Request) {
	url := "gleepost.networks.post"
	defer api.Time(time.Now(), url)
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		name := r.FormValue("name")
		url := r.FormValue("url")
		desc := r.FormValue("desc")
		privacy := r.FormValue("privacy")
		privacy = strings.ToLower(privacy)
		if privacy != "public" && privacy != "private" && privacy != "secret" {
			privacy = "private"
		}
		switch {
		case len(name) == 0:
			go api.Count(1, url+".400")
			jsonResponse(w, missingParamErr("name"), 400)
		default:
			network, err := api.CreateGroup(userID, name, url, desc, privacy)
			if err != nil {
				e, ok := err.(*gp.APIerror)
				if ok && *e == lib.ENOTALLOWED {
					go api.Count(1, url+".403")
					jsonResponse(w, e, 403)
				} else {
					go api.Count(1, url+".500")
					jsonErr(w, err, 500)
				}
				return
			}
			go api.Count(1, url+".201")
			jsonResponse(w, network, 201)
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func postNetworkUsers(w http.ResponseWriter, r *http.Request) {
	//This is a mess.
	//TODO: Consolidate the various AddUser* fns into one; return a composite error list.
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%s.users.post", vars["network"])
	defer api.Time(time.Now(), url)
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Count(1, url+".400")
			jsonErr(w, err, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		_users := strings.Split(r.FormValue("users"), ",")
		_fbUsers := strings.Split(r.FormValue("fbusers"), ",")
		var fbusers []uint64
		var users []gp.UserID
		for _, u := range _users {
			user, err := strconv.ParseUint(u, 10, 64)
			if err == nil {
				users = append(users, gp.UserID(user))
			}
		}
		for _, f := range _fbUsers {
			fbuser, err := strconv.ParseUint(f, 10, 64)
			if err == nil {
				fbusers = append(fbusers, fbuser)
			}
		}
		var added = false
		if len(users) > 0 {
			added = true
			_, err = api.UserAddUsersToGroup(userID, users, netID)
			if err != nil {
				e, ok := err.(*gp.APIerror)
				if ok && *e == lib.ENOTALLOWED {
					go api.Count(1, url+".403")
					jsonResponse(w, e, 403)
				} else {
					go api.Count(1, url+".500")
					jsonErr(w, err, 500)
				}
				return
			}
		}
		if len(fbusers) > 0 {
			added = true
			_, err = api.UserAddFBUsersToGroup(userID, fbusers, netID)
			if err != nil {
				e, ok := err.(*gp.APIerror)
				if ok && *e == lib.ENOTALLOWED {
					go api.Count(1, url+".403")
					jsonResponse(w, e, 403)
				} else {
					go api.Count(1, url+".500")
					jsonErr(w, err, 500)
				}
				return
			}
		}
		if len(r.FormValue("email")) > 5 {
			added = true
			err = api.UserInviteEmail(userID, netID, r.FormValue("email"))
			if err != nil {
				e, ok := err.(*gp.APIerror)
				if ok && *e == lib.ENOTALLOWED {
					go api.Count(1, url+".403")
					jsonResponse(w, e, 403)
				} else {
					go api.Count(1, url+".500")
					jsonErr(w, err, 500)
				}
				return
			}
		}
		if !added {
			go api.Count(1, url+".400")
			jsonResponse(w, gp.APIerror{Reason: "Must add either user(s), facebook user(s) or an email"}, 400)
			return
		}
		go api.Count(1, url+".204")
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func postNetworkAdmins(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%s.admins.post", vars["network"])
	defer api.Time(time.Now(), url)
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Count(1, url+".400")
			jsonErr(w, err, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		_users := strings.Split(r.FormValue("users"), ",")
		for _, u := range _users {
			_user, err := strconv.ParseUint(u, 10, 64)
			if err == nil {
				user := gp.UserID(_user)
				err = api.UserChangeRole(userID, user, netID, "administrator")
				if err != nil {
					e, ok := err.(*gp.APIerror)
					if ok && *e == lib.ENOTALLOWED {
						go api.Count(1, url+".403")
						jsonResponse(w, e, 403)
					} else {
						go api.Count(1, url+".500")
						jsonErr(w, err, 500)
					}
					return
				}
			}
		}
		users, err := api.UserGetGroupAdmins(userID, netID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
		}
		go api.Count(1, url+".200")
		jsonResponse(w, users, 200)
	}
}

//Note to self: should probably consolidate this with getNetworkUsers.
func getNetworkAdmins(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%s.admins.get", vars["network"])
	defer api.Time(time.Now(), url)
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Count(1, url+".400")
			jsonErr(w, err, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		users, err := api.UserGetGroupAdmins(userID, netID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
		}
		go api.Count(1, url+".200")
		jsonResponse(w, users, 200)
	}
}

func deleteNetworkAdmins(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%d.admins.%d.delete", vars["network"], vars["user"])
	defer api.Time(time.Now(), url)
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Count(1, url+".400")
			jsonErr(w, err, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		//Can ignore the error, because api.UserChangeRole will complain if id 0 anyway.
		_user, _ := strconv.ParseUint(vars["user"], 10, 64)
		user := gp.UserID(_user)
		err = api.UserChangeRole(userID, user, netID, "member")
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		go api.Count(1, url+".204")
		w.WriteHeader(204)
	}
}

func getNetworkUsers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%s.users.get", vars["network"])
	defer api.Time(time.Now(), url)
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Count(1, url+".400")
			jsonErr(w, err, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		users, err := api.UserGetGroupMembers(userID, netID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
		}
		go api.Count(1, url+".200")
		jsonResponse(w, users, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

//getGroupPosts is basically the same goddamn thing as getPosts. stop copy-pasting you cretin.
func getGroupPosts(w http.ResponseWriter, r *http.Request) {
	url := "gleepost.profile.networks.get"
	defer api.Time(time.Now(), url)
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
		if err != nil {
			start = 0
		}
		before, err := strconv.ParseInt(r.FormValue("before"), 10, 64)
		if err != nil {
			before = 0
		}
		after, err := strconv.ParseInt(r.FormValue("after"), 10, 64)
		if err != nil {
			after = 0
		}
		//First: which paging scheme are we using
		var mode int
		var index int64
		switch {
		case after > 0:
			mode = gp.OAFTER
			index = after
		case before > 0:
			mode = gp.OBEFORE
			index = before
		default:
			mode = gp.OSTART
			index = start
		}
		posts, err := api.UserGetGroupsPosts(userID, mode, index, api.Config.PostPageSize, r.FormValue("filter"))
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		go api.Count(1, url+".200")
		jsonResponse(w, posts, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func putNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%s.put", vars["network"])
	defer api.Time(time.Now(), url)
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "PUT":
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Count(1, url+".400")
			jsonErr(w, err, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		url := r.FormValue("url")
		err = api.UserSetNetworkImage(userID, netID, url)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		group, err := api.UserGetNetwork(userID, netID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		go api.Count(1, url+".200")
		jsonResponse(w, group, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func deleteUserNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.profile.networks.%s.delete", vars["networks"])
	defer api.Time(time.Now(), url)
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "DELETE":
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Count(1, url+".400")
			jsonErr(w, err, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		err = api.UserLeaveGroup(userID, netID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
		}
		go api.Count(1, url+".204")
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}