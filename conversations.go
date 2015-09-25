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
	base.Handle("/conversations/read_all", timeHandler(api, authenticated(readAll))).Methods("POST")
	base.Handle("/conversations/read_all", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/conversations/mute_badges", timeHandler(api, authenticated(muteBadges))).Methods("POST")
	base.Handle("/conversations/mute_badges", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/conversations", timeHandler(api, authenticated(getConversations))).Methods("GET")
	base.Handle("/conversations", timeHandler(api, authenticated(postConversations))).Methods("POST")
	base.Handle("/conversations", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/conversations", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/conversations/{id:[0-9]+}", timeHandler(api, authenticated(getSpecificConversation))).Methods("GET")
	base.Handle("/conversations/{id:[0-9]+}", timeHandler(api, authenticated(deleteSpecificConversation))).Methods("DELETE")
	base.Handle("/conversations/{id:[0-9]+}", timeHandler(api, authenticated(putConversation))).Methods("PUT")
	base.Handle("/conversations/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/conversations/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/conversations/{id:[0-9]+}/messages/search/{query}", timeHandler(api, authenticated(searchMessages))).Methods("GET")
	base.Handle("/conversations/{id:[0-9]+}/messages", timeHandler(api, authenticated(getMessages))).Methods("GET")
	base.Handle("/conversations/{id:[0-9]+}/messages", timeHandler(api, authenticated(postMessages))).Methods("POST")
	base.Handle("/conversations/{id:[0-9]+}/messages", timeHandler(api, authenticated(putMessages))).Methods("PUT")
	base.Handle("/conversations/{id:[0-9]+}/messages", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/conversations/{id:[0-9]+}/messages", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/conversations/{id:[0-9]+}/participants", timeHandler(api, authenticated(postParticipants))).Methods("POST")
	base.Handle("/conversations/{id:[0-9]+}/participants", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/conversations/{id:[0-9]+}/files", timeHandler(api, authenticated(getFiles))).Methods("GET")
	base.Handle("/conversations/{id:[0-9]+}/files", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func getConversations(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
	if err != nil {
		start = 0
	}
	conversations, err := api.GetConversations(userID, start, api.Config.ConversationPageSize)
	if err != nil {
		go api.Statsd.Count(1, "gleepost.conversations.get.500")
		jsonErr(w, err, 500)
	} else {
		go api.Statsd.Count(1, "gleepost.conversations.get.200")
		jsonResponse(w, conversations, 200)
	}
}

func postConversations(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	idstring := r.FormValue("participants")
	ids := strings.Split(idstring, ",")
	userIds := make([]gp.UserID, 0, 50)
	for _, _id := range ids {
		id, err := strconv.ParseUint(_id, 10, 64)
		if err == nil {
			userIds = append(userIds, gp.UserID(id))
		}
	}
	conversation, err := api.CreateConversationWith(userID, userIds)
	e, ok := err.(*gp.APIerror)
	switch {
	case ok && *e == gp.ENOSUCHUSER:
		go api.Statsd.Count(1, "gleepost.conversations.get.400")
		jsonResponse(w, e, 400)
	case ok && *e == lib.ENOTALLOWED:
		go api.Statsd.Count(1, "gleepost.conversations.get.403")
		jsonResponse(w, e, 403)
	case err != nil && (err == lib.ETOOMANY || err == lib.ETOOFEW):
		go api.Statsd.Count(1, "gleepost.conversations.get.400")
		jsonResponse(w, e, 400)
	case err != nil:
		go api.Statsd.Count(1, "gleepost.conversations.get.500")
		jsonErr(w, err, 500)
	default:
		go api.Statsd.Count(1, "gleepost.conversations.get.201")
		jsonResponse(w, conversation, 201)
	}
}

func maybeRedirect(w http.ResponseWriter, r *http.Request, conv gp.ConversationID, urlPattern string, statusCode int) bool {
	mergedID, err := api.ConversationMergedInto(conv)
	if err != nil {
		return false
	}
	url := fmt.Sprintf(urlPattern, mergedID)
	http.Redirect(w, r, url, statusCode)
	return true
}

func getSpecificConversation(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseInt(vars["id"], 10, 64)
	url := fmt.Sprintf("gleepost.conversations.%d.get", _convID)
	convID := gp.ConversationID(_convID)
	start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
	if err != nil {
		start = 0
	}
	conv, err := api.UserGetConversation(userID, convID, start, api.Config.MessagePageSize)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			if maybeRedirect(w, r, convID, "/api/v1/conversations/%d", 301) {
				go api.Statsd.Count(1, url+".301")
				return
			}
			go api.Statsd.Count(1, url+".403")
			jsonResponse(w, e, 403)
		} else {
			go api.Statsd.Count(1, url+".500")
			jsonErr(w, err, 500)
		}
		return
	}
	go api.Statsd.Count(1, url+".200")
	jsonResponse(w, conv, 200)
}

func deleteSpecificConversation(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseInt(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	url := fmt.Sprintf("gleepost.conversations.%d.delete", convID)
	err := api.UserDeleteConversation(userID, convID)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			if maybeRedirect(w, r, convID, "api/v1/conversations/%d", 301) {
				go api.Statsd.Count(1, url+".301")
				return
			}
			go api.Statsd.Count(1, url+".403")
			jsonResponse(w, e, 403)
			return
		}
		go api.Statsd.Count(1, url+".500")
		jsonErr(w, err, 500)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	go api.Statsd.Count(1, url+".204")
	w.WriteHeader(204)
}

func getMessages(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseUint(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	url := fmt.Sprintf("gleepost.conversations.%d.messages.get", convID)
	mode, index := interpretPagination(r.FormValue("start"), r.FormValue("before"), r.FormValue("after"))
	_count, _ := strconv.ParseInt(r.FormValue("count"), 10, 64)
	count := int(_count)
	messages, err := api.UserGetMessages(userID, convID, mode, index, count)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			if maybeRedirect(w, r, convID, "api/v1/conversations/%d/messages", 301) {
				go api.Statsd.Count(1, url+".301")
				return
			}
			go api.Statsd.Count(1, url+".403")
			jsonResponse(w, e, 403)
			return
		}
		go api.Statsd.Count(1, url+".500")
		jsonErr(w, err, 500)
	} else {
		go api.Statsd.Count(1, url+".200")
		jsonResponse(w, messages, 200)
	}
}

func postMessages(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseUint(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	url := fmt.Sprintf("gleepost.conversations.%d.messages.post", convID)
	text := r.FormValue("text")
	message, err := api.AddMessage(convID, userID, text)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			if maybeRedirect(w, r, convID, "api/v1/conversations/%d/messages", 301) {
				go api.Statsd.Count(1, url+".301")
				return
			}
			api.Statsd.Count(1, url+".403")
			jsonResponse(w, e, 403)
			return
		}
		go api.Statsd.Count(1, url+".500")
		jsonErr(w, err, 500)
	} else {
		go api.Statsd.Count(1, url+".201")
		jsonResponse(w, message, 201)
	}
}

