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

//ETOOFEW = You tried to create a conversation with 0 other participants (or you gave all invalid participants)
var ETOOFEW = gp.APIerror{Reason: "Must have at least one valid recipient."}

//ETOOMANY = You tried to create a conversation with a whole bunch of participants
var ETOOMANY = gp.APIerror{Reason: "Cannot send a message to more than 10 recipients"}

func init() {
	base.Handle("/conversations/live", timeHandler(api, http.HandlerFunc(goneHandler)))
	base.Handle("/conversations/read_all", timeHandler(api, http.HandlerFunc(readAll))).Methods("POST")
	base.Handle("/conversations/read_all/", timeHandler(api, http.HandlerFunc(readAll))).Methods("POST")
	base.Handle("/conversations/mute_badges", timeHandler(api, http.HandlerFunc(muteBadges))).Methods("POST")
	base.Handle("/conversations/mute_badges/", timeHandler(api, http.HandlerFunc(muteBadges))).Methods("POST")
	base.Handle("/conversations", timeHandler(api, http.HandlerFunc(getConversations))).Methods("GET")
	base.Handle("/conversations", timeHandler(api, http.HandlerFunc(postConversations))).Methods("POST")
	base.Handle("/conversations/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(getSpecificConversation))).Methods("GET")
	base.Handle("/conversations/{id:[0-9]+}/", timeHandler(api, http.HandlerFunc(getSpecificConversation))).Methods("GET")
	base.Handle("/conversations/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(goneHandler))).Methods("PUT")
	base.Handle("/conversations/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(deleteSpecificConversation))).Methods("DELETE")
	base.Handle("/conversations/{id:[0-9]+}/messages", timeHandler(api, http.HandlerFunc(getMessages))).Methods("GET")
	base.Handle("/conversations/{id:[0-9]+}/messages", timeHandler(api, http.HandlerFunc(postMessages))).Methods("POST")
	base.Handle("/conversations/{id:[0-9]+}/messages", timeHandler(api, http.HandlerFunc(putMessages))).Methods("PUT")
	base.Handle("/conversations/", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/conversations", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/conversations/{id}/messages", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/conversations/{id:[0-9]+}/participants", timeHandler(api, http.HandlerFunc(postParticipants))).Methods("POST")
}

func getConversations(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
	if err != nil {
		start = 0
	}
	conversations, err := api.GetConversations(userID, start, api.Config.ConversationPageSize)
	if err != nil {
		go api.Count(1, "gleepost.conversations.get.500")
		jsonErr(w, err, 500)
	} else {
		go api.Count(1, "gleepost.conversations.get.200")
		jsonResponse(w, conversations, 200)
	}
}

func postConversations(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	if err != nil {
		go api.Count(1, "gleepost.conversations.get.400")
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	var conversation gp.ConversationAndMessages
	idstring := r.FormValue("participants")
	ids := strings.Split(idstring, ",")
	userIds := make([]gp.UserID, 0, 50)
	for _, _id := range ids {
		id, err := strconv.ParseUint(_id, 10, 64)
		if err == nil {
			userIds = append(userIds, gp.UserID(id))
		}
	}
	switch {
	case len(userIds) < 1:
		go api.Count(1, "gleepost.conversations.get.400")
		jsonResponse(w, &ETOOFEW, 400)
		return
	case len(userIds) > 50:
		go api.Count(1, "gleepost.conversations.get.400")
		jsonResponse(w, &ETOOMANY, 400)
		return
	case len(userIds) == 1:
		conversation, err = api.CreateConversationWith(userID, true, userIds)
	default:
		conversation, err = api.CreateConversationWith(userID, false, userIds)
	}
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == gp.ENOSUCHUSER {
			go api.Count(1, "gleepost.conversations.get.400")
			jsonResponse(w, e, 400)
		} else if *e == lib.ENOTALLOWED {
			go api.Count(1, "gleepost.conversations.get.403")
			jsonResponse(w, e, 403)
		} else {
			go api.Count(1, "gleepost.conversations.get.500")
			jsonErr(w, err, 500)
		}
	} else {
		go api.Count(1, "gleepost.conversations.get.201")
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

func getSpecificConversation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseInt(vars["id"], 10, 64)
	url := fmt.Sprintf("gleepost.conversations.%d.get", _convID)
	userID, err := authenticate(r)
	if err != nil {
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
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
				go api.Count(1, url+".301")
				return
			}
			go api.Count(1, url+".403")
			jsonResponse(w, e, 403)
		} else {
			go api.Count(1, url+".500")
			jsonErr(w, err, 500)
		}
		return
	}
	go api.Count(1, url+".200")
	jsonResponse(w, conv, 200)
}

func deleteSpecificConversation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseInt(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	url := fmt.Sprintf("gleepost.conversations.%d.delete", convID)
	userID, err := authenticate(r)
	if err != nil {
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	err = api.UserDeleteConversation(userID, convID)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			if maybeRedirect(w, r, convID, "api/v1/conversations/%d", 301) {
				go api.Count(1, url+".301")
				return
			}
			go api.Count(1, url+".403")
			jsonResponse(w, e, 403)
			return
		}
		go api.Count(1, url+".500")
		jsonErr(w, err, 500)
		return
	}
	go api.Count(1, url+".204")
	w.WriteHeader(204)
}

func getMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseUint(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	url := fmt.Sprintf("gleepost.conversations.%d.messages.get", convID)
	userID, err := authenticate(r)
	if err != nil {
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
	if err != nil {
		start = 0
	}
	after, err := strconv.ParseInt(r.FormValue("after"), 10, 64)
	if err != nil {
		after = 0
	}
	before, err := strconv.ParseInt(r.FormValue("before"), 10, 64)
	if err != nil {
		before = 0
	}
	var messages []gp.Message
	switch {
	case after > 0:
		messages, err = api.UserGetMessages(userID, convID, after, "after", api.Config.MessagePageSize)
	case before > 0:
		messages, err = api.UserGetMessages(userID, convID, before, "before", api.Config.MessagePageSize)
	default:
		messages, err = api.UserGetMessages(userID, convID, start, "start", api.Config.MessagePageSize)
	}
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			if maybeRedirect(w, r, convID, "api/v1/conversations/%d/messages", 301) {
				go api.Count(1, url+".301")
				return
			}
			go api.Count(1, url+".403")
			jsonResponse(w, e, 403)
			return
		}
		go api.Count(1, url+".500")
		jsonErr(w, err, 500)
	} else {
		go api.Count(1, url+".200")
		jsonResponse(w, messages, 200)
	}
}

func postMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseUint(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	url := fmt.Sprintf("gleepost.conversations.%d.messages.post", convID)
	userID, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	text := r.FormValue("text")
	messageID, err := api.AddMessage(convID, userID, text)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			if maybeRedirect(w, r, convID, "api/v1/conversations/%d/messages", 301) {
				go api.Count(1, url+".301")
				return
			}
			api.Count(1, url+".403")
			jsonResponse(w, e, 403)
			return
		}
		go api.Count(1, url+".500")
		jsonErr(w, err, 500)
	} else {
		go api.Count(1, url+".201")
		jsonResponse(w, &gp.Created{ID: uint64(messageID)}, 201)
	}
}

func putMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseUint(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	url := fmt.Sprintf("gleepost.conversations.%d.messages.put", convID)
	userID, err := authenticate(r)
	if err != nil {
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	_upTo, err := strconv.ParseUint(r.FormValue("seen"), 10, 64)
	if err != nil {
		_upTo = 0
	}
	upTo := gp.MessageID(_upTo)
	err = api.MarkConversationSeen(userID, convID, upTo)
	if err != nil {
		if maybeRedirect(w, r, convID, "api/v1/conversations/%d/messages", 301) {
			go api.Count(1, url+".301")
			return
		}
		go api.Count(1, url+".500")
		jsonErr(w, err, 500)
	} else {
		conversation, err := api.GetConversation(userID, convID)
		if err != nil {
			go api.Count(1, url+".500")
			jsonErr(w, err, 500)
			return
		}
		go api.Count(1, url+".200")
		jsonResponse(w, conversation, 200)
	}
}

func readAll(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, "gleepost.conversations.read_all.post.400")
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		err = api.MarkAllConversationsSeen(userID)
		if err != nil {
			go api.Count(1, "gleepost.conversations.read_all.post.500")
			jsonResponse(w, err, 500)
			return
		}
		go api.Count(1, "gleepost.conversations.read_all.post.204")
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func muteBadges(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, "gleepost.conversations.mute_badges.post.400")
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		t := time.Now().UTC()
		err = api.UserMuteBadges(userID, t)
		if err != nil {
			go api.Count(1, "gleepost.conversations.mute_badges.post.500")
			jsonResponse(w, err, 500)
			return
		}
		go api.Count(1, "gleepost.conversations.mute_badges.post.204")
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func postParticipants(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("gleepost.conversations.%s.participants.post", vars["id"])
	userID, err := authenticate(r)
	switch {
	case err != nil:
		go api.Count(1, url+".400")
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
				go api.Count(1, url+".301")
				return
			}
			jsonErr(w, err, 400)
			go api.Count(1, url+".400")
			return
		}
		jsonResponse(w, participants, 201)
		go api.Count(1, url+".201")
	}
}
