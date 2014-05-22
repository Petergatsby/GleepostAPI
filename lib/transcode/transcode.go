package trans

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"os"
	"os/exec"
	"strings"

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
	} else {
		log.Println(err)
		return ""
	}
}

func HandleVideoUpload(file multipart.File, header multipart.FileHeader) (ID gp.VideoID, err error) {
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
		return ID, errors.New("Unsupported video type.")
	}
	name, err := TransientStoreFile(file, ext)
	if err != nil {
		return
	}
	//Then we'll pass off the video for transcoding / thumbnail extraction
	video := gp.Video{}
	switch {
	case ext == ".mp4":
		video.MP4 = name
	case ext == ".webm":
		video.WebM = name
	}
	go pipeline(video)
	return video.ID, nil

}

func pipeline(inProgress gp.Video) {
	var err error
	//Transcode mp4 to webm
	if inProgress.MP4 != "" {
		inProgress.WebM, err = MP4ToWebM(inProgress.MP4)
		if err != nil {
			log.Println(err)
			return
		}
	}
	//Extract initial thumb
	thumb, err := MP4Thumb(inProgress.MP4)
	if err != nil {
		log.Println(err)
		return
	}
	inProgress.Thumbs = append(inProgress.Thumbs, thumb)
	//Work out which bucket to put it in
	//Upload
	uploaded, err := Upload(inProgress)
	//Mark as processed

	//Delete temp files
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
func Upload(v gp.Video, b s3.Bucket) (uploaded gp.Video, err error) {
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

func upload(path, contentType string, b s3.Bucket) (url string, err error) {
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
