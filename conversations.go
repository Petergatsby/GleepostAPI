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

//ETOOFEW = You tried to create a conversation with 0 other participants (or you gave all invalid participants)
var ETOOFEW = gp.APIerror{Reason: "Must have at least one valid recipient."}

//ETOOMANY = You tried to create a conversation with a whole bunch of participants
var ETOOMANY = gp.APIerror{Reason: "Cannot send a message to more than 10 recipients"}

func init() {
	base.HandleFunc("/conversations/live", liveConversationHandler)
	base.HandleFunc("/conversations/read_all", readAll).Methods("POST")
	base.HandleFunc("/conversations/read_all/", readAll).Methods("POST")
	base.HandleFunc("/conversations", getConversations).Methods("GET")
	base.HandleFunc("/conversations", postConversations).Methods("POST")
	base.HandleFunc("/conversations/{id:[0-9]+}", getSpecificConversation).Methods("GET")
	base.HandleFunc("/conversations/{id:[0-9]+}/", getSpecificConversation).Methods("GET")
	base.HandleFunc("/conversations/{id:[0-9]+}", putSpecificConversation).Methods("PUT")
	base.HandleFunc("/conversations/{id:[0-9]+}", deleteSpecificConversation).Methods("DELETE")
	base.HandleFunc("/conversations/{id:[0-9]+}/messages", getMessages).Methods("GET")
	base.HandleFunc("/conversations/{id:[0-9]+}/messages", postMessages).Methods("POST")
	base.HandleFunc("/conversations/{id:[0-9]+}/messages", putMessages).Methods("PUT")
	base.HandleFunc("/conversations/", optionsHandler).Methods("OPTIONS")
	base.HandleFunc("/conversations/{id}/messages", optionsHandler).Methods("OPTIONS")
}

func liveConversationHandler(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.conversations.live.get")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		conversations, err := api.GetLiveConversations(userID)
		switch {
		case err != nil:
			jsonErr(w, err, 500)
			return
		case len(conversations) == 0:
			jsonResponse(w, []string{}, 200)
		default:
			jsonResponse(w, conversations, 200)
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func getConversations(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.conversations.get")
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
		jsonErr(w, err, 500)
	} else {
		if len(conversations) == 0 {
			// this is an ugly hack. But I can't immediately
			// think of a neater way to fix this
			// (json.Marshal(empty slice) returns "null" rather than
			// empty array "[]" which it obviously should
			jsonResponse(w, []string{}, 200)
		} else {
			jsonResponse(w, conversations, 200)
		}
	}
}

func postConversations(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.conversations.post")
	userID, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	random, err := strconv.ParseBool(r.FormValue("random"))
	var conversation gp.Conversation
	if err != nil {
		random = true
		err = nil
	}
	if random {
		partners, err := strconv.ParseUint(r.FormValue("participant_count"), 10, 64)
		switch {
		case err != nil:
			partners = 2
		case partners > 4:
			partners = 4
		case partners < 2:
			partners = 2
		}
		conversation, err = api.CreateRandomConversation(userID, int(partners), true)
	} else {
		idstring := r.FormValue("participants")
		ids := strings.Split(idstring, ",")
		userIds := make([]gp.UserID, 0, 10)
		for _, _id := range ids {
			id, err := strconv.ParseUint(_id, 10, 64)
			if err == nil {
				userIds = append(userIds, gp.UserID(id))
			}
		}
		switch {
		case len(userIds) < 1:
			jsonResponse(w, &ETOOFEW, 400)
			return
		case len(userIds) > 10:
			jsonResponse(w, &ETOOMANY, 400)
			return
		default:
			conversation, err = api.CreateConversationWith(userID, userIds, false)
		}

	}
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == gp.ENOSUCHUSER {
			jsonResponse(w, e, 400)
		} else if *e == lib.ENOTALLOWED {
			jsonResponse(w, e, 403)
		} else {
			jsonErr(w, err, 500)
		}
	} else {
		jsonResponse(w, conversation, 201)
	}
}

func getSpecificConversation(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.conversations.*.get")
	userID, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseInt(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
	if err != nil {
		start = 0
	}
	conv, err := api.UserGetConversation(userID, convID, start, api.Config.MessagePageSize)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			jsonResponse(w, e, 403)
		} else {
			jsonErr(w, err, 500)
		}
		return
	}
	jsonResponse(w, conv, 200)
}

func putSpecificConversation(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.conversations.*.put")
	userID, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseInt(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	expires, err := strconv.ParseBool(r.FormValue("expiry"))
	if err != nil {
		jsonErr(w, err, 400)
		return
	}
	if expires == false {
		err = api.UserDeleteExpiry(userID, convID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
				return
			}
			jsonErr(w, err, 500)
			return
		}
		conversation, err := api.GetConversation(userID, convID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				//This should never happen but just in case the UserDeleteExpiry block above is changed...
				jsonResponse(w, e, 403)
				return
			}
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, conversation, 200)
	} else {
		jsonResponse(w, gp.APIerror{Reason: "Missing parameter:expiry"}, 400)
	}
}

func deleteSpecificConversation(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.conversations.*.delete")
	userID, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseInt(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	err = api.UserDeleteConversation(userID, convID)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			jsonResponse(w, e, 403)
			return
		}
		jsonErr(w, err, 500)
		return
	}
	w.WriteHeader(204)
}

func getMessages(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.conversations.*.messages")
	userID, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseUint(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
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
			jsonResponse(w, e, 403)
			return
		}
		jsonErr(w, err, 500)
	} else {
		if len(messages) == 0 {
			// this is an ugly hack. But I can't immediately
			// think of a neater way to fix this
			// (json.Marshal(empty slice) returns "null" rather than
			// empty array "[]" which it obviously should
			jsonResponse(w, []string{}, 200)
		} else {
			jsonResponse(w, messages, 200)
		}
	}
}

func postMessages(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.conversations.*.messages.post")
	userID, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseUint(vars["id"], 10, 64)
	convID := gp.ConversationID(_convID)
	text := r.FormValue("text")
	messageID, err := api.AddMessage(convID, userID, text)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			jsonResponse(w, e, 403)
			return
		}
		jsonErr(w, err, 500)
	} else {
		jsonResponse(w, &gp.Created{ID: uint64(messageID)}, 201)
	}
}

func putMessages(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.conversations.*.messages.put")
	userID, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	vars := mux.Vars(r)
	_convID, _ := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		jsonErr(w, err, 400)
	}
	convID := gp.ConversationID(_convID)
	_upTo, err := strconv.ParseUint(r.FormValue("seen"), 10, 64)
	if err != nil {
		_upTo = 0
	}
	upTo := gp.MessageID(_upTo)
	err = api.MarkConversationSeen(userID, convID, upTo)
	if err != nil {
		jsonErr(w, err, 500)
	} else {
		conversation, err := api.GetConversation(userID, convID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, conversation, 200)
	}
}

func readAll(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.conversations.read_all.post")
	log.Println("Someone hit readAll")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		err = api.MarkAllConversationsSeen(userID)
		if err != nil {
			jsonResponse(w, err, 500)
			return
		}
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}
