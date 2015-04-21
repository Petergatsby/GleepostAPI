package transcode

import (
	"log"
	"mime/multipart"
	"testing"
)

func TestTranscode(t *testing.T) {
	transcoder := NewVideoTranscoder()
	var file multipart.File
	var header *multipart.FileHeader
	statusEvents, err := transcoder.EnqueueVideo(file, header, false)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case evt := <-statusEvents:
		log.Println("Something happened!", evt)
	default:

	}
}
