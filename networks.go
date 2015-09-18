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
	base.Handle("/networks", timeHandler(api, http.HandlerFunc(getNetworks))).Methods("GET")
	base.Handle("/networks", timeHandler(api, http.HandlerFunc(postNetworks))).Methods("POST")
	base.Handle("/networks", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/networks", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/networks/{network:[0-9]+}", timeHandler(api, http.HandlerFunc(getNetwork))).Methods("GET")
	base.Handle("/networks/{network:[0-9]+}", timeHandler(api, http.HandlerFunc(putNetwork))).Methods("PUT")
	base.Handle("/networks/{network:[0-9]+}", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/networks/{network:[0-9]+}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/networks/{network:[0-9]+}/posts", timeHandler(api, http.HandlerFunc(getPosts))).Methods("GET")
	base.Handle("/networks/{network:[0-9]+}/posts", timeHandler(api, http.HandlerFunc(postPosts))).Methods("POST")
	base.Handle("/networks/{network:[0-9]+}/posts", timeHandler(api, http.HandlerFunc(putPosts))).Methods("PUT")
	base.Handle("/networks/{network:[0-9]+}/posts", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/networks/{network:[0-9]+}/posts", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/networks/{network:[0-9]+}/users", timeHandler(api, http.HandlerFunc(postNetworkUsers))).Methods("POST")
	base.Handle("/networks/{network:[0-9]+}/users", timeHandler(api, http.HandlerFunc(getNetworkUsers))).Methods("GET")
	base.Handle("/networks/{network:[0-9]+}/users", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/networks/{network:[0-9]+}/users", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/networks/{network:[0-9]+}/admins", timeHandler(api, http.HandlerFunc(postNetworkAdmins))).Methods("POST")
	base.Handle("/networks/{network:[0-9]+}/admins", timeHandler(api, http.HandlerFunc(getNetworkAdmins))).Methods("GET")
	base.Handle("/networks/{network:[0-9]+}/admins", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/networks/{network:[0-9]+}/requests", timeHandler(api, http.HandlerFunc(postNetworkRequests))).Methods("POST")
	base.Handle("/networks/{network:[0-9]+}/requests", timeHandler(api, http.HandlerFunc(getNetworkRequests))).Methods("GET")
	base.Handle("/networks/{network:[0-9]+}/requests", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/networks/{network:[0-9]+}/requests/{user:[0-9]+}", timeHandler(api, http.HandlerFunc(deleteNetworkRequest))).Methods("DELETE")
	base.Handle("/networks/{network:[0-9]+}/requests/{user:[0-9]+}", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/networks/{network:[0-9]+}/admins/{user:[0-9]+}", timeHandler(api, http.HandlerFunc(deleteNetworkAdmins))).Methods("DELETE")
	base.Handle("/networks/{network:[0-9]+}/admins/{user:[0-9]+}", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/networks/{network:[0-9]+}/admins/{user:[0-9]+}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))

	base.Handle("/profile/networks/mute_badges", timeHandler(api, http.HandlerFunc(muteGroupBadge))).Methods("POST")
	base.Handle("/profile/networks/mute_badges", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/profile/networks/mute_badges", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/networks", timeHandler(api, http.HandlerFunc(getGroups))).Methods("GET")
	base.Handle("/profile/networks", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/networks/posts", timeHandler(api, http.HandlerFunc(getGroupPosts))).Methods("GET")
	base.Handle("/profile/networks/posts", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/profile/networks/{network:[0-9]+}", timeHandler(api, http.HandlerFunc(deleteUserNetwork))).Methods("DELETE")
	base.Handle("/profile/networks/{network:[0-9]+}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	//public
	base.Handle("/university/{network:[0-9]+}", timeHandler(api, http.HandlerFunc(publicGetUniversity))).Methods("GET")
}

func getGroups(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		var otherID gp.UserID
		vars := mux.Vars(r)
		_id, ok := vars["id"]
		if !ok {
			otherID = userID
		} else {
			id, err := strconv.ParseUint(_id, 10, 64)
			if err != nil {
				jsonErr(w, err, 400)
				return
			}
			otherID = gp.UserID(id)
		}
		index, _ := strconv.ParseInt(r.FormValue("start"), 10, 64)
		_count, _ := strconv.ParseInt(r.FormValue("count"), 10, 64)
		var order int
		switch {
		case r.FormValue("order") == "by_last_activity":
			order = lib.ByActivity
		case r.FormValue("order") == "by_last_message":
			order = lib.ByMessages
		default:
			order = lib.ByActivity
		}
		count := int(_count)
		networks, err := api.UserGetUserGroups(userID, otherID, index, count, order)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, networks, 200)
	}
}

func getNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%s.get", vars["network"])
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Statsd.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, err := strconv.ParseUint(vars["network"], 10, 16)
		if err != nil {
			go api.Statsd.Count(1, url+".400")
			jsonErr(w, err, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		network, err := api.UserGetNetwork(userID, netID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Statsd.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Statsd.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		go api.Statsd.Count(1, url+".200")
		jsonResponse(w, network, 200)
	}
}

func postNetworks(w http.ResponseWriter, r *http.Request) {
	url := "gleepost.networks.post"
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Statsd.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		name := r.FormValue("name")
		url := r.FormValue("url")
		desc := r.FormValue("desc")
		privacy := r.FormValue("privacy")
		category := r.FormValue("category")
		university, err := strconv.ParseBool(r.FormValue("university"))
		var network interface{}
		switch {
		case len(name) == 0:
			go api.Statsd.Count(1, url+".400")
			jsonResponse(w, missingParamErr("name"), 400)
		case err != nil || !university:
			network, err = api.CreateGroup(userID, name, url, desc, privacy, category)
		default:
			domains := strings.Split(r.FormValue("domains"), ",")
			network, err = api.AdminCreateUniversity(userID, name, domains...)
		}
		switch {
		case err == lib.ENOTALLOWED:
			go api.Statsd.Count(1, url+".403")
			jsonResponse(w, err, 403)
		case err != nil:
			go api.Statsd.Count(1, url+".500")
			jsonErr(w, err, 500)
		default:
			go api.Statsd.Count(1, url+".201")
			jsonResponse(w, network, 201)
		}
	}
}

func postNetworkUsers(w http.ResponseWriter, r *http.Request) {
	//This is a mess.
	//TODO: Consolidate the various AddUser* fns into one; return a composite error list.
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%s.users.post", vars["network"])
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Statsd.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Statsd.Count(1, url+".400")
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
		var emails []string
		if len(r.FormValue("email")) > 0 {
			emails = append(emails, r.FormValue("email"))
		}
		err = api.UserAddToGroup(userID, netID, users, fbusers, emails)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Statsd.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else if err == lib.NobodyAdded {
				go api.Statsd.Count(1, url+".400")
				jsonResponse(w, err, 400)
			} else {
				go api.Statsd.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		go api.Statsd.Count(1, url+".204")
		w.WriteHeader(204)
	}
}

func postNetworkAdmins(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%s.admins.post", vars["network"])
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Statsd.Count(1, url+".400")
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
						go api.Statsd.Count(1, url+".403")
						jsonResponse(w, e, 403)
					} else {
						go api.Statsd.Count(1, url+".500")
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
				go api.Statsd.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Statsd.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		go api.Statsd.Count(1, url+".200")
		jsonResponse(w, users, 200)
	}
}

//Note to self: should probably consolidate this with getNetworkUsers.
func getNetworkAdmins(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%s.admins.get", vars["network"])
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Statsd.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Statsd.Count(1, url+".400")
			jsonErr(w, err, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		users, err := api.UserGetGroupAdmins(userID, netID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Statsd.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Statsd.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		go api.Statsd.Count(1, url+".200")
		jsonResponse(w, users, 200)
	}
}

func deleteNetworkAdmins(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%s.admins.%s.delete", vars["network"], vars["user"])
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Statsd.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Statsd.Count(1, url+".400")
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
				go api.Statsd.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Statsd.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		go api.Statsd.Count(1, url+".204")
		w.WriteHeader(204)
	}
}

func getNetworkUsers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%s.users.get", vars["network"])
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Statsd.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Statsd.Count(1, url+".400")
			jsonErr(w, err, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		users, err := api.UserGetGroupMembers(userID, netID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Statsd.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Statsd.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		go api.Statsd.Count(1, url+".200")
		jsonResponse(w, users, 200)
	}
}

//getGroupPosts is basically the same goddamn thing as getPosts. stop copy-pasting you cretin.
func getGroupPosts(w http.ResponseWriter, r *http.Request) {
	url := "gleepost.profile.networks.get"
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Statsd.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		mode, index := interpretPagination(r.FormValue("start"), r.FormValue("before"), r.FormValue("after"))
		posts, err := api.UserGetGroupsPosts(userID, mode, index, api.Config.PostPageSize, r.FormValue("filter"))
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Statsd.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Statsd.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		go api.Statsd.Count(1, url+".200")
		jsonResponse(w, posts, 200)
	}
}

func putNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.networks.%s.put", vars["network"])
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Statsd.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Statsd.Count(1, url+".400")
			jsonErr(w, err, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		url := r.FormValue("url")
		err = api.UserSetNetworkImage(userID, netID, url)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Statsd.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Statsd.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		group, err := api.UserGetNetwork(userID, netID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Statsd.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Statsd.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
			return
		}
		go api.Statsd.Count(1, url+".200")
		jsonResponse(w, group, 200)
	}
}

func deleteUserNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.profile.networks.%s.delete", vars["networks"])
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Statsd.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			go api.Statsd.Count(1, url+".400")
			jsonErr(w, err, 400)
			return
		}
		netID := gp.NetworkID(_netID)
		err = api.UserLeaveGroup(userID, netID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				go api.Statsd.Count(1, url+".403")
				jsonResponse(w, e, 403)
			} else {
				go api.Statsd.Count(1, url+".500")
				jsonErr(w, err, 500)
			}
		}
		go api.Statsd.Count(1, url+".204")
		w.WriteHeader(204)
	}
}

func postNetworkRequests(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, _ := strconv.ParseUint(vars["network"], 10, 64)
		netID := gp.NetworkID(_netID)
		err = api.UserRequestAccess(userID, netID)
		if err != nil {
			switch {
			case err == lib.ENOTALLOWED:
				jsonResponse(w, err, 403)
			case err == lib.NoSuchNetwork:
				jsonResponse(w, err, 404)
			default:
				jsonErr(w, err, 500)
			}
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(204)
	}
}

func getNetworkRequests(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, _ := strconv.ParseUint(vars["network"], 10, 64)
		netID := gp.NetworkID(_netID)
		requests, err := api.NetworkRequests(userID, netID)
		switch {
		case err == lib.ENOTALLOWED:
			jsonResponse(w, err, 403)
		case err == lib.NoSuchNetwork:
			jsonResponse(w, err, 404)
		case err != nil:
			jsonErr(w, err, 500)
		default:
			jsonResponse(w, requests, 200)
		}
	}
}

