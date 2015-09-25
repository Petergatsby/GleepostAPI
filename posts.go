package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

func init() {
	base.Handle("/posts", timeHandler(api, authenticated(getPosts))).Methods("GET").Name("posts")
	base.Handle("/posts", timeHandler(api, authenticated(postPosts))).Methods("POST")
	base.Handle("/posts", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}/comments", timeHandler(api, authenticated(getComments))).Methods("GET")
	base.Handle("/posts/{id:[0-9]+}/comments", timeHandler(api, authenticated(postComments))).Methods("POST")
	base.Handle("/posts/{id:[0-9]+}/comments", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}", timeHandler(api, authenticated(getPost))).Methods("GET")
	base.Handle("/posts/{id:[0-9]+}", timeHandler(api, authenticated(putPost))).Methods("PUT")
	base.Handle("/posts/{id:[0-9]+}", timeHandler(api, authenticated(deletePost))).Methods("DELETE")
	base.Handle("/posts/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/posts/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}/images", timeHandler(api, authenticated(postImages))).Methods("POST")
	base.Handle("/posts/{id:[0-9]+}/images", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}/videos", timeHandler(api, authenticated(postVideos))).Methods("POST")
	base.Handle("/posts/{id:[0-9]+}/videos", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}/likes", timeHandler(api, authenticated(postLikes))).Methods("POST")
	base.Handle("/posts/{id:[0-9]+}/likes", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}/attendees", timeHandler(api, authenticated(getAttendees))).Methods("GET")
	base.Handle("/posts/{id:[0-9]+}/attendees", timeHandler(api, authenticated(putAttendees))).Methods("PUT")
	base.Handle("/posts/{id:[0-9]+}/attendees", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/posts/{id:[0-9]+}/attendees", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}/attending", timeHandler(api, authenticated(attendHandler))).Methods("POST", "DELETE")
	base.Handle("/posts/{id:[0-9]+}/attending", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}/votes", timeHandler(api, authenticated(postVotes))).Methods("POST")
	base.Handle("/posts/{id:[0-9]+}/votes", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/posts/{id:[0-9]+}/votes", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/live", timeHandler(api, authenticated(liveHandler))).Methods("GET")
	base.Handle("/live", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/live_summary", timeHandler(api, authenticated(liveSummaryHandler))).Methods("GET")
	base.Handle("/live_summary", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func ignored(key string) bool {
	keys := []string{"id", "token", "text", "url", "tags", "popularity", "video", "poll-expiry", "poll-options"}
	for _, v := range keys {
		if key == v {
			return true
		}
	}
	return false
}

func getPosts(userID gp.UserID, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	mode, index := interpretPagination(req.FormValue("start"), req.FormValue("before"), req.FormValue("after"))
	filter := req.FormValue("filter")
	id, ok := vars["network"]
	var network gp.NetworkID
	var posts []gp.PostSmall
	var _network uint64
	var err error
	switch {
	case ok:
		_network, err = strconv.ParseUint(id, 10, 64)
		if err != nil {
			jsonErr(w, err, 400)
			return
		}
		network = gp.NetworkID(_network)
		posts, err = api.UserGetNetworkPosts(userID, network, mode, index, api.Config.PostPageSize, filter)
	default: //We haven't been given a network, which means this handler is being called by /posts and we just want the users' default network
		posts, err = api.UserGetPrimaryNetworkPosts(userID, mode, index, api.Config.PostPageSize, filter)
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
	url, err := r.Get("posts").URLPath("version", vars["version"])
	if err != nil {
		log.Println(err)
	} else {
		stringURL := fullyQualify(url.String(), api.Config.DevelopmentMode)
		header := paginationHeaders(stringURL, posts)
		w.Header().Set("Link", header)
	}
	jsonResponse(w, posts, 200)
}

func postPosts(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	text := r.FormValue("text")
	url := r.FormValue("url")
	ts := strings.Split(r.FormValue("tags"), ",")
	pollExpiry := r.FormValue("poll-expiry")
	pollOptions := r.Form["poll-options"]
	attribs := make(map[string]string)
	for k, v := range r.Form {
		if !ignored(k) {
			attribs[k] = strings.Join(v, "")
		}
	}
	var postID gp.PostID
	n := vars["network"]
	_network, _ := strconv.ParseUint(n, 10, 64)
	network := gp.NetworkID(_network)
	_vID, _ := strconv.ParseUint(r.FormValue("video"), 10, 64)
	videoID := gp.VideoID(_vID)
	var pending bool
	var err error
	if network > 0 {
		postID, pending, err = api.UserAddPost(userID, network, text, attribs, videoID, false, url, pollExpiry, pollOptions, ts...)
	} else {
		postID, pending, err = api.UserAddPostToPrimary(userID, text, attribs, videoID, false, url, pollExpiry, pollOptions, ts...)
	}
	e, ok := err.(gp.APIerror)
	switch {
	case ok && e == lib.ENOTALLOWED:
		jsonResponse(w, e, 403)
	case ok:
		jsonResponse(w, err, 400)
	case err != nil:
		jsonErr(w, err, 500)
	default:
		jsonResponse(w, &gp.CreatedPost{ID: postID, Pending: pending}, 201)
	}
}

func putPosts(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	n := vars["network"]
	_network, _ := strconv.ParseUint(n, 10, 64)
	network := gp.NetworkID(_network)
	_postID, _ := strconv.ParseUint(r.FormValue("seen"), 10, 64)
	upTo := gp.PostID(_postID)
	err := api.MarkPostsSeen(userID, network, upTo)
	if err != nil {
		jsonErr(w, err, 500)
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(204)
	}
}

func getComments(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	postID := gp.PostID(_id)
	start, err := strconv.ParseInt(r.FormValue("start"), 10, 64)
	if err != nil {
		start = 0
	}
	comments, err := api.UserGetComments(userID, postID, start, api.Config.CommentPageSize)
	if err != nil {
		if err == lib.ENOTALLOWED {
			jsonErr(w, err, 403)
		} else {
			jsonErr(w, err, 500)
		}
	} else {
		jsonResponse(w, comments, 200)
	}
}

func postComments(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	postID := gp.PostID(_id)
	text := r.FormValue("text")
	commentID, err := api.CreateComment(postID, userID, text)
	if err != nil {
		switch {
		case err == lib.CommentTooShort:
			jsonErr(w, err, 400)
		case err == lib.CommentTooLong:
			jsonErr(w, err, 400)
		case err == gp.NoSuchPost:
			jsonErr(w, err, 400)
		case err == lib.ENOTALLOWED:
			jsonErr(w, err, 403)
		default:
			jsonErr(w, err, 500)
		}
	} else {
		jsonResponse(w, &gp.Created{ID: uint64(commentID)}, 201)
	}
}

func getPost(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	postID := gp.PostID(_id)
	post, err := api.UserGetPost(userID, postID)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			jsonResponse(w, e, 403)
		} else {
			jsonErr(w, err, 500)
		}
		return
	}
	jsonResponse(w, post, 200)
}

func putPost(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	postID := gp.PostID(_id)
	text := r.FormValue("text")
	reason := r.FormValue("reason")
	imgurl := r.FormValue("url")
	tags := r.FormValue("tags")
	var ts []string
	if len(tags) > 1 {
		ts = strings.Split(tags, ",")
	}
	_vID, _ := strconv.ParseUint(r.FormValue("video"), 10, 64)
	videoID := gp.VideoID(_vID)
	attribs := make(map[string]string)
	for k, v := range r.Form {
		if !ignored(k) {
			attribs[k] = strings.Join(v, "")
		}
	}
	updatedPost, err := api.UserEditPost(userID, postID, text, attribs, imgurl, videoID, reason, ts...)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			jsonResponse(w, e, 403)
		} else {
			jsonErr(w, err, 500)
		}
		return
	}
	jsonResponse(w, updatedPost, 200)
}

