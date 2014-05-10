package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

var (
	config     *gp.Config
	configLock = new(sync.RWMutex)
	api        *lib.API
)

var ETOOFEW = gp.APIerror{Reason: "Must have at least one valid recipient."}
var ETOOMANY = gp.APIerror{Reason: "Cannot send a message to more than 10 recipients"}
var EBADINPUT = gp.APIerror{Reason: "Missing parameter: first / last"}
var EBADTOKEN = gp.APIerror{Reason: "Invalid credentials"}
var EUNSUPPORTED = gp.APIerror{Reason: "Method not supported"}
var ENOTFOUND = gp.APIerror{Reason: "404 not found"}
var MissingParameterFirst = missingParamErr("first")
var MissingParameterLast = missingParamErr("last")
var MissingParameterPass = missingParamErr("pass")
var MissingParameterEmail = missingParamErr("email")
var MissingParameterAccepted = missingParamErr("accepted")
var InvalidEmail = gp.APIerror{Reason: "Invalid Email"}
var BadLogin = gp.APIerror{Reason: "Bad username/password"}
var NoSuchUpload = gp.APIerror{Reason: "That upload doesn't exist"}

func missingParamErr(param string) *gp.APIerror {
	return &gp.APIerror{Reason: "Missing parameter: " + param}
}

type Status struct {
	Status string `json:"status"`
	Email  string `json:"email"`
}

func NewStatus(status, email string) *Status {
	return &Status{Status: status, Email: email}
}

func init() {
	configInit()
	config = GetConfig()
	api = lib.New(*config)
	go api.FeedbackDaemon(60)
	go api.EndOldConversations()
	api.PeriodicSummary(time.Date(2014, time.April, 9, 8, 0, 0, 0, time.UTC), time.Duration(24*time.Hour))
}

//Note to self: validateToken should probably return an error at some point
func authenticate(r *http.Request) (userId gp.UserId, err error) {
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	userId = gp.UserId(id)
	token := r.FormValue("token")
	if len(token) == 0 {
		credentialsFromHeader := strings.Split(r.Header.Get("X-GP-Auth"), "-")
		id, _ = strconv.ParseUint(credentialsFromHeader[0], 10, 64)
		userId = gp.UserId(id)
		if len(credentialsFromHeader) == 2 {
			token = credentialsFromHeader[1]
		}
	}
	success := api.ValidateToken(userId, token)
	if success {
		return userId, nil
	} else {
		return 0, &EBADTOKEN
	}
}

func jsonResponse(w http.ResponseWriter, resp interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	marshaled, err := json.Marshal(resp)
	if err != nil {
		marshaled, _ = json.Marshal(gp.APIerror{Reason: err.Error()})
		w.WriteHeader(500)
		w.Write(marshaled)
	} else {
		w.WriteHeader(code)
		w.Write(marshaled)
	}
}

func jsonErr(w http.ResponseWriter, err error, code int) {
	switch err.(type) {
	case gp.APIerror:
		jsonResponse(w, err, code)
	default:
		jsonResponse(w, gp.APIerror{Reason: err.Error()}, code)
	}
}

/*********************************************************************************

Begin http handlers!

*********************************************************************************/

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
		jsonResponse(w, MissingParameterFirst, 400)
	case len(last) < 1:
		jsonResponse(w, MissingParameterLast, 400)
	case len(pass) == 0:
		jsonResponse(w, MissingParameterPass, 400)
	case len(email) == 0:
		jsonResponse(w, MissingParameterEmail, 400)
	default:
		validates, err := api.ValidateEmail(email)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		if !validates {
			jsonResponse(w, InvalidEmail, 400)
			return
		}
		rand, _ := lib.RandomString()
		user := first + "." + last + rand
		created, err := api.RegisterUser(user, pass, email, first, last, invite)
		if err != nil {
			_, ok := err.(gp.APIerror)
			if ok { //Duplicate user/email or password too short
				jsonResponse(w, err, 400)
			} else {
				jsonErr(w, err, 500)
			}
		} else {
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
	email := r.FormValue("email")
	pass := r.FormValue("pass")
	id, err := api.ValidatePass(email, pass)
	switch {
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	case err == nil:
		verified, err := api.IsVerified(id)
		switch {
		case err != nil:
			jsonErr(w, err, 500)
		case !verified:
			resp := NewStatus("unverified", email)
			jsonResponse(w, resp, 403)
		default:
			token, err := api.CreateAndStoreToken(id)
			if err == nil {
				jsonResponse(w, token, 200)
			} else {
				jsonErr(w, err, 500)
			}
		}
	default:
		jsonResponse(w, BadLogin, 400)
	}
}

