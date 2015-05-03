package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
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
		data["text"] = "Post " + i
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
}

func TestLive(t *testing.T) {
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
	}

}
