package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
)

/*********************************************************************************

Begin http handlers!

*********************************************************************************/

func jsonResp(w http.ResponseWriter, resp []byte, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(resp)
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
		errorJSON, _ := json.Marshal(APIerror{"Must be a POST request!"})
		jsonResp(w, errorJSON, 405)
	case len(user) == 0:
		//Note to future self : would be neater if
		//we returned _all_ errors not just the first
		errorJSON, _ := json.Marshal(APIerror{"Missing parameter: user"})
		jsonResp(w, errorJSON, 400)
	case len(pass) == 0:
		errorJSON, _ := json.Marshal(APIerror{"Missing parameter: pass"})
		jsonResp(w, errorJSON, 400)
	case len(email) == 0:
		errorJSON, _ := json.Marshal(APIerror{"Missing parameter: email"})
		jsonResp(w, errorJSON, 400)
	case !validateEmail(email):
		errorJSON, _ := json.Marshal(APIerror{"Invalid Email"})
		jsonResp(w, errorJSON, 400)
	default:
		id, err := registerUser(user, pass, email)
		if err != nil {
			_, ok := err.(APIerror)
			if ok { //Duplicate user/email
				errorJSON, _ := json.Marshal(err)
				jsonResp(w, errorJSON, 400)
			} else {
				errorJSON, _ := json.Marshal(APIerror{err.Error()})
				jsonResp(w, errorJSON, 500)
			}
		} else {
			resp := []byte(fmt.Sprintf("{\"id\":%d}", id))
			jsonResp(w, resp, 201)
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
		errorJSON, _ := json.Marshal(APIerror{"Must be a POST request!"})
		jsonResp(w, errorJSON, 405)
	case err == nil:
		token, err := createAndStoreToken(id)
		if err == nil {
			tokenJSON, _ := json.Marshal(token)
			jsonResp(w, tokenJSON, 200)
		} else {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		}
	default:
		errorJSON, _ := json.Marshal(APIerror{"Bad username/password"})
		jsonResp(w, errorJSON, 400)
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
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case r.Method == "GET":
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
		if err != nil {
			start = 0
		}
		networks := getUserNetworks(userId)
		posts, err := getPosts(networks[0].Id, start)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		}
		if len(posts) == 0 {
			// this is an ugly hack. But I can't immediately
			// think of a neater way to fix this
			// (json.Marshal(empty slice) returns null rather than
			// empty array ([]) which it obviously should
			w.Write([]byte("[]"))
		} else {
			postsJSON, err := json.Marshal(posts)
			if err != nil {
				log.Printf("Something went wrong with json parsing: %v", err)
			}
			jsonResp(w, postsJSON, 200)
		}
	case r.Method == "POST":
		text := r.FormValue("text")
		postId, err := addPost(userId, text)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			w.Write([]byte(fmt.Sprintf("{\"id\":%d}", postId)))
		}
	default:
		errorJSON, _ := json.Marshal(APIerror{"Must be a POST or GET request"})
		jsonResp(w, errorJSON, 405)
	}
}

func newConversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	userId := UserId(id)
	token := r.FormValue("token")
	switch {
	case r.Method != "POST":
		errorJSON, _ := json.Marshal(APIerror{"Must be a POST request"})
		jsonResp(w, errorJSON, 405)
	case !validateToken(userId, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	default:
		conversation, err := createConversation(userId, 2)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			conversationJSON, _ := json.Marshal(conversation)
			w.Write(conversationJSON)
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
		errorJSON, _ := json.Marshal(APIerror{"Must be a POST request"})
		jsonResp(w, errorJSON, 405)
	case !validateToken(userId, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	default:
		conversation, err := createConversation(userId, 4)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			conversationJSON, _ := json.Marshal(conversation)
			w.Write(conversationJSON)
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
		errorJSON, _ := json.Marshal(APIerror{"Must be a GET request"})
		jsonResp(w, errorJSON, 405)
	case !validateToken(userId, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	default:
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			start = 0
		}
		conversations, err := getConversations(userId, start)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			if len(conversations) == 0 {
				// this is an ugly hack. But I can't immediately
				// think of a neater way to fix this
				// (json.Marshal(empty slice) returns "null" rather than
				// empty array "[]" which it obviously should
				w.Write([]byte("[]"))
			} else {
				conversationsJSON, _ := json.Marshal(conversations)
				w.Write(conversationsJSON)
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
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
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
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			if len(messages) == 0 {
				// this is an ugly hack. But I can't immediately
				// think of a neater way to fix this
				// (json.Marshal(empty slice) returns "null" rather than
				// empty array "[]" which it obviously should
				w.Write([]byte("[]"))
			} else {
				messagesJSON, _ := json.Marshal(messages)
				w.Write(messagesJSON)
			}
		}
	case convIdString != nil && r.Method == "POST":
		_convId, _ := strconv.ParseUint(convIdString[1], 10, 16)
		convId := ConversationId(_convId)
		text := r.FormValue("text")
		messageId, err := addMessage(convId, userId, text)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		}
		w.Write([]byte(fmt.Sprintf("{\"id\":%d}", messageId)))
	case convIdString != nil: //Unsuported method
		errorJSON, _ := json.Marshal(APIerror{"Must be a GET or POST request"})
		jsonResp(w, errorJSON, 405)
	case convIdString2 != nil && r.Method != "GET":
		errorJSON, _ := json.Marshal(APIerror{"Must be a GET request"})
		jsonResp(w, errorJSON, 405)
	case convIdString2 != nil:
		_convId, _ := strconv.ParseInt(convIdString2[1], 10, 16)
		convId := ConversationId(_convId)
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			start = 0
		}
		conv, err := getFullConversation(convId, start)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		}
		conversationJSON, _ := json.Marshal(conv)
		w.Write(conversationJSON)
	default:
		errorJSON, _ := json.Marshal(APIerror{"404 not found"})
		jsonResp(w, errorJSON, 404)
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
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case commIdStringA != nil && r.Method == "GET":
		_id, _ := strconv.ParseUint(commIdStringA[1], 10, 16)
		postId := PostId(_id)
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 16)
		if err != nil {
			start = 0
		}
		comments, err := getComments(postId, start)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			if len(comments) == 0 {
				// this is an ugly hack. But I can't immediately
				// think of a neater way to fix this
				// (json.Marshal(empty slice) returns "null" rather than
				// empty array "[]" which it obviously should
				w.Write([]byte("[]"))
			} else {
				jsonComments, _ := json.Marshal(comments)
				w.Write(jsonComments)
			}
		}
	case commIdStringA != nil && r.Method == "POST":
		_id, _ := strconv.ParseUint(commIdStringA[1], 10, 16)
		postId := PostId(_id)
		text := r.FormValue("text")
		commentId, err := createComment(postId, userId, text)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			w.Write([]byte(fmt.Sprintf("{\"id\":%d}", commentId)))
		}
	case commIdStringB != nil && r.Method == "GET":
		_id, _ := strconv.ParseUint(commIdStringB[1], 10, 16)
		postId := PostId(_id)
		log.Printf("%d", postId)
		//implement getting a specific post
	default:
		errorJSON, _ := json.Marshal(APIerror{"Method not supported"})
		jsonResp(w, errorJSON, 405)
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
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case r.Method != "GET":
		errorJSON, _ := json.Marshal(APIerror{"Method not supported"})
		jsonResp(w, errorJSON, 405)
	case userIdString != nil:
		u, _ := strconv.ParseUint(userIdString[1], 10, 16)
		profileId := UserId(u)
		user, err := getProfile(profileId)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		}
		userjson, _ := json.Marshal(user)
		w.Write(userjson)
	default:
		errorJSON, _ := json.Marshal(APIerror{"User not found"})
		jsonResp(w, errorJSON, 404)
	}
}

func longPollHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 16)
	userId := UserId(id)
	token := r.FormValue("token")
	switch {
	case !validateToken(userId, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case r.Method != "GET":
		errorJSON, _ := json.Marshal(APIerror{"Method not supported"})
		jsonResp(w, errorJSON, 405)
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
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case r.Method == "GET":
		contacts, err := getContacts(userId)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			if len(contacts) == 0 {
				jsonResp(w, []byte("[]"), 200)
			} else {
				contactsJSON, _ := json.Marshal(contacts)
				jsonResp(w, contactsJSON, 200)
			}
		}
	case r.Method == "POST":
		_otherId, _ := strconv.ParseUint(r.FormValue("user"), 10, 64)
		otherId := UserId(_otherId)
		contact, err := addContact(userId, otherId)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			contactJSON, _ := json.Marshal(contact)
			jsonResp(w, contactJSON, 201)
		}
	default:
		errorJSON, _ := json.Marshal(APIerror{"Method not supported"})
		jsonResp(w, errorJSON, 405)
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
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case r.Method == "PUT" && contactIdStrings != nil:
		_contact, err := strconv.ParseUint(contactIdStrings[1], 10, 64)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 400)
		}
		contactId := UserId(_contact)
		accepted, err := strconv.ParseBool(r.FormValue("accepted"))
		if err != nil {
			accepted = false
		}
		if accepted {
			contact, err := acceptContact(userId, contactId)
			if err != nil {
				errorJSON, _ := json.Marshal(APIerror{err.Error()})
				jsonResp(w, errorJSON, 500)
			} else {
				contactJSON, _ := json.Marshal(contact)
				jsonResp(w, contactJSON, 200)
			}
		} else {
			errorJSON, _ := json.Marshal(APIerror{"Missing parameter: accepted"})
			jsonResp(w, errorJSON, 400)
		}
	case r.Method == "DELETE" && contactIdStrings != nil:
		//Implement refusing requests
	default:
		errorJSON, _ := json.Marshal(APIerror{"Method not supported"})
		jsonResp(w, errorJSON, 405)
	}
}

func deviceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	userId := UserId(id)
	token := r.FormValue("token")
	switch {
	case !validateToken(userId, token):
		errorJSON, _ := json.Marshal(APIerror{"Invalid credentials"})
		jsonResp(w, errorJSON, 400)
	case r.Method == "POST":
		deviceType := r.FormValue("type")
		deviceId := r.FormValue("device_id")
		device, err := addDevice(userId, deviceType, deviceId)
		if err != nil {
			errorJSON, _ := json.Marshal(APIerror{err.Error()})
			jsonResp(w, errorJSON, 500)
		} else {
			deviceJSON, _ := json.Marshal(device)
			jsonResp(w, deviceJSON, 201)
		}
	case r.Method == "GET":
		//implement getting tokens
	case r.Method == "DELETE":
		//Implement deregistering device
	default:
		errorJSON, _ := json.Marshal(APIerror{"Method not supported"})
		jsonResp(w, errorJSON, 405)
	}
}