func getNetworks(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		index, _ := strconv.ParseInt(r.FormValue("start"), 10, 64)
		filter := r.FormValue("filter")
		groups, err := api.GroupsByMembershipCount(userID, index, api.Config.GroupPageSize, filter)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, groups, 200)
	}
}

func deleteNetworkRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		_netID, _ := strconv.ParseUint(vars["network"], 10, 64)
		netID := gp.NetworkID(_netID)
		_requestor, _ := strconv.ParseUint(vars["user"], 10, 64)
		requestorID := gp.UserID(_requestor)
		err := api.RejectNetworkRequest(userID, netID, requestorID)
		if err != nil {
			switch {
			case err == lib.ENOTALLOWED || err == lib.AlreadyRejected || err == lib.AlreadyAccepted:
				jsonResponse(w, err, 403)
			case err == lib.NoSuchNetwork || err == lib.NoSuchRequest:
				jsonResponse(w, err, 404)
			default:
				jsonResponse(w, err, 500)
			}
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(204)
	}
}

func publicGetUniversity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_netID, _ := strconv.ParseUint(vars["network"], 10, 64)
	netID := gp.NetworkID(_netID)
	university, err := api.PublicUniversity(netID)
	switch {
	case err == lib.ENOTALLOWED:
		jsonResponse(w, err, 403)
	case err != nil:
		jsonResponse(w, err, 500)
	default:
		jsonResponse(w, university, 200)
	}
}

func muteGroupBadge(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		t := time.Now().UTC()
		err = api.UserMuteGroupBadge(userID, t)
		if err != nil {
			jsonResponse(w, err, 500)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(204)
	}
}
