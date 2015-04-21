package transcode

import "testing"

type transcodeTest struct {
	id            uint64
	filepath      string
	target        string
	rotate        bool
	expectedError error
}

func TestTranscode(t *testing.T) {
	queue := NewTranscoder()
	results := queue.Results()
	tests := []transcodeTest{
		{
			id:       1,
			filepath: "./testdata/bridge.mp4",
			target:   "webm",
			rotate:   false,
		},
		{
			id:       2,
			filepath: "./testdata/bridge.mp4",
			target:   "webm",
			rotate:   true,
		},
		{
			id:       3,
			filepath: "./testdata/bridge.mp4",
			target:   "jpg",
		},
	}
	for _, test := range tests {
		queue.Enqueue(test.id, test.filepath, test.target, test.rotate)
		res := <-results
		if res.Error != test.expectedError {
			t.Fatal("Error didn't match:", res.Error, test.expectedError)
		}
	}
}