func getPosts(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
		filter := r.FormValue("filter")
		vars := mux.Vars(r)
		id, ok := vars["network"]
		var network gp.NetworkId
		switch {
		case ok:
			_network, err := strconv.ParseUint(id, 10, 64)
			if err != nil {
				jsonErr(w, err, 500)
				return
			}
			network = gp.NetworkId(_network)
		default: //We haven't been given a network, which means this handler is being called by /posts and we just want the users' default network
			networks, err := api.GetUserNetworks(userId)
			if err != nil {
				jsonErr(w, err, 500)
				return
			}
			network = networks[0].Id
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
		var posts []gp.PostSmall
		posts, err = api.UserGetNetworkPosts(userId, network, mode, index, api.Config.PostPageSize, filter)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
			return
		}
		if len(posts) == 0 {
			// this is an ugly hack. But I can't immediately
			// think of a neater way to fix this
			// (json.Marshal(empty slice) returns null rather than
			// empty array ([]) which it obviously should
			jsonResponse(w, []string{}, 200)
		} else {
			jsonResponse(w, posts, 200)
		}
	}
}

func ignored(key string) bool {
	keys := []string{"id", "token", "text", "url", "tags", "popularity"}
	for _, v := range keys {
		if key == v {
			return true
		}
	}
	return false
}

func postPosts(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		text := r.FormValue("text")
		url := r.FormValue("url")
		tags := r.FormValue("tags")
		attribs := make(map[string]string)
		for k, v := range r.Form {
			if !ignored(k) {
				attribs[k] = strings.Join(v, "")
			}
		}
		var postId gp.PostId
		var ts []string
		if len(tags) > 1 {
			ts = strings.Split(tags, ",")
		}
		n, ok := vars["network"]
		var network gp.NetworkId
		if !ok {
			networks, err := api.GetUserNetworks(userId)
			if err != nil {
				jsonErr(w, err, 500)
				return
			}
			network = networks[0].Id
		} else {
			_network, err := strconv.ParseUint(n, 10, 64)
			if err != nil {
				jsonErr(w, err, 500)
				return
			}
			network = gp.NetworkId(_network)
		}
		switch {
		case len(url) > 5:
			postId, err = api.AddPostWithImage(userId, network, text, attribs, url, ts...)
		default:
			postId, err = api.AddPost(userId, network, text, attribs, ts...)
		}
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
		} else {
			jsonResponse(w, &gp.Created{Id: uint64(postId)}, 201)
		}
	}
}

