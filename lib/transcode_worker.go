package lib

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"launchpad.net/goamz/s3"

	"github.com/draaglom/GleepostAPI/lib/cache"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/transcode"
)

type transcodeWorker struct {
	db    *sql.DB
	tq    transcode.Queue
	b     *s3.Bucket
	cache *cache.Cache
}

type TranscodeWorker interface {
	upload(file string) (url string, err error)
	claimLoop()
	handleDone()
}

func newTranscodeWorker(db *sql.DB, tq transcode.Queue, b *s3.Bucket, cache *cache.Cache) (t TranscodeWorker) {
	t = transcodeWorker{db: db, tq: tq, b: b, cache: cache}
	go t.claimLoop()
	go t.handleDone()
	return
}

type StubTranscodeWorker struct {
}

func (s StubTranscodeWorker) upload(file string) (url string, err error) {
	return "https://gleepost.com/images/sm-logo.png", nil
}

func (s StubTranscodeWorker) claimLoop() {
	return
}

func (s StubTranscodeWorker) handleDone() {
	return
}

func (t transcodeWorker) claimJobs() (err error) {
	s, err := t.db.Prepare("SELECT id, source, target, rotate FROM `video_jobs` WHERE completion_time IS NULL AND (claim_time IS NULL OR claim_time < ?)")
	if err != nil {
		return
	}
	defer s.Close()
	since := time.Now().UTC().Add(-30 * time.Second)
	rows, err := s.Query(since)
	if err != nil {
		return
	}
	defer rows.Close()
	claimStmt, err := t.db.Prepare("UPDATE `video_jobs` SET claim_time = NOW() WHERE id = ?")
	if err != nil {
		return
	}
	defer claimStmt.Close()
	for rows.Next() {
		var id uint64
		var source, target string
		var rotate bool
		err = rows.Scan(&id, &source, &target, &rotate)
		if err != nil {
			return
		}
		_, err = claimStmt.Exec(id)
		if err != nil {
			return
		}
		//Note to self: get the source first!
		client := http.Client{}
		var resp *http.Response
		resp, err = client.Get(source)
		if err != nil {
			return
		}
		_, ext := inferContentType(source)
		location := "/tmp/" + randomFilename(ext)
		var tmp *os.File
		tmp, err = os.Create(location)
		if err != nil {
			return
		}
		_, err = io.Copy(tmp, resp.Body)
		if err != nil {
			return
		}
		tmp.Close()
		t.tq.Enqueue(id, location, target, rotate)
	}
	return
}

func (t transcodeWorker) claimLoop() {
	tick := time.Tick(500 * time.Millisecond)
	for {
		err := t.claimJobs()
		if err != nil {
			log.Println("Error claiming some transcode jobs?", err)
		}
		<-tick
	}
}

func (t transcodeWorker) handleDone() {
	results := t.tq.Results()
	for res := range results {
		if res.Error != nil {
			log.Println("There was an error transcoding this file:", res.Error)
			continue
		}
		url, err := t.upload(res.File)
		if err != nil {
			log.Println("There was an error uploading this file:", err)
			err = os.Remove(res.File)
			if err != nil {
				log.Println("Error removing tmp file:", err)
			}
			continue
		}
		//Mark job "done"
		err = t.done(res.ID, url)
		if err != nil {
			log.Println("Couldn't mark job as done:", err)
		}
		err = os.Remove(res.File)
		if err != nil {
			log.Println("Error removing tmp file:", err)
		}
		t.maybeReady(res.ID)
	}
}

func (t transcodeWorker) maybeReady(jobID uint64) {
	var video gp.UploadStatus
	var thumb string
	err := t.db.QueryRow("SELECT upload_id, url, mp4_url, webm_url, user_id FROM uploads JOIN video_jobs ON upload_id = video_jobs.parent_id WHERE url IS NOT NULL AND mp4_url IS NOT NULL AND webm_url IS NOT NULL AND video_jobs.id = ?", jobID).Scan(&video.ID, &thumb, &video.MP4, &video.WebM, &video.Owner)
	if err != nil {
		log.Println("Error getting parent video:", err)
		return
	}
	_, err = t.db.Exec("UPDATE uploads SET status = 'ready' WHERE upload_id = ?", video.ID)
	if err != nil {
		log.Println("Error marking ready:", err)
		return
	}
	video.Status = "ready"
	t.cache.PublishEvent("video-ready", fmt.Sprintf("/videos/%d", video.ID), video, []string{NotificationChannelKey(video.Owner)})

}

func (t transcodeWorker) done(jobID uint64, URL string) (err error) {
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
		q = "UPDATE uploads SET mp4_url = ? WHERE upload_id = (SELECT parent_id FROM video_jobs WHERE id = ?)"
	case fileType == "webm":
		q = "UPDATE uploads SET webm_url = ? WHERE upload_id = (SELECT parent_id FROM video_jobs WHERE id = ?)"
	case fileType == "jpg":
		q = "UPDATE uploads SET url = ? WHERE upload_id = (SELECT parent_id FROM video_jobs WHERE id = ?)"
	}

	_, err = t.db.Exec(q, URL, jobID)
	return
}

func (t transcodeWorker) upload(file string) (URL string, err error) {
	contentType, _ := inferContentType(file)
	if contentType == "" {
		err = errors.New("Couldn't determine content-type")
		return
	}
	url, err := upload(file, contentType, t.b)
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
