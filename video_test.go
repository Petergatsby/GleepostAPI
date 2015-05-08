package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/mail"
)

func TestVideo(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	config := conf.GetConfig()
	api = lib.New(*config)
	api.Mail = mail.NewMock()
	api.TW = lib.StubTranscodeWorker{}
	api.Start()
	server := httptest.NewServer(r)
	baseURL = server.URL + "/api/v1/"

	type videoTest struct {
		Token              gp.Token
		Video              string
		ExpectedType       string
		ExpectedStatusCode int
		ExpectedError      string
	}
	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatal("Error logging in:", err)
	}
	tests := []videoTest{
		{
			Token:              token,
			Video:              "lib/transcode/testdata/bridge.mp4",
			ExpectedType:       "UploadStatus",
			ExpectedStatusCode: http.StatusCreated,
		},
	}
	url := baseURL + "videos"
	for _, test := range tests {
		req, err := newfileUploadRequest(url, nil, "video", test.Video)
		if err != nil {
			t.Fatalf("Problem building request: %s\n", err)
		}
		req.Header.Set("X-GP-Auth", fmt.Sprintf("%d-%s", test.Token.UserID, test.Token.Token))
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Problem making upload request: %s\n", err)
		}
		if resp.StatusCode != test.ExpectedStatusCode {
			t.Fatalf("Received unexpected status code: %d (expecting %d)\n", resp.StatusCode, test.ExpectedStatusCode)
		}
		status := gp.UploadStatus{}
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&status)
		if err != nil {
			t.Fatal("Problem decoding response json:", err)
		}
		if status.ID == 0 {
			t.Fatal("Upload should return a nonzero ID")
		}
		if status.Status != "uploaded" {
			t.Fatal("Upload status should be 'uploaded'")
		}
	}
}

func newfileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	return req, err
}