func liveConversationHandler(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		conversations, err := api.GetLiveConversations(userId)
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
	userId, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
	if err != nil {
		start = 0
	}
	conversations, err := api.GetConversations(userId, start, api.Config.ConversationPageSize)
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
	userId, err := authenticate(r)
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
		conversation, err = api.CreateRandomConversation(userId, int(partners), true)
	} else {
		idstring := r.FormValue("participants")
		ids := strings.Split(idstring, ",")
		user_ids := make([]gp.UserId, 0, 10)
		for _, _id := range ids {
			id, err := strconv.ParseUint(_id, 10, 64)
			if err == nil {
				user_ids = append(user_ids, gp.UserId(id))
			}
		}
		switch {
		case len(user_ids) < 1:
			jsonResponse(w, &ETOOFEW, 400)
			return
		case len(user_ids) > 10:
			jsonResponse(w, &ETOOMANY, 400)
			return
		default:
			conversation, err = api.CreateConversationWith(userId, user_ids, false)
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
	userId, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	vars := mux.Vars(r)
	_convId, _ := strconv.ParseInt(vars["id"], 10, 64)
	convId := gp.ConversationId(_convId)
	start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
	if err != nil {
		start = 0
	}
	conv, err := api.UserGetConversation(userId, convId, start, api.Config.MessagePageSize)
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
	userId, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	vars := mux.Vars(r)
	_convId, _ := strconv.ParseInt(vars["id"], 10, 64)
	convId := gp.ConversationId(_convId)
	expires, err := strconv.ParseBool(r.FormValue("expiry"))
	if err != nil {
		jsonErr(w, err, 400)
		return
	}
	if expires == false {
		err = api.UserDeleteExpiry(userId, convId)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
				return
			}
			jsonErr(w, err, 500)
			return
		}
		conversation, err := api.GetConversation(userId, convId)
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
	userId, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	vars := mux.Vars(r)
	_convId, _ := strconv.ParseInt(vars["id"], 10, 64)
	convId := gp.ConversationId(_convId)
	err = api.UserEndConversation(userId, convId)
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
	userId, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	vars := mux.Vars(r)
	_convId, _ := strconv.ParseUint(vars["id"], 10, 64)
	convId := gp.ConversationId(_convId)
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
		messages, err = api.UserGetMessages(userId, convId, after, "after", api.Config.MessagePageSize)
	case before > 0:
		messages, err = api.UserGetMessages(userId, convId, before, "before", api.Config.MessagePageSize)
	default:
		messages, err = api.UserGetMessages(userId, convId, start, "start", api.Config.MessagePageSize)
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
	userId, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	vars := mux.Vars(r)
	_convId, _ := strconv.ParseUint(vars["id"], 10, 64)
	convId := gp.ConversationId(_convId)
	text := r.FormValue("text")
	messageId, err := api.AddMessage(convId, userId, text)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			jsonResponse(w, e, 403)
			return
		}
		jsonErr(w, err, 500)
	} else {
		jsonResponse(w, &gp.Created{Id: uint64(messageId)}, 201)
	}
}

func putMessages(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
	vars := mux.Vars(r)
	_convId, _ := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		jsonErr(w, err, 400)
	}
	convId := gp.ConversationId(_convId)
	_upTo, err := strconv.ParseUint(r.FormValue("seen"), 10, 64)
	if err != nil {
		_upTo = 0
	}
	upTo := gp.MessageId(_upTo)
	err = api.MarkConversationSeen(userId, convId, upTo)
	if err != nil {
		jsonErr(w, err, 500)
	} else {
		conversation, err := api.GetConversation(userId, convId)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, conversation, 200)
	}
}

func getComments(w http.ResponseWriter, r *http.Request) {
	_, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		postId := gp.PostId(_id)
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
		if err != nil {
			start = 0
		}
		comments, err := api.GetComments(postId, start, api.Config.CommentPageSize)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			if len(comments) == 0 {
				// this is an ugly hack. But I can't immediately
				// think of a neater way to fix this
				// (json.Marshal(empty slice) returns "null" rather than
				// empty array "[]" which it obviously should
				jsonResponse(w, []string{}, 200)
			} else {
				jsonResponse(w, comments, 200)
			}
		}
	}
}

func postComments(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		postId := gp.PostId(_id)
		text := r.FormValue("text")
		commentId, err := api.CreateComment(postId, userId, text)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			jsonResponse(w, &gp.Created{Id: uint64(commentId)}, 201)
		}
	}
}

func getPost(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		postId := gp.PostId(_id)
		post, err := api.UserGetPost(userId, postId)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
			return
		} else {
			jsonResponse(w, post, 200)
		}
	}
}

func postImages(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		postId := gp.PostId(_id)
		url := r.FormValue("url")
		exists, err := api.UserUploadExists(userId, url)
		if exists && err == nil {
			err := api.AddPostImage(postId, url)
			if err != nil {
				jsonErr(w, err, 500)
			} else {
				images := api.GetPostImages(postId)
				jsonResponse(w, images, 201)
			}
		} else {
			jsonErr(w, NoSuchUpload, 400)
		}
	}
}

