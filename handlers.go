package main

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
)

/*********************************************************************************

Begin http handlers!

*********************************************************************************/

func jsonResponder(w http.ResponseWriter, resp interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	marshaled, err := json.Marshal(resp)
	if err != nil {
		marshaled, _ = json.Marshal(APIerror{err.Error()})
		w.WriteHeader(500)
		w.Write(marshaled)
	} else {
		w.WriteHeader(code)
		w.Write(marshaled)
	}
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
	user := r.FormValue("user")
	pass := r.FormValue("pass")
	email := r.FormValue("email")
	switch {
	case r.Method != "POST":
		jsonResponder(w, APIerror{"Must be a POST request!"}, 405)
	case len(user) == 0:
		//Note to future self : would be neater if
		//we returned _all_ errors not just the first
		jsonResponder(w, APIerror{"Missing parameter: user"}, 400)
	case len(pass) == 0:
		jsonResponder(w, APIerror{"Missing parameter: pass"}, 400)
	case len(email) == 0:
		jsonResponder(w, APIerror{"Missing parameter: email"}, 400)
	case !validateEmail(email):
		jsonResponder(w, APIerror{"Invalid Email"}, 400)
	default:
		id, err := registerUser(user, pass, email)
		if err != nil {
			_, ok := err.(APIerror)
			if ok { //Duplicate user/email
				jsonResponder(w, err, 400)
			} else {
				jsonResponder(w, APIerror{err.Error()}, 500)
			}
		} else {
			jsonResponder(w, &Created{uint64(id)}, 201)
		}
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	/* POST /login
		requires parameters: user, pass
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
	user := r.FormValue("user")
	pass := r.FormValue("pass")
	id, err := validatePass(user, pass)
	switch {
	case r.Method != "POST":
		jsonResponder(w, APIerror{"Must be a POST request!"}, 405)
	case err == nil:
		token, err := createAndStoreToken(id)
		if err == nil {
			jsonResponder(w, token, 200)
		} else {
			jsonResponder(w, APIerror{err.Error()}, 500)
		}
	default:
		jsonResponder(w, APIerror{"Bad username/password"}, 400)
	}
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	/* GET /posts
		   requires parameters: id, token

	           POST /posts
		   requires parameters: id, token

	*/
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	userId := UserId(id)
	token := r.FormValue("token")
	switch {
	case !validateToken(userId, token):
		jsonResponder(w, APIerror{"Invalid credentials"}, 400)
	case r.Method == "GET":
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
		if err != nil {
			start = 0
		}
		networks := getUserNetworks(userId)
		posts, err := getPosts(networks[0].Id, start)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		}
		if len(posts) == 0 {
			// this is an ugly hack. But I can't immediately
			// think of a neater way to fix this
			// (json.Marshal(empty slice) returns null rather than
			// empty array ([]) which it obviously should
			jsonResponder(w, []string{}, 200)
		} else {
			jsonResponder(w, posts, 200)
		}
	case r.Method == "POST":
		text := r.FormValue("text")
		postId, err := addPost(userId, text)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		} else {
			jsonResponder(w, &Created{uint64(postId)}, 201)
		}
	default:
		jsonResponder(w, APIerror{"Must be a POST or GET request"}, 405)
	}
}

func newConversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	userId := UserId(id)
	token := r.FormValue("token")
	switch {
	case r.Method != "POST":
		jsonResponder(w, APIerror{"Must be a POST request"}, 405)
	case !validateToken(userId, token):
		jsonResponder(w, APIerror{"Invalid credentials"}, 400)
	default:
		conversation, err := createConversation(userId, 2)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		} else {
			jsonResponder(w, conversation, 201)
		}
	}
}

func newGroupConversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	token := r.FormValue("token")
	userId := UserId(id)
	switch {
	case r.Method != "POST":
		jsonResponder(w, APIerror{"Must be a POST request"}, 405)
	case !validateToken(userId, token):
		jsonResponder(w, APIerror{"Invalid credentials"}, 400)
	default:
		conversation, err := createConversation(userId, 4)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		} else {
			jsonResponder(w, conversation, 201)
		}
	}
}

func conversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	token := r.FormValue("token")
	userId := UserId(id)
	switch {
	case r.Method != "GET":
		jsonResponder(w, APIerror{"Must be a GET request"}, 405)
	case !validateToken(userId, token):
		jsonResponder(w, APIerror{"Invalid credentials"}, 400)
	default:
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			start = 0
		}
		conversations, err := getConversations(userId, start)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		} else {
			if len(conversations) == 0 {
				// this is an ugly hack. But I can't immediately
				// think of a neater way to fix this
				// (json.Marshal(empty slice) returns "null" rather than
				// empty array "[]" which it obviously should
				jsonResponder(w, []string{}, 200)
			} else {
				jsonResponder(w, conversations, 200)
			}
		}
	}
}

func anotherConversationHandler(w http.ResponseWriter, r *http.Request) { //lol
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	userId := UserId(id)
	token := r.FormValue("token")
	regex, _ := regexp.Compile("conversations/(\\d+)/messages/?$")
	convIdString := regex.FindStringSubmatch(r.URL.Path)
	regex2, _ := regexp.Compile("conversations/(\\d+)/?$")
	convIdString2 := regex2.FindStringSubmatch(r.URL.Path)
	switch {
	case !validateToken(userId, token):
		jsonResponder(w, APIerror{"Invalid credentials"}, 400)
	case convIdString != nil && r.Method == "GET":
		_convId, _ := strconv.ParseUint(convIdString[1], 10, 16)
		convId := ConversationId(_convId)
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			start = 0
		}
		after, err := strconv.ParseInt(r.FormValue("after"), 10, 64)
		if err != nil {
			after = 0
		}
		var messages []Message
		if after > 0 {
			messages, err = getMessagesAfter(convId, after)
		} else {
			messages, err = getMessages(convId, start)
		}
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		} else {
			if len(messages) == 0 {
				// this is an ugly hack. But I can't immediately
				// think of a neater way to fix this
				// (json.Marshal(empty slice) returns "null" rather than
				// empty array "[]" which it obviously should
				jsonResponder(w, []string{}, 200)
			} else {
				jsonResponder(w, messages, 200)
			}
		}
	case convIdString != nil && r.Method == "POST":
		_convId, _ := strconv.ParseUint(convIdString[1], 10, 16)
		convId := ConversationId(_convId)
		text := r.FormValue("text")
		messageId, err := addMessage(convId, userId, text)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		} else {
			jsonResponder(w, &Created{uint64(messageId)}, 201)
		}
	case convIdString != nil && r.Method == "PUT":
		_convId, err := strconv.ParseUint(convIdString[1], 10, 16)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 400)
		}
		convId := ConversationId(_convId)
		_upTo, err := strconv.ParseUint(r.FormValue("seen"), 10, 16)
		if err != nil {
			_upTo = 0
		}
		upTo := MessageId(_upTo)
		conversation, err := markConversationSeen(userId, convId, upTo)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		} else {
			jsonResponder(w, conversation, 200)
		}
	case convIdString != nil: //Unsuported method
		jsonResponder(w, APIerror{"Must be a GET or POST request"}, 405)
	case convIdString2 != nil && r.Method != "GET":
		jsonResponder(w, APIerror{"Must be a GET request"}, 405)
	case convIdString2 != nil:
		_convId, _ := strconv.ParseInt(convIdString2[1], 10, 16)
		convId := ConversationId(_convId)
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			start = 0
		}
		conv, err := getFullConversation(convId, start)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		}
		jsonResponder(w, conv, 200)
	default:
		jsonResponder(w, APIerror{"404 not found"}, 404)
	}
}

func anotherPostHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	token := r.FormValue("token")
	userId := UserId(id)
	regexComments, _ := regexp.Compile("posts/(\\d+)/comments/?$")
	regexNoComments, _ := regexp.Compile("posts/(\\d+)/?$")
	commIdStringA := regexComments.FindStringSubmatch(r.URL.Path)
	commIdStringB := regexNoComments.FindStringSubmatch(r.URL.Path)
	switch {
	case !validateToken(userId, token):
		jsonResponder(w, APIerror{"Invalid credentials"}, 400)
	case commIdStringA != nil && r.Method == "GET":
		_id, _ := strconv.ParseUint(commIdStringA[1], 10, 16)
		postId := PostId(_id)
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			start = 0
		}
		comments, err := getComments(postId, start)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		} else {
			if len(comments) == 0 {
				// this is an ugly hack. But I can't immediately
				// think of a neater way to fix this
				// (json.Marshal(empty slice) returns "null" rather than
				// empty array "[]" which it obviously should
				jsonResponder(w, []string{}, 200)
			} else {
				jsonResponder(w, comments, 200)
			}
		}
	case commIdStringA != nil && r.Method == "POST":
		_id, _ := strconv.ParseUint(commIdStringA[1], 10, 16)
		postId := PostId(_id)
		text := r.FormValue("text")
		commentId, err := createComment(postId, userId, text)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		} else {
			jsonResponder(w, &Created{uint64(commentId)}, 201)
		}
	case commIdStringB != nil && r.Method == "GET":
		_id, _ := strconv.ParseUint(commIdStringB[1], 10, 16)
		postId := PostId(_id)
		log.Printf("%d", postId)
		//implement getting a specific post
	default:
		jsonResponder(w, APIerror{"Method not supported"}, 405)
	}
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	userId := UserId(id)
	token := r.FormValue("token")
	regexUser, _ := regexp.Compile("user/(\\d+)/?$")
	userIdString := regexUser.FindStringSubmatch(r.URL.Path)
	switch {
	case !validateToken(userId, token):
		jsonResponder(w, APIerror{"Invalid credentials"}, 400)
	case r.Method != "GET":
		jsonResponder(w, APIerror{"Method not supported"}, 405)
	case userIdString != nil:
		u, _ := strconv.ParseUint(userIdString[1], 10, 16)
		profileId := UserId(u)
		user, err := getProfile(profileId)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		}
		jsonResponder(w, user, 200)
	default:
		jsonResponder(w, APIerror{"User not found"}, 404)
	}
}

func longPollHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	userId := UserId(id)
	token := r.FormValue("token")
	switch {
	case !validateToken(userId, token):
		jsonResponder(w, APIerror{"Invalid credentials"}, 400)
	case r.Method != "GET":
		jsonResponder(w, APIerror{"Method not supported"}, 405)
	default:
		//awaitOneMessage will block until a message arrives over redis
		message := awaitOneMessage(userId)
		w.Write(message)
	}
}

func contactsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	userId := UserId(id)
	token := r.FormValue("token")
	switch {
	case !validateToken(userId, token):
		jsonResponder(w, APIerror{"Invalid credentials"}, 400)
	case r.Method == "GET":
		contacts, err := getContacts(userId)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		} else {
			if len(contacts) == 0 {
				jsonResponder(w, []string{}, 200)
			} else {
				jsonResponder(w, contacts, 200)
			}
		}
	case r.Method == "POST":
		_otherId, _ := strconv.ParseUint(r.FormValue("user"), 10, 64)
		otherId := UserId(_otherId)
		contact, err := addContact(userId, otherId)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		} else {
			jsonResponder(w, contact, 201)
		}
	default:
		jsonResponder(w, APIerror{"Method not supported"}, 405)
	}
}

func anotherContactsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	userId := UserId(id)
	token := r.FormValue("token")
	rx, _ := regexp.Compile("contacts/(\\d+)/?$")
	contactIdStrings := rx.FindStringSubmatch(r.URL.Path)
	switch {
	case !validateToken(userId, token):
		jsonResponder(w, APIerror{"Invalid credentials"}, 400)
	case r.Method == "PUT" && contactIdStrings != nil:
		_contact, err := strconv.ParseUint(contactIdStrings[1], 10, 64)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 400)
		}
		contactId := UserId(_contact)
		accepted, err := strconv.ParseBool(r.FormValue("accepted"))
		if err != nil {
			accepted = false
		}
		if accepted {
			contact, err := acceptContact(userId, contactId)
			if err != nil {
				jsonResponder(w, APIerror{err.Error()}, 500)
			} else {
				jsonResponder(w, contact, 200)
			}
		} else {
			jsonResponder(w, APIerror{"Missing parameter: accepted"}, 400)
		}
	case r.Method == "DELETE" && contactIdStrings != nil:
		//Implement refusing requests
	default:
		jsonResponder(w, APIerror{"Method not supported"}, 405)
	}
}

func deviceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	userId := UserId(id)
	token := r.FormValue("token")
	switch {
	case !validateToken(userId, token):
		jsonResponder(w, APIerror{"Invalid credentials"}, 400)
	case r.Method == "POST":
		deviceType := r.FormValue("type")
		deviceId := r.FormValue("device_id")
		device, err := addDevice(userId, deviceType, deviceId)
		if err != nil {
			jsonResponder(w, APIerror{err.Error()}, 500)
		} else {
			jsonResponder(w, device, 201)
		}
	case r.Method == "GET":
		//implement getting tokens
	case r.Method == "DELETE":
		//Implement deregistering device
	default:
		jsonResponder(w, APIerror{"Method not supported"}, 405)
	}
}
