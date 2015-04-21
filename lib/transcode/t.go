package transcode

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os/exec"
)

//Queue represents a single queueing transcoder. You may Enqueue jobs with an arbitrary ID, an input file and a target filetype.
//Results() returns the results for the enqueued jobs as they happen.
type Queue interface {
	Enqueue(id uint64, file string, target string)
	Results() (results <-chan Result)
}

//Result maps an arbitrary ID to a (probably temp) filename or an error.
type Result struct {
	ID   uint64
	file string
	err  error
}

type job struct {
	ID     uint64
	source string
	target string
	rotate bool
}

func newJob(id uint64, source, target string) (j job) {
	j = job{ID: id, source: source, target: target}
	return
}

func (j job) do() (res Result) {
	res.ID = j.ID
	var cmd *exec.Cmd
	switch {
	case j.target == "webm":
		res.file = "/tmp/" + randomFilename(".webm")
		if j.rotate {
			cmd = exec.Command("ffmpeg", "-i", j.source, "-codec:v", "libvpx", "-quality", "good", "-cpu-used", "0", "-b:v", "500k", "-qmin", "10", "-qmax", "42", "-maxrate", "500k", "-bufsize", "1000k", "-threads", "6", "-vf", "scale=-1:480", "transpose=1", "-codec:a", "libvorbis", "-b:a", "128k", "-ac", "2", "-f", "webm", res.file)
		} else {
			cmd = exec.Command("ffmpeg", "-i", j.source, "-codec:v", "libvpx", "-quality", "good", "-cpu-used", "0", "-b:v", "500k", "-qmin", "10", "-qmax", "42", "-maxrate", "500k", "-bufsize", "1000k", "-threads", "6", "-vf", "scale=-1:480", "-codec:a", "libvorbis", "-b:a", "128k", "-ac", "2", "-f", "webm", res.file)
		}
	case j.target == "jpg":
		res.file = "/tmp/" + randomFilename(".jpg")
		cmd = exec.Command("ffmpeg", "-ss", "00:00:00", "-i", j.source, "-frames:v", "1", res.file)
	}
	res.err = cmd.Run()
	return
}

type transcodeQueue struct {
	jobs    chan job
	results chan Result
}

//NewTranscoder returns a transcodeQueue. A process should probably only use one of these as they are already optimized to transcode in parallel where possible.
func NewTranscoder() Queue {
	queue := transcodeQueue{}
	queue.jobs = make(chan job)
	queue.results = make(chan Result)
	go queue.process()
	return queue
}

func (q transcodeQueue) Enqueue(id uint64, file, target string) {
	j := newJob(id, file, target)
	q.jobs <- j
}

func (q transcodeQueue) process() {
	for {
		j := <-q.jobs
		res := j.do()
		q.results <- res
	}
}

//Results returns a chan of Result, upon which all transcode results are sent.
//This queue MUST be consumed, and the consumer must deal with the temporary files in each Result.
func (q transcodeQueue) Results() <-chan Result {
	return q.results
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
	return ""
}
