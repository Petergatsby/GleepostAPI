package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

func init() {
	base.HandleFunc("/posts", getPosts).Methods("GET")
	base.HandleFunc("/posts", postPosts).Methods("POST")
	base.HandleFunc("/posts/{id:[0-9]+}/comments", getComments).Methods("GET")
	base.HandleFunc("/posts/{id:[0-9]+}/comments", postComments).Methods("POST")
	base.HandleFunc("/posts/{id:[0-9]+}", getPost).Methods("GET")
	base.HandleFunc("/posts/{id:[0-9]+}/", getPost).Methods("GET")
	base.HandleFunc("/posts/{id:[0-9]+}/", deletePost).Methods("DELETE")
	base.HandleFunc("/posts/{id:[0-9]+}", deletePost).Methods("DELETE")
	base.HandleFunc("/posts/{id:[0-9]+}/images", postImages).Methods("POST")
	base.HandleFunc("/posts/{id:[0-9]+}/videos", postVideos).Methods("POST")
	base.HandleFunc("/posts/{id:[0-9]+}/likes", postLikes).Methods("POST")
	base.HandleFunc("/posts/{id:[0-9]+}/attendees", getAttendees).Methods("GET")
	base.HandleFunc("/posts/{id:[0-9]+}/attendees", putAttendees).Methods("PUT")
	base.HandleFunc("/posts/{id:[0-9]+}/attending", attendHandler)
	base.HandleFunc("/live", liveHandler)
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

func getPosts(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.posts.get")
	userID, err := authenticate(r)
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
		var network gp.NetworkID
		switch {
		case ok:
			_network, err := strconv.ParseUint(id, 10, 64)
			if err != nil {
				jsonErr(w, err, 500)
				return
			}
			network = gp.NetworkID(_network)
		default: //We haven't been given a network, which means this handler is being called by /posts and we just want the users' default network
			networks, err := api.GetUserNetworks(userID)
			if err != nil {
				jsonErr(w, err, 500)
				return
			}
			network = networks[0].ID
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
		posts, err = api.UserGetNetworkPosts(userID, network, mode, index, api.Config.PostPageSize, filter)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
			return
		}
		jsonResponse(w, posts, 200)
	}
}

func postPosts(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.posts.post")
	userID, err := authenticate(r)
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
		var postID gp.PostID
		var ts []string
		if len(tags) > 1 {
			ts = strings.Split(tags, ",")
		}
		n, ok := vars["network"]
		var network gp.NetworkID
		if !ok {
			networks, err := api.GetUserNetworks(userID)
			if err != nil {
				jsonErr(w, err, 500)
				return
			}
			network = networks[0].ID
		} else {
			_network, err := strconv.ParseUint(n, 10, 64)
			if err != nil {
				jsonErr(w, err, 500)
				return
			}
			network = gp.NetworkID(_network)
		}
		_vID, _ := strconv.ParseUint(r.FormValue("video"), 10, 64)
		videoID := gp.VideoID(_vID)
		switch {
		case videoID > 0:
			postID, err = api.AddPostWithVideo(userID, network, text, attribs, videoID, ts...)
		case len(url) > 5:
			postID, err = api.AddPostWithImage(userID, network, text, attribs, url, ts...)
		default:
			postID, err = api.AddPost(userID, network, text, attribs, ts...)
		}
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
		} else {
			jsonResponse(w, &gp.Created{ID: uint64(postID)}, 201)
		}
	}
}

func getComments(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.posts.*.comments.get")
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
	defer api.Time(time.Now(), "gleepost.posts.*.comments.post")
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
	defer api.Time(time.Now(), "gleepost.posts.*.get")
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

func postImages(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.posts.*.images.post")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_id, _ := strconv.ParseUint(vars["id"], 10, 64)
		postID := gp.PostID(_id)
		url := r.FormValue("url")
		exists, err := api.UserUploadExists(userID, url)
		if exists && err == nil {
			err := api.UserAddPostImage(userID, postID, url)
			if err != nil {
				if err == lib.ENOTALLOWED {
					jsonErr(w, err, 403)
				} else {
					jsonErr(w, err, 500)
				}
			} else {
				images := api.GetPostImages(postID)
				jsonResponse(w, images, 201)
			}
		} else {
			jsonErr(w, NoSuchUpload, 400)
		}
	}
}

func postVideos(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.posts.*.videos.post")
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
		err = api.UserAddPostVideo(userID, postID, videoID)
		if err != nil {
			jsonErr(w, err, 500)
		} else {
			videos := api.GetPostVideos(postID)
			jsonResponse(w, videos, 201)
		}
	}
}

func postLikes(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.posts.*.likes.post")
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
	defer api.Time(time.Now(), "gleepost.posts.live..get")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
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
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
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
	case r.Method == "GET":
		//Implement
	case r.Method == "POST":
		defer api.Time(time.Now(), "gleepost.posts.*.attending.post")
		//For now, assume that err is because the user specified a bad post.
		//Could also be a db error.
		err := api.UserAttend(post, userID, true)
		if err != nil {
			jsonResponse(w, err, 400)
		}
		w.WriteHeader(204)
	case r.Method == "DELETE":
		defer api.Time(time.Now(), "gleepost.posts.*.attending.delete")
		//For now, assume that err is because the user specified a bad post.
		//Could also be a db error.
		err := api.UserAttend(post, userID, false)
		if err != nil {
			jsonResponse(w, err, 400)
		}
		w.WriteHeader(204)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func deletePost(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.posts.*.delete")
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
	defer api.Time(time.Now(), "gleepost.posts.*.attendees.put")
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
	defer api.Time(time.Now(), "gleepost.posts.*.attendees.get")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		vars := mux.Vars(r)
		_postID, _ := strconv.ParseUint(vars["id"], 10, 64)
		postID := gp.PostID(_postID)
		attendees, err := api.UserGetEventAttendees(userID, postID)
		if err != nil {
			e, ok := err.(*gp.APIerror)
			if ok && *e == lib.ENOTALLOWED {
				jsonResponse(w, e, 403)
			} else {
				jsonErr(w, err, 500)
			}
			return
		}
		popularity, attendeeCount, err := api.UserGetEventPopularity(userID, postID)
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
		}{Popularity: popularity, AttendeeCount: attendeeCount, Attendees: attendees}
		jsonResponse(w, resp, 200)

	}
}
