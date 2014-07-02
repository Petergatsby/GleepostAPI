package lib

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"mime/multipart"
	"strings"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
)

func randomFilename(extension string) (string, error) {
	hash := sha256.New()
	random := make([]byte, 32) //Number pulled out of my... ahem.
	_, err := io.ReadFull(rand.Reader, random)
	if err == nil {
		hash.Write(random)
		digest := hex.EncodeToString(hash.Sum(nil))
		return digest + extension, nil
	}
	return "", err
}

func (api *API) getS3(network gp.NetworkID) (s *s3.S3) {
	var auth aws.Auth
	auth.AccessKey, auth.SecretKey = api.Config.AWS.KeyID, api.Config.AWS.SecretKey
	//1911 == Stanford.
	//TODO: Make the bucket a property of the university / group of universities
	if network == 1911 {
		s = s3.New(auth, aws.USWest)
	} else {
		s = s3.New(auth, aws.EUWest)
	}
	return
}

//StoreFile takes an uploaded file, checks if it is allowed (ie, is jpg / png / appropriate video file) and uploads it to s3 (selecting a bucket based on the user who uploaded it).
func (api *API) StoreFile(id gp.UserID, file multipart.File, header *multipart.FileHeader) (url string, err error) {
	var filename string
	var contenttype string
	switch {
	case strings.HasSuffix(header.Filename, ".jpg"):
		filename, err = randomFilename(".jpg")
		contenttype = "image/jpeg"
	case strings.HasSuffix(header.Filename, ".jpeg"):
		filename, err = randomFilename(".jpg")
		contenttype = "image/jpeg"
	case strings.HasSuffix(header.Filename, ".png"):
		filename, err = randomFilename(".png")
		contenttype = "image/png"
	case strings.HasSuffix(header.Filename, ".mp4"):
		filename, err = randomFilename(".mp4")
		contenttype = "video/mp4"
	case strings.HasSuffix(header.Filename, ".webm"):
		filename, err = randomFilename(".webm")
		contenttype = "video/webm"
	default:
		return "", gp.APIerror{Reason: "Unsupported file type"}
	}
	if err != nil {
		return "", gp.APIerror{Reason: err.Error()}
	}
	//store on s3
	networks, _ := api.GetUserNetworks(id)
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
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	err = bucket.Put(filename, data, contenttype, s3.PublicRead)
	url = bucket.URL(filename)
	if err != nil {
		return "", err
	}
	err = api.userAddUpload(id, url)
	return url, err
}

func (api *API) userAddUpload(id gp.UserID, url string) (err error) {
	return api.db.AddUpload(id, url)
}

//UserUploadExists returns true if the user has uploaded the file at url
func (api *API) UserUploadExists(id gp.UserID, url string) (exists bool, err error) {
	return api.db.UploadExists(id, url)
}

//GetUploadStatus returns the current status of this upload.
//That's one of "uploaded", "transcode", "transfer", "done".
func (api *API) GetUploadStatus(user gp.UserID, upload gp.VideoID) (UploadStatus gp.UploadStatus, err error) {
	return api.db.GetUploadStatus(user, upload)
}

//SetUploadStatus records the current status of this upload.
//Status must be one of "uploaded", "transcode", "transfer", "done".
//If provided, urls[0] will be its mp4 format and urls[1] its webm..
func (api *API) SetUploadStatus(user gp.UserID, upload gp.VideoID, status string, urls ...string) (id gp.VideoID, err error) {
	return api.db.SetUploadStatus(user, upload, status, urls...)
}
