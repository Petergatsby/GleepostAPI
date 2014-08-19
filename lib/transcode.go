package lib

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"os/exec"
	"strings"

	"launchpad.net/goamz/s3"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

var transcodeQueue chan gp.UploadStatus

func init() {
	transcodeQueue = make(chan gp.UploadStatus, 100)
}

func (api *API) process(vids chan gp.UploadStatus) {
	for inProgress := range vids {
		api.pipeline(inProgress)
	}
}

func randomFilename(extension string) string {
	hash := sha256.New()
	random := make([]byte, 32) //Number pulled out of my... ahem.
	_, err := io.ReadFull(rand.Reader, random)
	if err == nil {
		hash.Write(random)
		digest := hex.EncodeToString(hash.Sum(nil))
		return digest + extension
	}
	log.Println(err)
	return ""
}

func (api *API) pipeline(inProgress gp.UploadStatus) {
	log.Println("Initial state:", inProgress)
	inProgress.Status = "transcoding"
	api.SetUploadStatus(inProgress)
	var err error
	//Transcode mp4 to webm
	if inProgress.MP4 != "" {
		inProgress.WebM, err = MP4ToWebM(inProgress.MP4)
		if err != nil {
			log.Println(err)
			return
		}
	}
	log.Println("State after transcode to webM:", inProgress)
	//Extract initial thumb
	thumb, err := MP4Thumb(inProgress.MP4)
	if err != nil {
		log.Println(err)
		return
	}
	inProgress.Thumbs = append(inProgress.Thumbs, thumb)
	log.Println("State after extracting thumb:", inProgress)
	//Upload
	inProgress.Status = "transferring"
	api.SetUploadStatus(inProgress)
	uploaded, err := api.Upload(inProgress)
	if err != nil {
		log.Println("Upload error:", err)
	}
	log.Println("State after uploading:", uploaded)
	//Mark as processed
	uploaded.Status = "ready"
	id, err := api.SetUploadStatus(uploaded)
	if err != nil {
		log.Println(id, err)
	}
	//Emit "Done" event
	api.cache.PublishEvent("video-ready", fmt.Sprintf("/videos/%d", uploaded.ID), uploaded, []string{NotificationChannelKey(uploaded.Owner)})
	//Delete temp files
	err = del(inProgress)
	if err != nil {
		log.Println("Error cleaning up temp files:", err)
	}
}

//MP4ToWebM converts an MP4 video to WebM, returning the path to the output video.
//The caller is responsible for cleaning up after itself (ie, deleting the videos from local storage when it is done)
func MP4ToWebM(in string) (output string, err error) {
	//do transcode
	output = "/tmp/" + randomFilename(".webm")
	log.Println("Creating ffmpeg command")
	cmd := exec.Command("ffmpeg", "-i", in, "-codec:v", "libvpx", "-quality", "good", "-cpu-used", "0", "-b:v", "500k", "-qmin", "10", "-qmax", "42", "-maxrate", "500k", "-bufsize", "1000k", "-threads", "6", "-vf", "scale=-1:480", "-codec:a", "libvorbis", "-b:a", "128k", "-ac", "2", "-f", "webm", output)
	err = cmd.Run()
	if err != nil {
		return
	}
	//hand back temp file?
	return output, nil
}

//MP4Thumb attempts to extract a thumbnail from the first second of a video at path `in`, returning the path for the thumbnail
//The caller is responsible for cleaning up after itself (ie, deleting the files from local storage when it is done)
func MP4Thumb(in string) (output string, err error) {
	output = "/tmp/" + randomFilename(".jpg")
	log.Println("Extracting thumbnail")
	cmd := exec.Command("ffmpeg", "-ss", "00:00:01", "-i", in, "-frames:v", "1", output)
	err = cmd.Run()
	return
}

//TransientStoreFile writes a multipart.File to disk, returning its location.
func TransientStoreFile(f multipart.File, ext string) (location string, err error) {
	location = "/tmp/" + randomFilename(ext)
	tmp, err := os.Create(location)
	if err != nil {
		return
	}
	defer tmp.Close()
	_, err = io.Copy(tmp, f)
	if err != nil {
		return
	}
	return location, nil
}

//Upload sends all versions and thumbnails of a Video to the bucket b.
func (api *API) Upload(v gp.UploadStatus) (uploaded gp.UploadStatus, err error) {
	b := api.getBucket(v.Owner)
	v.MP4, err = upload(v.MP4, "video/mp4", b)
	if err != nil {
		return
	}
	v.WebM, err = upload(v.WebM, "video/webm", b)
	if err != nil {
		return
	}
	var ts []string
	var url string
	for _, i := range v.Thumbs {
		url, err = upload(i, "image/jpeg", b)
		if err != nil {
			return
		}
		ts = append(ts, url)
	}
	v.Thumbs = ts
	v.Uploaded = true
	return v, nil
}

func (api *API) getBucket(user gp.UserID) (b *s3.Bucket) {
	networks, _ := api.GetUserNetworks(user)
	var s *s3.S3
	var bucket *s3.Bucket
	switch {
	case len(networks) > 0:
		s = api.getS3(networks[0].ID)
		if networks[0].ID == 1911 {
			bucket = s.Bucket("gpcali")
		} else {
			bucket = s.Bucket("gpimg")
		}
	default:
		s = api.getS3(1)
		bucket = s.Bucket("gpimg")
	}
	return bucket
}

func upload(path, contentType string, b *s3.Bucket) (url string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	fi, err := file.Stat()
	if err != nil {
		return
	}
	//The [5:] is assuming all files will be in "/tmp/" (so it extracts their filename from their full path)
	err = b.PutReader(path[5:], file, fi.Size(), contentType, s3.PublicRead)
	if err != nil {
		return
	}
	url = b.URL(path[5:])
	return
}

//del removes all temp files associated with this video
func del(v gp.UploadStatus) (err error) {
	err = os.Remove(v.MP4)
	if err != nil {
		return
	}
	err = os.Remove(v.WebM)
	if err != nil {
		return
	}
	for _, v := range v.Thumbs {
		err = os.Remove(v)
		if err != nil {
			return
		}
	}
	return nil
}

//EnqueueVideo takes a user-uploaded video and enqueues it for processing.
func (api *API) EnqueueVideo(user gp.UserID, file multipart.File, header *multipart.FileHeader) (inProgress gp.UploadStatus, err error) {
	//First we must copy the file to /tmp,
	//because while ffmpeg _can_ operate on a stream directly,
	//that throws away the video length.
	var ext string
	switch {
	case strings.HasSuffix(header.Filename, ".mp4"):
		ext = ".mp4"
	case strings.HasSuffix(header.Filename, ".webm"):
		ext = ".webm"
	default:
		return inProgress, errors.New("unsupported video type")
	}
	log.Println("Storing the video in /tmp for now")
	name, err := TransientStoreFile(file, ext)
	if err != nil {
		return
	}
	//Then we'll pass off the video for transcoding / thumbnail extraction
	video := gp.UploadStatus{}
	switch {
	case ext == ".mp4":
		video.MP4 = name
	case ext == ".webm":
		video.WebM = name
	}
	video.Status = "uploaded"
	video.Owner = user
	log.Println("Recording upload status")
	id, err := api.SetUploadStatus(video)
	if err != nil {
		return video, err
	} else {
		video.ID = id
		go api.enqueueVideo(video)
		video.MP4 = ""
		video.WebM = ""
		return video, nil
	}
}

func (api *API) enqueueVideo(video gp.UploadStatus) {
	api.SetUploadStatus(video)
	transcodeQueue <- video
	return
}
