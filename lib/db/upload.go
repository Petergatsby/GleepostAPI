package db

import (
	"database/sql"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

/********************************************************************
		Upload
********************************************************************/

//AddUpload records that this user has uploaded this URL.
func (db *DB) AddUpload(user gp.UserID, url string) (err error) {
	s, err := db.prepare("INSERT INTO uploads (user_id, url) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(user, url)
	return
}

//UploadExists checks that this user has uploaded this URL.
func (db *DB) UploadExists(user gp.UserID, url string) (exists bool, err error) {
	s, err := db.prepare("SELECT COUNT(*) FROM uploads WHERE user_id = ? AND url = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(user, url).Scan(&exists)
	return
}

//SetUploadStatus records this upload's status ("uploaded", "transcode", "transfer", "done").
func (db *DB) SetUploadStatus(uploadStatus gp.UploadStatus) (ID gp.VideoID, err error) {
	var q string
	var s *sql.Stmt
	if uploadStatus.ID == 0 {
		q = "INSERT INTO uploads(user_id, type, status) VALUES(?, 'video', ?)"
	} else {
		q = "REPLACE INTO uploads(user_id, type, status, mp4_url, webm_url, url) VALUES (?, 'video', ?, ?, ?)"
		ID = uploadStatus.ID
	}
	s, err = db.prepare(q)
	if err != nil {
		return
	}
	thumb := ""
	if len(uploadStatus.Thumbs) > 0 {
		thumb = uploadStatus.Thumbs[1]
	}
	res, err := s.Exec(uploadStatus.Owner, uploadStatus.Status, uploadStatus.MP4, uploadStatus.WebM, thumb)
	if uploadStatus.ID == 0 {
		_ID, _ := res.LastInsertId()
		ID = gp.VideoID(_ID)
	}
	return
}

//GetUploadStatus returns the current state of the upload.
func (db *DB) GetUploadStatus(user gp.UserID, upload gp.VideoID) (uploadStatus gp.UploadStatus, err error) {
	s, err := db.prepare("SELECT status, mp4_url, webm_url, url FROM uploads WHERE upload_id = ?")
	if err != nil {
		return
	}
	var status, mp4URL, webmURL, URL sql.NullString
	err = s.QueryRow(upload).Scan(&status, &mp4URL, &webmURL, &URL)
	if err != nil {
		return
	}
	if status.Valid {
		uploadStatus.Status = status.String
	}
	if mp4URL.Valid {
		uploadStatus.MP4 = mp4URL.String
	}
	if webmURL.Valid {
		uploadStatus.WebM = webmURL.String
	}
	if URL.Valid {
		uploadStatus.Thumbs = append(uploadStatus.Thumbs, URL.String)
	}
	uploadStatus.ID = upload
	return
}