func postLikes(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		postId := gp.PostId(_id)
		liked, err := strconv.ParseBool(r.FormValue("liked"))
		switch {
		case err != nil:
			jsonErr(w, err, 400)
		case liked:
			err = api.AddLike(userId, postId)
			if err != nil {
				e, ok := err.(*gp.APIerror)
				if ok && *e == lib.ENOTALLOWED {
					jsonResponse(w, e, 403)
				} else {
					jsonErr(w, err, 500)
				}
			} else {
				jsonResponse(w, gp.Liked{Post: postId, Liked: true}, 200)
			}
		default:
			err = api.DelLike(userId, postId)
			if err != nil {
				jsonErr(w, err, 500)
			} else {
				jsonResponse(w, gp.Liked{Post: postId, Liked: false}, 200)
			}
		}
	}
}

func getUser(w http.ResponseWriter, r *http.Request) {
	_, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		profileId := gp.UserId(_id)
		user, err := api.GetProfile(profileId)
		if err != nil {
			if err == gp.ENOSUCHUSER {
				jsonErr(w, err, 404)
				return
			}
			jsonErr(w, err, 500)
		} else {
			jsonResponse(w, user, 200)
		}
	}
}

func getUserPosts(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		profileId := gp.UserId(_id)
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
		var index int64
		var mode int
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
		if err != nil {
			after = 0
		}
		posts, err := api.GetUserPosts(profileId, userId, mode, index, api.Config.PostPageSize, r.FormValue("filter"))
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		if len(posts) == 0 {
			// this is an ugly hack. But I can't immediately
			// think of a neater way to fix this
			// (json.Marshal(empty slice) returns null rather than
			// empty array ([]) which it obviously should
			jsonResponse(w, []string{}, 200)
			return
		}
		jsonResponse(w, posts, 200)
	}
}

func longPollHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "GET":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		//awaitOneMessage will block until a message arrives over redis
		message := api.AwaitOneMessage(userId)
		w.Write(message)
	}
}

func contactsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		contacts, err := api.GetContacts(userId)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			if len(contacts) == 0 {
				jsonResponse(w, []string{}, 200)
			} else {
				jsonResponse(w, contacts, 200)
			}
		}
	case r.Method == "POST":
		_otherId, _ := strconv.ParseUint(r.FormValue("user"), 10, 64)
		otherId := gp.UserId(_otherId)
		contact, err := api.AddContact(userId, otherId)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			jsonResponse(w, contact, 201)
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func contactHandler(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	vars := mux.Vars(r)
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	contactId := gp.UserId(_id)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "PUT":
		accepted, err := strconv.ParseBool(r.FormValue("accepted"))
		if err != nil {
			accepted = false
		}
		if accepted {
			contact, err := api.AcceptContact(userId, contactId)
			if err != nil {
				jsonErr(w, err, 500)
			} else {
				jsonResponse(w, contact, 200)
			}
		} else {
			jsonResponse(w, MissingParameterAccepted, 400)
		}
	case r.Method == "DELETE":
		//Implement refusing requests
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func postDevice(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		deviceType := r.FormValue("type")
		deviceId := r.FormValue("device_id")
		log.Println("Device:", deviceType, deviceId)
		device, err := api.AddDevice(userId, deviceType, deviceId)
		log.Println(device, err)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			jsonResponse(w, device, 201)
		}
	case r.Method == "GET":
		//implement getting tokens
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func deleteDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Println("Delete device hit")
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, EBADTOKEN, 400)
		log.Println("Bad auth")
	case r.Method == "DELETE":
		vars := mux.Vars(r)
		err := api.DeleteDevice(userId, vars["id"])
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		w.WriteHeader(204)
		return
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}

}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		file, header, err := r.FormFile("image")
		if err != nil {
			jsonErr(w, err, 400)
		} else {
			defer file.Close()
			url, err := api.StoreFile(userId, file, header)
			if err != nil {
				jsonErr(w, err, 400)
			} else {
				jsonResponse(w, gp.URLCreated{URL: url}, 201)
			}
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func profileImageHandler(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		url := r.FormValue("url")
		exists, err := api.UserUploadExists(userId, url)
		if err != nil {
			jsonErr(w, err, 400)
			return
		}
		if !exists {
			jsonResponse(w, NoSuchUpload, 400)
		} else {
			err = api.SetProfileImage(userId, url)
			if err != nil {
				jsonErr(w, err, 500)
			} else {
				user, err := api.GetProfile(userId)
				if err != nil {
					jsonErr(w, err, 500)
				}
				jsonResponse(w, user, 200)
			}
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func busyHandler(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		status, err := strconv.ParseBool(r.FormValue("status"))
		if err != nil {
			jsonResponse(w, gp.APIerror{Reason: "Bad input"}, 400)
		}
		err = api.SetBusyStatus(userId, status)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			jsonResponse(w, &gp.BusyStatus{Busy: status}, 200)
		}
	case r.Method == "GET":
		status, err := api.BusyStatus(userId)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, &gp.BusyStatus{Busy: status}, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func changePassHandler(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		oldPass := r.FormValue("old")
		newPass := r.FormValue("new")
		err := api.ChangePass(userId, oldPass, newPass)
		if err != nil {
			//Assuming that most errors will be bad input for now
			jsonErr(w, err, 400)
			return
		}
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func changeNameHandler(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		firstName := r.FormValue("first")
		lastName := r.FormValue("last")
		err := api.SetUserName(userId, firstName, lastName)
		if err != nil {
			jsonResponse(w, &EBADINPUT, 400)
			return
		}
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func notificationHandler(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "PUT":
		_upTo, err := strconv.ParseUint(r.FormValue("seen"), 10, 64)
		if err != nil {
			_upTo = 0
		}
		includeSeen, _ := strconv.ParseBool(r.FormValue("include_seen"))
		notificationId := gp.NotificationId(_upTo)
		err = api.MarkNotificationsSeen(userId, notificationId)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			notifications, err := api.GetUserNotifications(userId, includeSeen)
			if err != nil {
				jsonErr(w, err, 500)
			} else {
				if len(notifications) == 0 {
					jsonResponse(w, []string{}, 200)
				} else {
					jsonResponse(w, notifications, 200)
				}
			}
		}
	case r.Method == "GET":
		includeSeen, _ := strconv.ParseBool(r.FormValue("include_seen"))
		notifications, err := api.GetUserNotifications(userId, includeSeen)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			if len(notifications) == 0 {
				jsonResponse(w, []string{}, 200)
			} else {
				jsonResponse(w, notifications, 200)
			}
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func facebookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		_fbToken := r.FormValue("token")
		email := r.FormValue("email")
		invite := r.FormValue("invite")
		//Is this a valid facebook token for this app?
		fbToken, err := api.FBValidateToken(_fbToken)
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
			} else {
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
				status := struct {
					Status string `json:"status"`
				}{"registered"}
				jsonResponse(w, status, 200)
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
				jsonResponse(w, Status{"unverified", storedEmail}, 201)
				return
			} else {
				status := Status{"registered", storedEmail}
				jsonResponse(w, status, 200)
				return
			}
		case len(email) < 3 && (err != nil):
			jsonResponse(w, gp.APIerror{Reason: "Email required"}, 400)
		}
	} else {
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func verificationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		vars := mux.Vars(r)
		err := api.Verify(vars["token"])
		if err != nil {
			log.Println(err)
			jsonResponse(w, gp.APIerror{Reason: "Bad verification token"}, 400)
			return
		}
		jsonResponse(w, struct {
			Verified bool `json:"verified"`
		}{true}, 200)
		return
	} else {
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func jsonServer(ws *websocket.Conn) {
	r := ws.Request()
	defer ws.Close()
	userId, err := authenticate(r)
	if err != nil {
		ws.Write([]byte(err.Error()))
		return
	}
	//Change this. 12/12/13
	chans := lib.ConversationChannelKeys([]gp.User{gp.User{Id: userId}})
	chans = append(chans, lib.NotificationChannelKey(userId))
	events := api.EventSubscribe(chans)
	for {
		message, ok := <-events.Messages
		if !ok {
			log.Println("Message channel is closed...")
			ws.Close()
			return
		}
		n, err := ws.Write(message)
		if err != nil {
			log.Println("Saw an error: ", err)
			events.Commands <- gp.QueueCommand{Command: "UNSUBSCRIBE", Value: ""}
			close(events.Commands)
			return
		}
		log.Println("Sent bytes: ", n)
	}
}

func requestResetHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "POST":
		email := r.FormValue("email")
		err := api.RequestReset(email)
		if err != nil {
			jsonErr(w, err, 400)
			return
		}
		w.WriteHeader(204)
		return
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func resetPassHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "POST":
		vars := mux.Vars(r)
		id, err := strconv.ParseUint(vars["id"], 10, 64)
		if err != nil {
			jsonErr(w, err, 400)
			return
		}
		userId := gp.UserId(id)
		pass := r.FormValue("pass")
		err = api.ResetPass(userId, vars["token"], pass)
		if err != nil {
			jsonErr(w, err, 400)
			return
		}
		w.WriteHeader(204)
		return
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func resendVerificationHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "POST":
		email := r.FormValue("email")
		userId, err := api.UserWithEmail(email)
		if err != nil {
			fbid, err := api.FBUserWithEmail(email)
			if err == nil {
				jsonErr(w, err, 400)
				return
			}
			api.FBissueVerification(fbid)
		} else {
			user, err := api.GetUser(userId)
			if err != nil {
				jsonErr(w, err, 500)
				return
			}
			api.GenerateAndSendVerification(userId, user.Name, email)
		}
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func inviteMessageHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		resp := struct {
			Message string `json:"message"`
		}{"Check out gleepost! https://gleepost.com"}
		jsonResponse(w, resp, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func liveHandler(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		after := r.FormValue("after")
		posts, err := api.UserGetLive(userId, after, api.Config.PostPageSize)
		if err != nil {
			code := 500
			if err == lib.EBADTIME {
				code = 400
			}
			jsonErr(w, err, code)
			return
		}
		if len(posts) == 0 {
			// this is an ugly hack. But I can't immediately
			// think of a neater way to fix this
			// (json.Marshal(empty slice) returns null rather than
			// empty array ([]) which it obviously should
			jsonResponse(w, []string{}, 200)
			return
		}
		jsonResponse(w, posts, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func attendHandler(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	vars := mux.Vars(r)
	//We can safely ignore this error since vars["id"] matches a numeric regex
	//... maybe. What if it's bigger than max(uint64) ??
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	post := gp.PostId(_id)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		//Implement
	case r.Method == "POST":
		//For now, assume that err is because the user specified a bad post.
		//Could also be a db error.
		err := api.UserAttend(post, userId, true)
		if err != nil {
			jsonResponse(w, err, 400)
		}
		w.WriteHeader(204)
	case r.Method == "DELETE":
		//For now, assume that err is because the user specified a bad post.
		//Could also be a db error.
		err := api.UserAttend(post, userId, false)
		if err != nil {
			jsonResponse(w, err, 400)
		}
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func userAttending(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		events, err := api.UserAttends(userId)
		if err != nil {
			jsonResponse(w, err, 500)
		}
		if len(events) == 0 {
			// this is an ugly hack. But I can't immediately
			// think of a neater way to fix this
			// (json.Marshal(empty slice) returns null rather than
			// empty array ([]) which it obviously should
			jsonResponse(w, []string{}, 200)
			return
		}
		jsonResponse(w, events, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func readAll(w http.ResponseWriter, r *http.Request) {
	log.Println("Someone hit readAll")
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		err = api.MarkAllConversationsSeen(userId)
		if err != nil {
			jsonResponse(w, err, 500)
		}
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func unread(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case userId != 2:
		jsonResponse(w, gp.APIerror{Reason: "Not allowed"}, 403)
	case r.Method == "GET":
		vars := mux.Vars(r)
		_uid, _ := strconv.ParseInt(vars["id"], 10, 64)
		uid := gp.UserId(_uid)
		count, err := api.UnreadMessageCount(uid)
		if err != nil {
			jsonResponse(w, err, 500)
		}
		jsonResponse(w, count, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func totalLiveConversations(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case userId != 2:
		jsonResponse(w, gp.APIerror{Reason: "Not allowed"}, 403)
	case r.Method == "GET":
		vars := mux.Vars(r)
		_uid, _ := strconv.ParseInt(vars["id"], 10, 64)
		uid := gp.UserId(_uid)
		count, err := api.TotalLiveConversations(uid)
		if err != nil {
			jsonResponse(w, err, 500)
		}
		jsonResponse(w, count, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func getGroups(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		networks, err := api.GetUserGroups(userId)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		if len(networks) == 0 {
			// this is an ugly hack. But I can't immediately
			// think of a neater way to fix this
			// (json.Marshal(empty slice) returns null rather than
			// empty array ([]) which it obviously should
			jsonResponse(w, []string{}, 200)
			return
		}
		jsonResponse(w, networks, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func getNetwork(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		vars := mux.Vars(r)
		_netId, err := strconv.ParseUint(vars["network"], 10, 16)
		if err != nil {
			jsonErr(w, err, 400)
			return
		}
		netId := gp.NetworkId(_netId)
		network, err := api.UserGetNetwork(userId, netId)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
			return
		}
		jsonResponse(w, network, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func postNetworks(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		name := r.FormValue("name")
		url := r.FormValue("url")
		desc := r.FormValue("desc")
		switch {
		case len(name) == 0:
			jsonResponse(w, missingParamErr("name"), 400)
		default:
			network, err := api.CreateGroup(userId, name, url, desc)
			if err != nil {
				e, ok := err.(*gp.APIerror)
				if ok && *e == lib.ENOTALLOWED {
					jsonResponse(w, e, 403)
				} else {
					jsonErr(w, err, 500)
				}
				return
			}
			jsonResponse(w, network, 201)
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func postNetworkUsers(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		vars := mux.Vars(r)
		_netId, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			jsonErr(w, err, 400)
			return
		}
		netId := gp.NetworkId(_netId)
		_users := strings.Split(r.FormValue("users"), ",")
		_fbUsers := strings.Split(r.FormValue("fbusers"), ",")
		var fbusers []uint64
		var users []gp.UserId
		for _, u := range _users {
			user, err := strconv.ParseUint(u, 10, 64)
			if err == nil {
				users = append(users, gp.UserId(user))
			}
		}
		for _, f := range _fbUsers {
			fbuser, err := strconv.ParseUint(f, 10, 64)
			if err == nil {
				fbusers = append(fbusers, fbuser)
			}
		}
		switch {
		case len(users) > 0:
			_, err = api.UserAddUsersToGroup(userId, users, netId)
		case len(fbusers) > 0:
			_, err = api.UserAddFBUsersToGroup(userId, fbusers, netId)
		case len(r.FormValue("email")) > 5:
			err = api.UserInviteEmail(userId, netId, r.FormValue("email"))
		default:
			jsonResponse(w, gp.APIerror{Reason: "Must add either user(s), facebook user(s) or an email"}, 400)
			return
		}
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
			return
		}
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func getNetworkUsers(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		vars := mux.Vars(r)
		_netId, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			jsonErr(w, err, 400)
			return
		}
		netId := gp.NetworkId(_netId)
		users, err := api.UserGetGroupMembers(userId, netId)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
		}
		if len(users) == 0 {
			// this is an ugly hack. But I can't immediately
			// think of a neater way to fix this
			// (json.Marshal(empty slice) returns null rather than
			// empty array ([]) which it obviously should
			jsonResponse(w, []string{}, 200)
			return
		}
		jsonResponse(w, users, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func deleteUserNetwork(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "DELETE":
		vars := mux.Vars(r)
		_netId, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			jsonErr(w, err, 400)
			return
		}
		netId := gp.NetworkId(_netId)
		err = api.UserLeaveGroup(userId, netId)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
		}
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func searchUsers(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		vars := mux.Vars(r)
		query := strings.Split(vars["query"], " ")
		for i := range query {
			query[i] = strings.TrimSpace(query[i])
		}
		networks, err := api.GetUserNetworks(userId)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		users, err := api.UserSearchUsersInNetwork(userId, query[0], strings.Join(query[1:], " "), networks[0].Id)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			switch {
			case !ok:
				jsonErr(w, err, 500)
			case *e == lib.ENOTALLOWED:
				jsonResponse(w, e, 403)
			case *e == lib.ETOOSHORT:
				jsonResponse(w, e, 400)
			default:
				jsonErr(w, err, 500)
			}
			return
		}
		if len(users) == 0 {
			// this is an ugly hack. But I can't immediately
			// think of a neater way to fix this
			// (json.Marshal(empty slice) returns null rather than
			// empty array ([]) which it obviously should
			jsonResponse(w, []string{}, 200)
			return
		}
		jsonResponse(w, users, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

//getGroupPosts is basically the same goddamn thing as getPosts. stop copy-pasting you cretin.
func getGroupPosts(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
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
		posts, err := api.UserGetGroupsPosts(userId, mode, index, api.Config.PostPageSize, r.FormValue("filter"))
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
			return
		}
		if len(posts) == 0 {
			// this is an ugly hack. But I can't immediately
			// think of a neater way to fix this
			// (json.Marshal(empty slice) returns null rather than
			// empty array ([]) which it obviously should
			jsonResponse(w, []string{}, 200)
		} else {
			jsonResponse(w, posts, 200)
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func putNetwork(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "PUT":
		vars := mux.Vars(r)
		_netId, err := strconv.ParseUint(vars["network"], 10, 64)
		if err != nil {
			jsonErr(w, err, 400)
			return
		}
		netId := gp.NetworkId(_netId)
		url := r.FormValue("url")
		err = api.UserSetNetworkImage(userId, netId, url)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
			return
		}
		group, err := api.UserGetNetwork(userId, netId)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
			return
		}
		jsonResponse(w, group, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func deletePost(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		postId := gp.PostId(_id)
		err := api.UserDeletePost(userId, postId)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
			return
		} else {
			w.WriteHeader(204)
		}
	}
}

func facebookAssociate(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	pass := r.FormValue("pass")
	id, err := api.ValidatePass(email, pass)
	_fbToken := r.FormValue("fbtoken")
	//Is this a valid facebook token for this app?
	fbToken, errtoken := api.FBValidateToken(_fbToken)
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
				if token.UserId == userID {
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
			if token.UserId == id {
				//The facebook account is already associated with this gleepost account
				w.WriteHeader(204)
			} else {
				jsonResponse(w, gp.APIerror{Reason: "Facebook account already associated with another gleepost account..."}, 400)
			}

		}
	}
}

func getAttendees(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_postId, _ := strconv.ParseUint(vars["id"], 10, 64)
		postId := gp.PostId(_postId)
		attendees, err := api.UserGetEventAttendees(userId, postId)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
			return
		}
		popularity, attendee_count, err := api.UserGetEventPopularity(userId, postId)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
			return
		}
		resp := struct {
			Popularity    int       `json:"popularity"`
			AttendeeCount int       `json:"attendee_count"`
			Attendees     []gp.User `json:"attendees,omitempty"`
		}{Popularity: popularity, AttendeeCount: attendee_count, Attendees: attendees}
		jsonResponse(w, resp, 200)

	}
}

func putAttendees(w http.ResponseWriter, r *http.Request) {
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		attending, _ := strconv.ParseBool(r.FormValue("attending"))
		vars := mux.Vars(r)
		_postId, _ := strconv.ParseUint(vars["id"], 10, 64)
		postId := gp.PostId(_postId)
		err = api.UserAttend(postId, userId, attending)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
			return
		}
		getAttendees(w, r)
	}
}
