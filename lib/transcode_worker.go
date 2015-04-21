package lib

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"launchpad.net/goamz/s3"

	"github.com/draaglom/GleepostAPI/lib/transcode"
)

type TranscodeWorker struct {
	db *sql.DB
	tq transcode.Queue
	b  *s3.Bucket
}

func (t TranscodeWorker) claimJobs() (err error) {
	s, err := t.db.Prepare("SELECT id, source, target, rotate FROM `video_jobs` WHERE completion_time IS NULL AND (claim_time IS NULL OR claim_time < ?)")
	if err != nil {
		return
	}
	since := time.Now().UTC().Add(-30 * time.Second)
	rows, err := s.Query(since)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id uint64
		var source, target string
		var rotate bool
		err = rows.Scan(&id, &source, &target, &rotate)
		if err != nil {
			return
		}
		_, err = t.db.Query("UPDATE `video_jobs` SET claim_time = NOW() WHERE id = ?", id)
		if err != nil {
			return
		}
		//Note to self: get the source first!
		t.tq.Enqueue(id, source, target, rotate)
	}
	return
}

func (t TranscodeWorker) claimLoop() {
	tick := time.Tick(500 * time.Millisecond)
	for {
		err := t.claimJobs()
		if err != nil {
			log.Println("Error claiming some transcode jobs?", err)
		}
		<-tick
	}
}

func (t TranscodeWorker) handleDone(b *s3.Bucket) {
	results := t.tq.Results()
	for res := range results {
		if res.Error != nil {
			//Record the error

			continue
		}
		url, err := t.upload(res.File, t.b)
		if err != nil {
			//Record the error
		}
		//Mark job "done"
		err = t.done(res.ID, url)
		if err != nil {
			log.Println("Couldn't mark job as done:", err)
		}
		// - Iff all URLs are ready, set the parent upload as Ready and trigger evt.
	}
}

func (t TranscodeWorker) done(jobID uint64, URL string) (err error) {
	_, err = t.db.Query("UPDATE `video_jobs` SET completion_time = NOW() WHERE id = ?", jobID)
	if err != nil {
		return
	}
	var fileType string
	err = t.db.QueryRow("SELECT target FROM video_jobs WHERE id = ?", jobID).Scan(&fileType)
	if err != nil {
		return
	}
	var q string
	switch {
	case fileType == "mp4":
		q = "UPDATE uploads SET mp4_url = ? WHERE upload_id = SELECT parent_id FROM video_jobs WHERE id = ?"
	case fileType == "webm":
		q = "UPDATE uploads SET webm_url = ? WHERE upload_id = SELECT parent_id FROM video_jobs WHERE id = ?"
	case fileType == "jpg":
		q = "UPDATE uploads SET url = ? WHERE upload_id = SELECT parent_id FROM video_jobs WHERE id = ?"
	}

	_, err = t.db.Exec(q, URL, jobID)
	return
}

func (t TranscodeWorker) upload(file string, b *s3.Bucket) (URL string, err error) {
	var contentType string
	fileExt := strings.SplitAfter(file, ".")
	if len(fileExt) != 1 {
		err = errors.New("Couldn't determine content-type")
		return
	}
	switch {
	case fileExt[0] == "mp4":
		contentType = "video/mp4"
	case fileExt[0] == "jpg" || fileExt[0] == "jpeg":
		contentType = "image/jpeg"
	case fileExt[0] == "webm":
		contentType = "video/webm"
	}
	url, err := upload(file, contentType, b)
	if err != nil {
		return
	}

	URL = cloudfrontify(url)
	return
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

/*
TODO

- Stick a db.DB in the TranscodeWorker (so can get the upload status)
- And an api.Cache (so we can broadcast the done event)
- Check that all URLs != nil in the done handler && mark the video ready if so
- Broadcast the vieo-ready event
- Add intervening stage, GETing the remote file URL before passing it to the transcode worker
- On upload, create an Upload and the appropriate Jobs
- Delete tmp files after upload!

*/
