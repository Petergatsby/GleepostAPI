package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
)

//NoSuchUpload = You tried to attach a URL you didn't upload to tomething
var NoSuchUpload = gp.APIerror{Reason: "That upload doesn't exist"}

func init() {
	base.HandleFunc("/upload", uploadHandler)
	base.HandleFunc("/upload/{id}", getUpload)
	base.HandleFunc("/videos", postVideoUpload).Methods("POST")
	base.HandleFunc("/videos/{id}", getVideos).Methods("GET")
}

func postVideoUpload(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.videos.put")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		shouldRotate, _ := strconv.ParseBool(r.FormValue("rotate"))
		log.Println("Entering upload handler")
		file, header, err := r.FormFile("video")
		log.Println("Got file from request body")
		if err != nil {
			jsonErr(w, err, 400)
			return
		}
		defer file.Close()
		log.Println("About to enqueue the video for processing...")
		videoStatus, err := api.EnqueueVideo(userID, file, header, shouldRotate)
		if err != nil {
			jsonErr(w, err, 400)
		} else {
			jsonResponse(w, videoStatus, 201)
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func getVideos(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.videos.*.get")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		vars := mux.Vars(r)
		_id, err := strconv.ParseUint(vars["id"], 10, 64)
		if err != nil {
			_id = 0
		}
		videoID := gp.VideoID(_id)
		upload, err := api.GetUploadStatus(userID, videoID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, upload, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func getUpload(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.uploads.*.get")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "GET":
		vars := mux.Vars(r)
		_id, err := strconv.ParseUint(vars["id"], 10, 64)
		if err != nil {
			_id = 0
		}
		videoID := gp.VideoID(_id)
		upload, err := api.GetUploadStatus(userID, videoID)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		jsonResponse(w, upload, 200)
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	defer api.Time(time.Now(), "gleepost.uploads.post")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method == "POST":
		file, header, err := r.FormFile("image")
		if err != nil {
			file, header, err = r.FormFile("video")
			if err != nil {
				jsonErr(w, err, 400)
				return
			}
		}
		defer file.Close()
		url, err := api.StoreFile(userID, file, header)
		if err != nil {
			jsonErr(w, err, 400)
		} else {
			jsonResponse(w, gp.URLCreated{URL: url}, 201)
		}
	default:
		jsonResponse(w, &EUNSUPPORTED, 405)
	}
}
