package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/mail"
)

func liveInit() error {
	err := initDB()
	if err != nil {
		return err
	}

	client := &http.Client{}

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		return err
	}
	for i := 0; i < 100; i++ {
		data := make(url.Values)
		data["token"] = []string{token.Token}
		data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
		data["text"] = []string{"Post " + strconv.Itoa(i)}
		switch {
		case i < 10:
			data["event-time"] = []string{time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339)}
		case i < 35:
			data["event-time"] = []string{time.Now().UTC().Add(25 * time.Minute).Format(time.RFC3339)}
		case i < 37:
			data["event-time"] = []string{time.Now().UTC().Add(250 * time.Minute).Format(time.RFC3339)}
		case i < 50:
			data["event-time"] = []string{time.Now().UTC().Add(2500 * time.Minute).Format(time.RFC3339)}
		default:

		}

		req, err := http.NewRequest("POST", baseURL+"posts", strings.NewReader(data.Encode()))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Close = true

		_, err = client.Do(req)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestLive(t *testing.T) {

	config := conf.GetConfig()
	api = lib.New(*config)
	api.Mail = mail.NewMock()
	api.TW = lib.StubTranscodeWorker{}
	api.Start()
	server := httptest.NewServer(r)
	baseURL = server.URL + "/api/v1/"

	err := liveInit()

	if err != nil {
		t.Fatal("Error setting up live test:", err)
	}

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatal("Error logging in:", err)
	}

	type liveTest struct {
		Token              gp.Token
		After              string
		Until              string
		ExpectedStatusCode int
		ExpectedType       string
		ExpectedCount      int
		ExpectedError      string
	}
	tests := []liveTest{
		{
			Token:              token,
			After:              time.Now().UTC().Format(time.RFC3339),
			ExpectedStatusCode: http.StatusOK,
			ExpectedType:       "[]gp.PostSmall",
			ExpectedCount:      20,
		},
		{
			Token:              token,
			After:              strconv.FormatInt(time.Now().UTC().Unix(), 10),
			ExpectedStatusCode: http.StatusOK,
			ExpectedType:       "[]gp.PostSmall",
			ExpectedCount:      20,
		},
	}
	client := &http.Client{}
	for _, test := range tests {
		data := make(url.Values)
		data["after"] = []string{test.After}
		data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
		data["token"] = []string{token.Token}
		resp, err := client.Get(baseURL + "live?" + data.Encode())
		if err != nil {
			t.Fatal("Error getting live events:", err)
		}
		if resp.StatusCode != test.ExpectedStatusCode {
			errResp := gp.APIerror{}
			dec := json.NewDecoder(resp.Body)
			dec.Decode(&errResp)
			log.Println(errResp)
			t.Fatalf("Expected status code %d but got %d\n", test.ExpectedStatusCode, resp.StatusCode)
		}
		switch {
		case test.ExpectedType == "[]gp.PostSmall":
			posts := make([]gp.PostSmall, 0)
			dec := json.NewDecoder(resp.Body)
			err = dec.Decode(&posts)
			if err != nil {
				t.Fatal("Couldn't decode as []gp.PostSmall", err)
			}
			if len(posts) != test.ExpectedCount {
				t.Fatalf("Got an unexpected number of posts - got %d but expected %d\n", len(posts), test.ExpectedCount)
			}
		case test.ExpectedType == "gp.APIerror":
			errResp := gp.APIerror{}
			dec := json.NewDecoder(resp.Body)
			err = dec.Decode(&errResp)
			if err != nil {
				t.Fatal("Couldn't decode as APIerror", err)
			}
			if errResp.Reason != test.ExpectedError {
				t.Fatalf("Expected %s but got %s\n", test.ExpectedError, errResp.Reason)
			}
		}
	}
}
