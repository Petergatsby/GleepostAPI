package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var (
	config     *gp.Config
	configLock = new(sync.RWMutex)
	api        *lib.API
)

var ETOOFEW = gp.APIerror{"Must have at least one valid recipient."}
var ETOOMANY = gp.APIerror{"Cannot send a message to more than 10 recipients"}
var EBADINPUT = gp.APIerror{"Missing parameter: first / last"}

func init() {
	configInit()
	config = GetConfig()
	api = lib.New(*config)
	go api.FeedbackDaemon(60)
	go api.EndOldConversations()
}

//Note to self: validateToken should probably return an error at some point
func authenticate(r *http.Request) (userId gp.UserId, err error) {
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	userId = gp.UserId(id)
	token := r.FormValue("token")
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
		marshaled, _ = json.Marshal(gp.APIerror{err.Error()})
		w.WriteHeader(500)
		w.Write(marshaled)
	} else {
		w.WriteHeader(code)
		w.Write(marshaled)
	}
}

var EBADTOKEN = gp.APIerror{"Invalid credentials"}
var EUNSUPPORTED = gp.APIerror{"Method not supported"}
var ENOTFOUND = gp.APIerror{"404 not found"}

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
	switch {
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
		//Note to future self : would be neater if
		//we returned _all_ errors not just the first
	case len(first) < 2:
		jsonResponse(w, gp.APIerror{"Missing parameter: first"}, 400)
	case len(last) < 1:
		jsonResponse(w, gp.APIerror{"Missing parameter: last"}, 400)
	case len(pass) == 0:
		jsonResponse(w, gp.APIerror{"Missing parameter: pass"}, 400)
	case len(email) == 0:
		jsonResponse(w, gp.APIerror{"Missing parameter: email"}, 400)
	default:
		validates, err := api.ValidateEmail(email)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
			return
		}
		if !validates {
			jsonResponse(w, gp.APIerror{"Invalid Email"}, 400)
			return
		}
		rand, _ := lib.RandomString()
		user := first + "." + last + rand
		id, err := api.RegisterUser(user, pass, email, first, last)
		if err != nil {
			_, ok := err.(gp.APIerror)
			if ok { //Duplicate user/email or password too short
				jsonResponse(w, err, 400)
			} else {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
			}
		} else {
			jsonResponse(w, &gp.Created{uint64(id)}, 201)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		case !verified:
			resp := struct {
				Status string `json:"status"`
			}{"unverified"}
			jsonResponse(w, resp, 403)
		default:
			token, err := api.CreateAndStoreToken(id)
			if err == nil {
				jsonResponse(w, token, 200)
			} else {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
			}
		}
	default:
		jsonResponse(w, gp.APIerror{"Bad username/password"}, 400)
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
		networks, err := api.GetUserNetworks(userId)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			//First: which paging scheme are we using
			var selector string
			var index int64
			switch {
			case after > 0:
				selector = "after"
				index = after
			case before > 0:
				selector = "before"
				index = before
			default:
				selector = "start"
				index = start
			}
			//Then: if we have a filter
			var posts []gp.PostSmall
			switch {
			case len(filter) > 0:
				posts, err = api.GetPostsByCategory(networks[0].Id, index, selector, api.Config.PostPageSize, filter)
			default:
				log.Println("By network")
				posts, err = api.GetPosts(networks[0].Id, index, selector, api.Config.PostPageSize)
			}
			log.Println(posts, err)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
		switch {
		case len(url) > 5:
			postId, err = api.AddPostWithImage(userId, text, attribs, url, ts...)
		default:
			postId, err = api.AddPost(userId, text, attribs, ts...)
		}
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, &gp.Created{uint64(postId)}, 201)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
		jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
		} else {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
		jsonResponse(w, gp.APIerror{err.Error()}, 400)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
		jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
		jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
		jsonResponse(w, gp.APIerror{err.Error()}, 500)
	} else {
		jsonResponse(w, &gp.Created{uint64(messageId)}, 201)
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
		jsonResponse(w, gp.APIerror{err.Error()}, 400)
	}
	convId := gp.ConversationId(_convId)
	_upTo, err := strconv.ParseUint(r.FormValue("seen"), 10, 64)
	if err != nil {
		_upTo = 0
	}
	upTo := gp.MessageId(_upTo)
	err = api.MarkConversationSeen(userId, convId, upTo)
	if err != nil {
		jsonResponse(w, gp.APIerror{err.Error()}, 500)
	} else {
		conversation, err := api.GetConversation(userId, convId)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, &gp.Created{uint64(commentId)}, 201)
		}
	}
}

func getPost(w http.ResponseWriter, r *http.Request) {
	_, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		postId := gp.PostId(_id)
		post, err := api.GetPostFull(postId)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
			} else {
				images := api.GetPostImages(postId)
				jsonResponse(w, images, 201)
			}
		} else {
			jsonResponse(w, gp.APIerror{"That upload doesn't exist"}, 400)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 400)
		case liked:
			err = api.AddLike(userId, postId)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
			} else {
				jsonResponse(w, gp.Liked{Post: postId, Liked: true}, 200)
			}
		default:
			err = api.DelLike(userId, postId)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
				jsonResponse(w, gp.APIerror{err.Error()}, 404)
				return
			}
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, user, 200)
		}
	}
}

