package lib

import (
	"database/sql"
	"io/ioutil"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
)

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

func inferContentType(header *multipart.FileHeader) (contentType string) {
	_contenttype, ok := header.Header["Content-Type"]
	if ok && len(_contenttype) > 0 {
		contentType = _contenttype[0]
	}
	return
}

//StoreFile takes an uploaded file, checks if it is allowed (ie, is jpg / png / appropriate video file) and uploads it to s3 (selecting a bucket based on the user who uploaded it).
func (api *API) StoreFile(id gp.UserID, file multipart.File, header *multipart.FileHeader) (url string, err error) {
	contenttype := inferContentType(header)
	ext := filepath.Ext(header.Filename)
	filename := randomFilename(ext)
	bucket := api.getBucket(id)
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	err = bucket.Put(filename, data, contenttype, s3.PublicRead)
	url = bucket.URL(filename)
	if err != nil {
		return "", err
	}
	url = cloudfrontify(url)
	err = api.userAddUpload(id, url)
	return url, err
}

var cloudfronts = map[string]string{"gpcali": "https://d3itv2rmlfeij9.cloudfront.net/", "gpimg": "https://d2tc2ce3464r63.cloudfront.net/"}

func cloudfrontify(url string) (cdnurl string) {
	for bucket, cloudfront := range cloudfronts {
		if strings.Contains(url, bucket) {
			bits := strings.Split(url, "/")
			final := bits[len(bits)-1]
			return cloudfront + final
		}
	}
	return url
}

//userAddUpload records that this user has uploaded this URL.
func (api *API) userAddUpload(id gp.UserID, url string) (err error) {
	s, err := api.sc.Prepare("INSERT INTO uploads (user_id, url) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(id, url)
	return
}

//UserUploadExists returns true if the user has uploaded the file at url
func (api *API) userUploadExists(id gp.UserID, url string) (exists bool, err error) {
	s, err := api.sc.Prepare("SELECT COUNT(*) > 0 FROM uploads WHERE user_id = ? AND url = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id, url).Scan(&exists)
	return
}

//GetUploadStatus returns the current status of this upload.
//That's one of "uploaded", "transcode", "transfer", "done".
func (api *API) GetUploadStatus(user gp.UserID, upload gp.VideoID) (UploadStatus gp.UploadStatus, err error) {
	s, err := api.sc.Prepare("SELECT status, mp4_url, webm_url, url FROM uploads WHERE upload_id = ?")
	if err != nil {
		return
	}
	var status, mp4URL, webmURL, URL sql.NullString
	err = s.QueryRow(upload).Scan(&status, &mp4URL, &webmURL, &URL)
	if err != nil {
		return
	}
	if status.Valid {
		UploadStatus.Status = status.String
	}
	if mp4URL.Valid {
		UploadStatus.MP4 = mp4URL.String
	}
	if webmURL.Valid {
		UploadStatus.WebM = webmURL.String
	}
	if URL.Valid {
		UploadStatus.Thumbs = append(UploadStatus.Thumbs, URL.String)
	}
	UploadStatus.ID = upload
	return
}

//SetUploadStatus records the current status of this upload.
//Status must be one of "uploaded", "transcode", "transfer", "done".
//If provided, urls[0] will be its mp4 format and urls[1] its webm..
func (api *API) setUploadStatus(uploadStatus gp.UploadStatus) (ID gp.VideoID, err error) {
	defer api.Statsd.Time(time.Now(), "gleepost.uploads.setStatus.db")
	var q string
	var s *sql.Stmt
	if uploadStatus.ID == 0 {
		q = "INSERT INTO uploads(user_id, type, status) VALUES(?, 'video', ?)"
	} else {
		q = "REPLACE INTO uploads(user_id, type, status, mp4_url, webm_url, url, upload_id) VALUES (?, 'video', ?, ?, ?, ?, ?)"
		ID = uploadStatus.ID
	}
	s, err = api.sc.Prepare(q)
	if err != nil {
		return
	}
	thumb := ""
	if len(uploadStatus.Thumbs) > 0 {
		thumb = uploadStatus.Thumbs[0]
	}
	var res sql.Result
	switch {
	case uploadStatus.ID == 0:
		//First time, create an ID
		res, err = s.Exec(uploadStatus.Owner, uploadStatus.Status)
		_ID, _ := res.LastInsertId()
		ID = gp.VideoID(_ID)
	case uploadStatus.Uploaded == true:
		//If it's done, record the urls of the files
		res, err = s.Exec(uploadStatus.Owner, uploadStatus.Status, uploadStatus.MP4, uploadStatus.WebM, thumb, uploadStatus.ID)
	default:
		//Otherwise, just update the status.
		res, err = s.Exec(uploadStatus.Owner, uploadStatus.Status, "", "", "", uploadStatus.ID)
	}
	if err != nil {
		log.Println(err)
	}
	return
}

//CreateJob records a Transcoding job into the queue
func (api *API) createJob(source, target string, rotate bool, parent gp.VideoID) (err error) {
	s, err := api.sc.Prepare("INSERT INTO video_jobs(parent_id, source, target, rotate) VALUES (?,?,?,?)")
	if err != nil {
		return
	}
	_, err = s.Exec(parent, source, target, rotate)
	return
}
