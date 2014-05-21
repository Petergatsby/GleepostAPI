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

	//Upload

	//Mark as processed

	//Delete temp files
}

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
