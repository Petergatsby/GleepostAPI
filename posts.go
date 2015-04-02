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
	base.Handle("/posts", timeHandler(api, http.HandlerFunc(getPosts))).Methods("GET").Name("posts")
	base.Handle("/posts", timeHandler(api, http.HandlerFunc(postPosts))).Methods("POST")
	base.Handle("/posts", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}/comments", timeHandler(api, http.HandlerFunc(getComments))).Methods("GET")
	base.Handle("/posts/{id:[0-9]+}/comments", timeHandler(api, http.HandlerFunc(postComments))).Methods("POST")
	base.Handle("/posts/{id:[0-9]+}/comments", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(getPost))).Methods("GET")
	base.Handle("/posts/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(putPost))).Methods("PUT")
	base.Handle("/posts/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(deletePost))).Methods("DELETE")
	base.Handle("/posts/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}/images", timeHandler(api, http.HandlerFunc(postImages))).Methods("POST")
	base.Handle("/posts/{id:[0-9]+}/images", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}/videos", timeHandler(api, http.HandlerFunc(postVideos))).Methods("POST")
	base.Handle("/posts/{id:[0-9]+}/videos", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}/likes", timeHandler(api, http.HandlerFunc(postLikes))).Methods("POST")
	base.Handle("/posts/{id:[0-9]+}/likes", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}/attendees", timeHandler(api, http.HandlerFunc(getAttendees))).Methods("GET")
	base.Handle("/posts/{id:[0-9]+}/attendees", timeHandler(api, http.HandlerFunc(putAttendees))).Methods("PUT")
	base.Handle("/posts/{id:[0-9]+}/attendees", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/posts/{id:[0-9]+}/attending", timeHandler(api, http.HandlerFunc(attendHandler))).Methods("POST", "DELETE")
	base.Handle("/posts/{id:[0-9]+}/attending", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/live", timeHandler(api, http.HandlerFunc(liveHandler))).Methods("GET")
	base.Handle("/live", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
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

func getPosts(w http.ResponseWriter, req *http.Request) {
	userID, err := authenticate(req)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		start, err := strconv.ParseInt(req.FormValue("start"), 10, 64)
		if err != nil {
			start = 0
		}
		before, err := strconv.ParseInt(req.FormValue("before"), 10, 64)
		if err != nil {
			before = 0
		}
		after, err := strconv.ParseInt(req.FormValue("after"), 10, 64)
		if err != nil {
			after = 0
		}
		filter := req.FormValue("filter")
		vars := mux.Vars(req)
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
		id, ok := vars["network"]
		var network gp.NetworkID
		var posts []gp.PostSmall
		var _network uint64
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
}

func postPosts(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		text := r.FormValue("text")
		url := r.FormValue("url")
		ts := strings.Split(r.FormValue("tags"), ",")
		pollExpiry := r.FormValue("poll-expiry")
		pollOptions := strings.Split(r.FormValue("poll-options"), ",")
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
		if network > 0 {
			postID, pending, err = api.UserAddPost(userID, network, text, attribs, videoID, false, url, pollExpiry, pollOptions, ts...)
		} else {
			postID, pending, err = api.UserAddPostToPrimary(userID, text, attribs, videoID, false, url, pollExpiry, pollOptions, ts...)
		}
		if err != nil {
			e, ok := err.(gp.APIerror)
			switch {
			case ok && e == lib.ENOTALLOWED:
				jsonResponse(w, e, 403)
			case ok:
				jsonResponse(w, err, 400)
			default:
				jsonErr(w, err, 500)
			}
		} else {
			jsonResponse(w, &gp.CreatedPost{ID: postID, Pending: pending}, 201)
		}
	}
}

func getComments(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
}

func postComments(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		postID := gp.PostID(_id)
		text := r.FormValue("text")
		commentID, err := api.CreateComment(postID, userID, text)
		if err != nil {
			switch {
			case err == lib.CommentTooShort:
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
}

func getPost(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
}

func putPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := authenticate(r)
	if err != nil {
		jsonResponse(w, &EBADTOKEN, 400)
		return
	}
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

func postImages(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
}

func postVideos(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
}

func postLikes(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		postID := gp.PostID(_id)
		liked, err := strconv.ParseBool(r.FormValue("liked"))
		switch {
		case err != nil:
			jsonErr(w, err, 400)
		case liked:
			err = api.AddLike(userID, postID)
			if err != nil {
				e, ok := err.(*gp.APIerror)
				if ok && *e == lib.ENOTALLOWED {
					jsonResponse(w, e, 403)
				} else {
					jsonErr(w, err, 500)
				}
			} else {
				jsonResponse(w, gp.Liked{Post: postID, Liked: true}, 200)
			}
		default:
			err = api.DelLike(userID, postID)
			if err != nil {
				jsonErr(w, err, 500)
			} else {
				jsonResponse(w, gp.Liked{Post: postID, Liked: false}, 200)
			}
		}
	}
}

func liveHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		after := r.FormValue("after")
		posts, err := api.UserGetLive(userID, after, api.Config.PostPageSize)
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
}

func attendHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	vars := mux.Vars(r)
	//We can safely ignore this error since vars["id"] matches a numeric regex
	//... maybe. What if it's bigger than max(uint64) ??
	_id, _ := strconv.ParseUint(vars["id"], 10, 64)
	post := gp.PostID(_id)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
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

func deletePost(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
		w.WriteHeader(204)
	}
}

func putAttendees(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		attending, _ := strconv.ParseBool(r.FormValue("attending"))
		vars := mux.Vars(r)
		_postID, _ := strconv.ParseUint(vars["id"], 10, 64)
		postID := gp.PostID(_postID)
		err = api.UserAttend(postID, userID, attending)
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

func getAttendees(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