func putMessages(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseUint(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	url := fmt.Sprintf("gleepost.conversations.%d.messages.put", convID)
	_upTo, err := strconv.ParseUint(r.FormValue("seen"), 10, 64)
	if err != nil {
		_upTo = 0
	}
	upTo := gp.MessageID(_upTo)
	err = api.MarkConversationSeen(userID, convID, upTo)
	if err != nil {
		if maybeRedirect(w, r, convID, "api/v1/conversations/%d/messages", 301) {
			go api.Statsd.Count(1, url+".301")
			return
		}
		go api.Statsd.Count(1, url+".500")
		jsonErr(w, err, 500)
	} else {
		conversation, err := api.GetConversation(userID, convID)
		if err != nil {
			go api.Statsd.Count(1, url+".500")
			jsonErr(w, err, 500)
			return
		}
		go api.Statsd.Count(1, url+".200")
		jsonResponse(w, conversation, 200)
	}
}

func readAll(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	err := api.MarkAllConversationsSeen(userID)
	if err != nil {
		go api.Statsd.Count(1, "gleepost.conversations.read_all.post.500")
		jsonResponse(w, err, 500)
		return
	}
	go api.Statsd.Count(1, "gleepost.conversations.read_all.post.204")
	w.WriteHeader(204)
}

func muteBadges(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	t := time.Now().UTC()
	err := api.UserMuteBadges(userID, t)
	if err != nil {
		go api.Statsd.Count(1, "gleepost.conversations.mute_badges.post.500")
		jsonResponse(w, err, 500)
		return
	}
	go api.Statsd.Count(1, "gleepost.conversations.mute_badges.post.204")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(204)
}

func postParticipants(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.conversations.%s.participants.post", vars["id"])
	_convID, _ := strconv.ParseUint(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	_users := strings.Split(r.FormValue("users"), ",")
	var users []gp.UserID
	for _, u := range _users {
		user, err := strconv.ParseUint(u, 10, 64)
		if err == nil {
			users = append(users, gp.UserID(user))
		}
	}
	participants, err := api.UserAddParticipants(userID, convID, users...)
	if err != nil {
		if maybeRedirect(w, r, convID, "api/v1/conversations/%d/messages", 301) {
			go api.Statsd.Count(1, url+".301")
			return
		}
		jsonErr(w, err, 400)
		go api.Statsd.Count(1, url+".400")
		return
	}
	jsonResponse(w, participants, 201)
	go api.Statsd.Count(1, url+".201")
}

func putConversation(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseInt(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	muted, _ := strconv.ParseBool(r.FormValue("muted"))
	err := api.SetMuteStatus(userID, convID, muted)
	switch {
	case err == lib.ENOTALLOWED:
		jsonErr(w, err, 403)
	case err != nil:
		jsonErr(w, err, 500)
	default:
		conv, err := api.UserGetConversation(userID, convID, 0, api.Config.MessagePageSize)
		if err != nil {
			jsonResponse(w, err, 500)
		}
		jsonResponse(w, conv, 200)
	}
}

func getFiles(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseInt(vars["id"], 10, 64)
	mode, index := interpretPagination(r.FormValue("start"), r.FormValue("before"), r.FormValue("after"))
	convID := gp.ConversationID(_convID)
	files, err := api.ConversationFiles(userID, convID, mode, index, api.Config.MessagePageSize)
	switch {
	case err == lib.ENOTALLOWED:
		jsonErr(w, err, 403)
	case err != nil:
		jsonErr(w, err, 500)
	default:
		jsonResponse(w, files, 200)
	}
}

func searchMessages(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseInt(vars["id"], 10, 64)
	mode, index := interpretPagination(r.FormValue("start"), r.FormValue("before"), r.FormValue("after"))
	convID := gp.ConversationID(_convID)
	results, err := api.SearchMessagesInConversation(userID, convID, vars["query"], mode, index)
	switch {
	case err == lib.ENOTALLOWED:
		jsonErr(w, err, 403)
	case err != nil:
		jsonErr(w, err, 500)
	default:
		jsonResponse(w, results, 200)
	}
}
