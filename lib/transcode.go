package lib

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"os"

	"launchpad.net/goamz/s3"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

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

//TransientStoreFile writes a multipart.File to disk, returning its location.
func transientStoreFile(f multipart.File, ext string) (location string, err error) {
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

func (api *API) getBucket(user gp.UserID) (b *s3.Bucket) {
	primary, _ := api.getUserUniversity(user)
	var s *s3.S3
	var bucket *s3.Bucket
	s = api.getS3(primary.ID)
	if primary.ID == 1911 {
		bucket = s.Bucket("gpcali")
	} else {
		bucket = s.Bucket("gpimg")
	}
	return bucket
}

//EnqueueVideo takes a user-uploaded video and enqueues it for processing.
func (api *API) EnqueueVideo(user gp.UserID, file multipart.File, header *multipart.FileHeader, shouldRotate bool) (inProgress gp.UploadStatus, err error) {
	_, ext := inferContentType(header.Filename)
	if ext == "" {
		return inProgress, errors.New("unsupported video type")
	}
	//Saved locally because PutReader needs a content-length, which tw.upload will get from the saved file.
	filePath, err := transientStoreFile(file, ext)
	if err != nil {
		log.Println("Problem temp saving file:", err)
		return inProgress, err
	}
	url, err := api.tw.upload(filePath)
	if err != nil {
		return inProgress, err
	}
	err = os.Remove(filePath)
	if err != nil {
		log.Println("Error removing file:", filePath, err)
	}
	video := gp.UploadStatus{}
	video.ShouldRotate = shouldRotate
	video.Status = "uploaded"
	video.Owner = user
	id, err := api.setUploadStatus(video)
	if err != nil {
		return video, err
	}
	err = api.createJob(url, "webm", shouldRotate, id)
	if err != nil {
		return video, err
	}
	err = api.createJob(url, "jpg", shouldRotate, id)
	if err != nil {
		return video, err
	}
	video.ID = id
	video.MP4 = url
	video.Uploaded = true
	_, err = api.setUploadStatus(video)
	if err != nil {
		log.Println("Error saving mp4 url:", err)
	}
	video.MP4 = ""
	return video, nil
}