func getUserPosts(w http.ResponseWriter, r *http.Request) {
	_, err := authenticate(r)
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
		var selector string
		var index int64
		switch {
		case after > 0:
			selector = "after"
			index = after
		case before > 0:
			selector = "before"
			index = before
		default:
			selector = "start"
			index = start
		}
		if err != nil {
			after = 0
		}
		posts, err := api.GetUserPosts(profileId, index, api.Config.PostPageSize, selector)
		if err != nil {
			jsonResponse(w, &gp.APIerror{err.Error()}, 500)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
			} else {
				jsonResponse(w, contact, 200)
			}
		} else {
			jsonResponse(w, gp.APIerror{"Missing parameter: accepted"}, 400)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, EBADTOKEN, 400)
	case r.Method == "DELETE":
		log.Println("Delete device hit")
		vars := mux.Vars(r)
		err := api.DeleteDevice(userId, vars["id"])
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 400)
		} else {
			defer file.Close()
			url, err := api.StoreFile(userId, file, header)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 400)
			} else {
				jsonResponse(w, gp.URLCreated{url}, 201)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 400)
			return
		}
		if !exists {
			jsonResponse(w, gp.APIerror{"Image doesn't exist!"}, 400)
		} else {
			err = api.SetProfileImage(userId, url)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
			} else {
				user, err := api.GetProfile(userId)
				if err != nil {
					jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
			jsonResponse(w, gp.APIerror{"Bad input"}, 400)
		}
		err = api.SetBusyStatus(userId, status)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, &gp.BusyStatus{status}, 200)
		}
	case r.Method == "GET":
		status, err := api.BusyStatus(userId)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
			return
		}
		jsonResponse(w, &gp.BusyStatus{status}, 200)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 400)
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
		notificationId := gp.NotificationId(_upTo)
		err = api.MarkNotificationsSeen(userId, notificationId)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			notifications, err := api.GetUserNotifications(userId)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
			} else {
				if len(notifications) == 0 {
					jsonResponse(w, []string{}, 200)
				} else {
					jsonResponse(w, notifications, 200)
				}
			}
		}
	case r.Method == "GET":
		notifications, err := api.GetUserNotifications(userId)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
		//Is this a valid facebook token for this app?
		fbToken, err := api.FBValidateToken(_fbToken)
		if err != nil {
			jsonResponse(w, gp.APIerror{"Bad token"}, 400)
			return
		}
		token, err := api.FacebookLogin(_fbToken)
		//If we have an error here, that means that there is no associated gleepost user account.
		if err != nil {
			//Have we seen this facebook user before?
			_, err := api.FBGetEmail(fbToken.FBUser)
			if err != nil {
				//No. That means we need their email to create and verify their account.
				if len(email) < 3 {
					jsonResponse(w, gp.APIerror{"Email required"}, 400)
					return
				}
				validates, err := api.ValidateEmail(email)
				if !validates {
					jsonResponse(w, gp.APIerror{"Invalid email"}, 400)
					return
				}
				if err != nil {
					jsonResponse(w, gp.APIerror{err.Error()}, 500)
					return
				}
				err = api.FacebookRegister(_fbToken, email)
				if err != nil {
					jsonResponse(w, gp.APIerror{err.Error()}, 500)
					return
				}
			}
			log.Println("Should be unverified response")
			jsonResponse(w, struct {
				Status string `json:"status"`
			}{"unverified"}, 201)
			return
		}
		//If there's an associated user, they're verified already so there's no need to check.
		log.Println("Token: ", token)
		jsonResponse(w, token, 201)
	} else {
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func verificationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		vars := mux.Vars(r)
		err := api.Verify(vars["token"])
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 400)
			return
		}
		jsonResponse(w, struct {
			Verified bool `json:"verified"`
		}{true}, 200)
		return
		jsonResponse(w, gp.APIerror{"Bad verification token"}, 400)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 400)
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
			jsonResponse(w, gp.APIerror{err.Error()}, 400)
			return
		}
		userId := gp.UserId(id)
		pass := r.FormValue("pass")
		err = api.ResetPass(userId, vars["token"], pass)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 400)
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
				jsonResponse(w, gp.APIerror{err.Error()}, 400)
				return
			}
			api.FBissueVerification(fbid)
		} else {
			user, err := api.GetUser(userId)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
			jsonResponse(w, &gp.APIerror{err.Error()}, code)
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
		err := api.Attend(post, userId)
		if err != nil {
			jsonResponse(w, err, 400)
		}
		w.WriteHeader(204)
	case r.Method == "DELETE":
		//For now, assume that err is because the user specified a bad post.
		//Could also be a db error.
		err := api.UnAttend(post, userId)
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