func postImages(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	postID := gp.PostID(_id)
	url := r.FormValue("url")
	images, err := api.UserAddPostImage(userID, postID, url)
	switch {
	case err == lib.ENOTALLOWED:
		jsonErr(w, err, 403)
	case err == lib.NoSuchUpload:
		jsonErr(w, err, 400)
	case err != nil:
		jsonErr(w, err, 500)
	default:
		jsonResponse(w, images, 201)
	}
}

func postVideos(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	postID := gp.PostID(_id)
	_videoID, err := strconv.ParseUint(r.FormValue("video"), 10, 64)
	videoID := gp.VideoID(_videoID)
	videos, err := api.UserAddPostVideo(userID, postID, videoID)
	if err != nil {
		jsonErr(w, err, 500)
	} else {
		jsonResponse(w, videos, 201)
	}
}

func postLikes(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	postID := gp.PostID(_id)
	liked, err := strconv.ParseBool(r.FormValue("liked"))
	switch {
	case err != nil:
		jsonErr(w, err, 400)
	default:
		err = api.UserSetLike(userID, postID, liked)
		switch {
		case err == lib.ENOTALLOWED:
			jsonResponse(w, err, 403)
		case err != nil:
			jsonErr(w, err, 500)
		default:
			jsonResponse(w, gp.Liked{Post: postID, Liked: liked}, 200)
		}
	}
}

