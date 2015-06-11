package main

import (
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/mux"
	"github.com/patdek/gongflow"
)

//NoSuchUpload = You tried to attach a URL you didn't upload to tomething
var NoSuchUpload = gp.APIerror{Reason: "That upload doesn't exist"}

func init() {
	base.Handle("/upload", timeHandler(api, http.HandlerFunc(uploadHandler))).Methods("POST")
	base.Handle("/upload", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/upload/{id}", timeHandler(api, http.HandlerFunc(getUpload))).Methods("GET")
	base.Handle("/upload/{id}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/videos", timeHandler(api, http.HandlerFunc(postVideoUpload))).Methods("POST")
	base.Handle("/videos", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/videos/{id}", timeHandler(api, http.HandlerFunc(getVideos))).Methods("GET")
	base.Handle("/videos/{id}", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
	base.Handle("/flow_upload", timeHandler(api, http.HandlerFunc(ngflowUpload))).Methods("GET", "POST")
	base.Handle("/flow_upload", timeHandler(api, http.HandlerFunc(optionsHandler))).Methods("OPTIONS")
	base.Handle("/flow_upload", timeHandler(api, http.HandlerFunc(unsupportedHandler)))
}

func postVideoUpload(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		shouldRotate, _ := strconv.ParseBool(r.FormValue("rotate"))
		file, header, err := r.FormFile("video")
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
	}
}

func getVideos(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
	}
}

func getUpload(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
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
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	default:
		file, header, err := r.FormFile("image")
		if err != nil {
			file, header, err = r.FormFile("file")
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
	}
}

var tempPath = path.Join(os.TempDir(), "gongflow")

func init() {
	// ensure the tempPath exists
	os.MkdirAll(tempPath, 0777)
}

func ngflowUpload(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticate(r)
	ngFlowData, errFlow := gongflow.ChunkFlowData(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case errFlow != nil:
		log.Println(r.FormValue("flowChunkNumber"))
		jsonErr(w, errFlow, 500)
	case r.Method == "GET":
		msg, code := gongflow.ChunkStatus(tempPath, ngFlowData)
		jsonResponse(w, msg, code)
	case r.Method == "POST":
		filePath, err := gongflow.ChunkUpload(tempPath, ngFlowData, r)
		if err != nil {
			jsonErr(w, err, 500)
			return
		}
		if filePath != "" {
			url, err := api.StoreFilePath(userID, filePath)
			if err != nil {
				jsonErr(w, err, 400)
			}
			jsonResponse(w, gp.URLCreated{URL: url}, 201)
			return
		}
		jsonResponse(w, "continuing to upload chunks", 200)
	}
}

func cleanupUploads() {
	loopDur := time.Duration(1) * time.Minute   // loop every minute
	tooOldDur := time.Duration(5 * time.Minute) // older than 5 minutes to be deleted
	t := time.NewTicker(loopDur)
	for _ = range t.C {
		err := gongflow.ChunksCleanup(tempPath, tooOldDur)
		if err != nil {
			log.Println(err)
		}
	}
}
