package main

import (
	"encoding/json"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"code.google.com/p/go.net/websocket"
)

//Note to self: validateToken should probably return an error at some point
func authenticate(r *http.Request) (userId gp.UserId, err error) {
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	userId = gp.UserId(id)
	token := r.FormValue("token")
	success := lib.ValidateToken(userId, token)
	if success {
		return userId, nil
	} else {
		return 0, &EBADTOKEN
	}
}

func jsonResponse(w http.ResponseWriter, resp interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
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
	user := r.FormValue("user")
	pass := r.FormValue("pass")
	email := r.FormValue("email")
	switch {
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	case len(user) == 0:
		//Note to future self : would be neater if
		//we returned _all_ errors not just the first
		jsonResponse(w, gp.APIerror{"Missing parameter: user"}, 400)
	case len(pass) == 0:
		jsonResponse(w, gp.APIerror{"Missing parameter: pass"}, 400)
	case len(email) == 0:
		jsonResponse(w, gp.APIerror{"Missing parameter: email"}, 400)
	default:
		validates, err := lib.ValidateEmail(email)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
			return
		}
		if !validates {
			jsonResponse(w, gp.APIerror{"Invalid Email"}, 400)
			return
		}
		id, err := lib.RegisterUser(user, pass, email)
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
	id, err := lib.ValidatePass(user, pass)
	switch {
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	case err == nil:
		token, err := lib.CreateAndStoreToken(id)
		if err == nil {
			jsonResponse(w, token, 200)
		} else {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		}
	default:
		jsonResponse(w, gp.APIerror{"Bad username/password"}, 400)
	}
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	/* GET /posts
		   requires parameters: id, token

	           POST /posts
		   requires parameters: id, token

	*/
	w.Header().Set("Content-Type", "application/json")
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
		networks, err := lib.GetUserNetworks(userId)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			var posts []gp.PostSmall
			switch {
			case after > 0:
				posts, err = lib.GetPosts(networks[0].Id, after, "after")
			case before > 0:
				posts, err = lib.GetPosts(networks[0].Id, before, "before")
			default:
				posts, err = lib.GetPosts(networks[0].Id, start, "start")
			}
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
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
	case r.Method == "POST":
		text := r.FormValue("text")
		postId, err := lib.AddPost(userId, text)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, &gp.Created{uint64(postId)}, 201)
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func newConversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userId, err := authenticate(r)
	switch {
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		conversation, err := lib.CreateConversation(userId, 2, true)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, conversation, 201)
		}
	}
}

func newGroupConversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userId, err := authenticate(r)
	switch {
	case r.Method != "POST":
		jsonResponse(w, &EUNSUPPORTED, 405)
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		conversation, err := lib.CreateConversation(userId, 4, true)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, conversation, 201)
		}
	}
}

func conversationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userId, err := authenticate(r)
	switch {
	case r.Method != "GET":
		jsonResponse(w, &EUNSUPPORTED, 405)
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
		if err != nil {
			start = 0
		}
		conf := gp.GetConfig()
		conversations, err := lib.GetConversations(userId, start, conf.ConversationPageSize)
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
}

func anotherConversationHandler(w http.ResponseWriter, r *http.Request) { //lol
	w.Header().Set("Content-Type", "application/json")
	userId, err := authenticate(r)
	regex, _ := regexp.Compile("conversations/(\\d+)/messages/?$")
	convIdString := regex.FindStringSubmatch(r.URL.Path)
	regex2, _ := regexp.Compile("conversations/(\\d+)/?$")
	convIdString2 := regex2.FindStringSubmatch(r.URL.Path)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case convIdString != nil && r.Method == "GET":
		_convId, _ := strconv.ParseUint(convIdString[1], 10, 64)
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
			messages, err = lib.GetMessages(convId, after, "after")
		case before > 0:
			messages, err = lib.GetMessages(convId, before, "before")
		default:
			messages, err = lib.GetMessages(convId, start, "start")
		}
		if err != nil {
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
	case convIdString != nil && r.Method == "POST":
		_convId, _ := strconv.ParseUint(convIdString[1], 10, 64)
		convId := gp.ConversationId(_convId)
		text := r.FormValue("text")
		messageId, err := lib.AddMessage(convId, userId, text)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, &gp.Created{uint64(messageId)}, 201)
		}
	case convIdString != nil && r.Method == "PUT":
		_convId, err := strconv.ParseUint(convIdString[1], 10, 64)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 400)
		}
		convId := gp.ConversationId(_convId)
		_upTo, err := strconv.ParseUint(r.FormValue("seen"), 10, 64)
		if err != nil {
			_upTo = 0
		}
		upTo := gp.MessageId(_upTo)
		conversation, err := lib.MarkConversationSeen(userId, convId, upTo)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, conversation, 200)
		}
	case convIdString != nil: //Unsuported method
		jsonResponse(w, &EUNSUPPORTED, 405)
	case convIdString2 != nil && r.Method == "GET":
		_convId, _ := strconv.ParseInt(convIdString2[1], 10, 64)
		convId := gp.ConversationId(_convId)
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
		if err != nil {
			start = 0
		}
		conv, err := lib.GetFullConversation(convId, start)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		}
		jsonResponse(w, conv, 200)
	case convIdString2 != nil && r.Method == "DELETE":
		_convId, _ := strconv.ParseInt(convIdString2[1], 10, 64)
		convId := gp.ConversationId(_convId)
		err := lib.TerminateConversation(convId)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
			return
		}
		w.WriteHeader(204)
	case convIdString2 != nil && r.Method == "PUT":
		_convId, _ := strconv.ParseInt(convIdString2[1], 10, 64)
		convId := gp.ConversationId(_convId)
		expires, err := strconv.ParseBool(r.FormValue("expiry"))
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 400)
			return
		}
		if expires == false {
			err = lib.DeleteExpiry(convId)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
				return
			}
		}
		conversation, err := lib.GetConversation(userId, convId)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
			return
		}
		jsonResponse(w, conversation, 200)
	case convIdString2 != nil:
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		jsonResponse(w, ENOTFOUND, 404)
	}
}

func anotherPostHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userId, err := authenticate(r)
	regexComments, _ := regexp.Compile("posts/(\\d+)/comments/?$")
	regexNoComments, _ := regexp.Compile("posts/(\\d+)/?$")
	regexImages, _ := regexp.Compile("posts/(\\d+)/images/?$")
	regexLikes, _ := regexp.Compile("posts/(\\d+)/likes/?$")
	commIdStringA := regexComments.FindStringSubmatch(r.URL.Path)
	commIdStringB := regexNoComments.FindStringSubmatch(r.URL.Path)
	commIdStringC := regexImages.FindStringSubmatch(r.URL.Path)
	commIdStringD := regexLikes.FindStringSubmatch(r.URL.Path)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case commIdStringA != nil && r.Method == "GET":
		_id, _ := strconv.ParseUint(commIdStringA[1], 10, 64)
		postId := gp.PostId(_id)
		start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
		if err != nil {
			start = 0
		}
		comments, err := lib.GetComments(postId, start)
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
	case commIdStringA != nil && r.Method == "POST":
		_id, _ := strconv.ParseUint(commIdStringA[1], 10, 64)
		postId := gp.PostId(_id)
		text := r.FormValue("text")
		commentId, err := lib.CreateComment(postId, userId, text)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, &gp.Created{uint64(commentId)}, 201)
		}
	case commIdStringB != nil && r.Method == "GET":
		_id, _ := strconv.ParseUint(commIdStringB[1], 10, 64)
		postId := gp.PostId(_id)
		post, err := lib.GetPostFull(postId)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, post, 200)
		}
	case commIdStringC != nil && r.Method == "POST":
		_id, _ := strconv.ParseUint(commIdStringC[1], 10, 64)
		postId := gp.PostId(_id)
		url := r.FormValue("url")
		exists, err := lib.UserUploadExists(userId, url)
		if exists && err == nil {
			err := lib.AddPostImage(postId, url)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
			} else {
				images := lib.GetPostImages(postId)
				jsonResponse(w, images, 201)
			}
		} else {
			jsonResponse(w, gp.APIerror{"That upload doesn't exist"}, 400)
		}
	case commIdStringD != nil && r.Method == "POST":
		_id, _ := strconv.ParseUint(commIdStringD[1], 10, 64)
		postId := gp.PostId(_id)
		liked, err := strconv.ParseBool(r.FormValue("liked"))
		switch {
		case err != nil:
			jsonResponse(w, gp.APIerror{err.Error()}, 400)
		case liked:
			err = lib.AddLike(userId, postId)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
			} else {
				jsonResponse(w, gp.Liked{Post: postId, Liked: true}, 200)
			}
		default:
			err = lib.DelLike(userId, postId)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
			} else {
				jsonResponse(w, gp.Liked{Post: postId, Liked: false}, 200)
			}
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, err := authenticate(r)
	regexUser, _ := regexp.Compile("user/(\\d+)/?$")
	userIdString := regexUser.FindStringSubmatch(r.URL.Path)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "GET":
		jsonResponse(w, EUNSUPPORTED, 405)
	case userIdString != nil:
		u, _ := strconv.ParseUint(userIdString[1], 10, 64)
		profileId := gp.UserId(u)
		user, err := lib.GetProfile(profileId)
		if err != nil {
			if err == gp.ENOSUCHUSER {
				jsonResponse(w, gp.APIerror{err.Error()}, 404)
				return
			}
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, user, 200)
		}
	default:
		jsonResponse(w, gp.APIerror{"User not found"}, 404)
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
		message := lib.AwaitOneMessage(userId)
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
		contacts, err := lib.GetContacts(userId)
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
		contact, err := lib.AddContact(userId, otherId)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, contact, 201)
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func anotherContactsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userId, err := authenticate(r)
	rx, _ := regexp.Compile("contacts/(\\d+)/?$")
	contactIdStrings := rx.FindStringSubmatch(r.URL.Path)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "PUT" && contactIdStrings != nil:
		_contact, err := strconv.ParseUint(contactIdStrings[1], 10, 64)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 400)
		}
		contactId := gp.UserId(_contact)
		accepted, err := strconv.ParseBool(r.FormValue("accepted"))
		if err != nil {
			accepted = false
		}
		if accepted {
			contact, err := lib.AcceptContact(userId, contactId)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
			} else {
				jsonResponse(w, contact, 200)
			}
		} else {
			jsonResponse(w, gp.APIerror{"Missing parameter: accepted"}, 400)
		}
	case r.Method == "DELETE" && contactIdStrings != nil:
		//Implement refusing requests
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func deviceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		deviceType := r.FormValue("type")
		deviceId := r.FormValue("device_id")
		device, err := lib.AddDevice(userId, deviceType, deviceId)
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

func deleteDeviceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userId, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, EBADTOKEN, 400)
	case r.Method == "DELETE":
		regex, _ := regexp.Compile("devices/([:alnum:]+)/?$")
		deviceIdString := regex.FindStringSubmatch(r.URL.Path)
		if deviceIdString != nil {
			err := lib.DeleteDevice(userId, deviceIdString[1])
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
				return
			}
			w.WriteHeader(204)
			return
		}
		jsonResponse(w, gp.APIerror{"Provide a device id"}, 400)
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
			url, err := lib.StoreFile(userId, file, header)
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
		exists, err := lib.UserUploadExists(userId, url)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 400)
			return
		}
		if !exists {
			jsonResponse(w, gp.APIerror{"Image doesn't exist!"}, 400)
		} else {
			err = lib.SetProfileImage(userId, url)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
			} else {
				user, err := lib.GetProfile(userId)
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
		err = lib.SetBusyStatus(userId, status)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			jsonResponse(w, &gp.BusyStatus{status}, 200)
		}
	case r.Method == "GET":
		status, err := lib.BusyStatus(userId)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
			return
		}
		jsonResponse(w, &gp.BusyStatus{status}, 200)
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
		err = lib.MarkNotificationsSeen(userId, notificationId)
		if err != nil {
			jsonResponse(w, gp.APIerror{err.Error()}, 500)
		} else {
			notifications, err := lib.GetUserNotifications(userId)
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
		notifications, err := lib.GetUserNotifications(userId)
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
		fbToken := r.FormValue("token")
		email := r.FormValue("email")
		_, err := lib.FBValidateToken(fbToken)
		if err != nil {
			jsonResponse(w, gp.APIerror{"Bad token"}, 400)
			return
		}
		token, err := lib.FacebookLogin(fbToken)
		if err != nil {
			if len(email) < 3 {
				jsonResponse(w, gp.APIerror{"Email required"}, 400)
				return
			}
			validates, err := lib.ValidateEmail(email)
			if !validates {
				jsonResponse(w, gp.APIerror{"Invalid email"}, 400)
				return
			}
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
				return
			}
			err = lib.FacebookRegister(fbToken, email)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 500)
				return
			}
			log.Println("Should be unverified response")
			jsonResponse(w, struct {
				Status string `json:"status"`
			}{"unverified"}, 201)
			return
		}
		log.Println("Token: ", token)
		jsonResponse(w, token, 201)
	} else {
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func verificationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		regex, _ := regexp.Compile("verify/(\\d+)/?$")
		tokenString := regex.FindStringSubmatch(r.URL.Path)
		if tokenString != nil {
			token := tokenString[1]
			err := lib.Verify(token)
			if err != nil {
				jsonResponse(w, gp.APIerror{err.Error()}, 400)
				return
			}
			jsonResponse(w, struct {
				Verified bool `json:"verified"`
			}{true}, 200)
			return
		}
		jsonResponse(w, gp.APIerror{"Bad verification token"}, 400)
	} else {
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func jsonServer(ws *websocket.Conn) {
	c := lib.GetMessageChan(2395)
	for {
		select {
		case message := <-c:
			ws.Write(message)
		}
	}
}