func liveHandler(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	after := r.FormValue("after")
	until := r.FormValue("until")
	category := r.FormValue("filter")
	posts, err := api.UserGetLive(userID, after, until, api.Config.PostPageSize, category)
	if err != nil {
		code := 500
		if err == lib.EBADTIME {
			code = 400
		}
		jsonErr(w, err, code)
		return
	}
	jsonResponse(w, posts, 200)
}

func liveSummaryHandler(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	after := r.FormValue("after")
	until := r.FormValue("until")
	summary, err := api.UserGetLiveSummary(userID, after, until)
	if err != nil {
		code := 500
		if err == lib.EBADTIME {
			code = 400
		}
		jsonErr(w, err, code)
		return
	}
	jsonResponse(w, summary, 200)
}

func attendHandler(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//We can safely ignore this error since vars["id"] matches a numeric regex
	//... maybe. What if it's bigger than max(uint64) ??
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	post := gp.PostID(_id)
	switch {
	case r.Method == "POST":
		//For now, assume that err is because the user specified a bad post.
		//Could also be a db error.
		err := api.UserAttend(post, userID, true)
		if err != nil {
			jsonResponse(w, err, 400)
		}
		w.WriteHeader(204)
	case r.Method == "DELETE":
		//For now, assume that err is because the user specified a bad post.
		//Could also be a db error.
		err := api.UserAttend(post, userID, false)
		if err != nil {
			jsonResponse(w, err, 400)
		}
		w.WriteHeader(204)
	}
}

func deletePost(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	postID := gp.PostID(_id)
	err := api.UserDeletePost(userID, postID)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			jsonResponse(w, e, 403)
		} else {
			jsonErr(w, err, 500)
		}
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(204)
}

func putAttendees(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	attending, _ := strconv.ParseBool(r.FormValue("attending"))
	vars := mux.Vars(r)
	_postID, _ := strconv.ParseUint(vars["id"], 10, 64)
	postID := gp.PostID(_postID)
	err := api.UserAttend(postID, userID, attending)
	if err != nil {
		e, ok := err.(*gp.APIerror)
		if ok && *e == lib.ENOTALLOWED {
			jsonResponse(w, e, 403)
		} else {
			jsonErr(w, err, 500)
		}
		return
	}
	getAttendees(userID, w, r)
}

func getAttendees(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_postID, _ := strconv.ParseUint(vars["id"], 10, 64)
	postID := gp.PostID(_postID)
	attendees, err := api.UserGetEventAttendees(userID, postID)
	switch {
	case err == lib.ENOTALLOWED:
		jsonResponse(w, err, 403)
	case err != nil:
		jsonErr(w, err, 500)
	default:
		jsonResponse(w, attendees, 200)
	}
}

func fullyQualify(urlFragment string, development bool) (url string) {
	if development {
		return "https://dev.gleepost.com" + urlFragment
	}
	return "https://gleepost.com" + urlFragment
}

func paginationHeaders(baseURL string, posts []gp.PostSmall) (header string) {
	if len(posts) == 0 {
		return
	}
	newest := posts[0].ID
	oldest := posts[len(posts)-1].ID
	prev := fmt.Sprintf("<%s?after=%d>; rel=\"prev\"", baseURL, newest)
	next := fmt.Sprintf("<%s?before=%d>; rel=\"next\"", baseURL, oldest)
	header = prev + ",\n" + next
	return
}

func postVotes(userID gp.UserID, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_postID, _ := strconv.ParseUint(vars["id"], 10, 64)
	postID := gp.PostID(_postID)
	option, _ := strconv.ParseInt(r.FormValue("option"), 10, 64)
	err := api.UserCastVote(userID, postID, int(option))
	switch {
	case err == lib.ENOTALLOWED:
		jsonResponse(w, err, 403)
	case err == lib.InvalidOption:
		fallthrough
	case err == lib.AlreadyVoted:
		fallthrough
	case err == lib.PollExpired:
		jsonResponse(w, err, 400)
	case err != nil:
		jsonErr(w, err, 500)
	default:
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(204)
	}
}
