package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func TestCreatePoll(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatalf("Couldn't log in: %v\n", err)
	}

	client := &http.Client{}

	type createPollTest struct {
		Token              gp.Token
		Text               string
		Tags               []string
		PollOptions        []string
		PollExpiry         string
		ExpectedStatusCode int
		ExpectedType       string
		ExpectedError      string
	}
	testGood := createPollTest{
		Token:              token,
		Text:               "Which is the best option?",
		Tags:               []string{"poll"},
		PollOptions:        []string{"Option 1", "Another option", "Nothing"},
		PollExpiry:         time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		ExpectedStatusCode: 201,
		ExpectedType:       "CreatedPost",
	}
	testMissingExpiry := createPollTest{
		Token:              token,
		Text:               "Which is the best option?",
		Tags:               []string{"poll"},
		PollOptions:        []string{"Option 1", "Another option", "Nothing"},
		ExpectedStatusCode: 400,
		ExpectedType:       "Error",
		ExpectedError:      "Missing parameter: poll-expiry",
	}
	testExpiryPast := createPollTest{
		Token:              token,
		Text:               "Which is the best option?",
		Tags:               []string{"poll"},
		PollOptions:        []string{"Option 1", "Another option", "Nothing"},
		PollExpiry:         time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		ExpectedStatusCode: 400,
		ExpectedType:       "Error",
		ExpectedError:      "Poll ending in the past",
	}
	testTooSoon := createPollTest{
		Token:              token,
		Text:               "Which is the best option?",
		Tags:               []string{"poll"},
		PollOptions:        []string{"Option 1", "Another option", "Nothing"},
		PollExpiry:         time.Now().Add(10 * time.Second).Format(time.RFC3339),
		ExpectedStatusCode: 400,
		ExpectedType:       "Error",
		ExpectedError:      "Poll ending too soon",
	}
	testTooLate := createPollTest{
		Token:              token,
		Text:               "Which is the best option?",
		Tags:               []string{"poll"},
		PollOptions:        []string{"Option 1", "Another option", "Nothing"},
		PollExpiry:         time.Now().AddDate(1, 0, 0).Format(time.RFC3339),
		ExpectedStatusCode: 400,
		ExpectedType:       "Error",
		ExpectedError:      "Poll ending too late",
	}
	testFewOptions := createPollTest{
		Token:              token,
		Text:               "Which is the best option?",
		Tags:               []string{"poll"},
		PollOptions:        []string{"Option 1"},
		PollExpiry:         time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		ExpectedStatusCode: 400,
		ExpectedType:       "Error",
		ExpectedError:      "Poll: too few options",
	}
	testManyOptions := createPollTest{
		Token:              token,
		Text:               "Which is the best option?",
		Tags:               []string{"poll"},
		PollOptions:        []string{"Option 1", "Another option", "Lrrr", "Zaphod Beeblebrox", "Norton Juster"},
		PollExpiry:         time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		ExpectedStatusCode: 400,
		ExpectedType:       "Error",
		ExpectedError:      "Poll: too many options",
	}
	testShort := createPollTest{
		Token:              token,
		Text:               "Which is the best option?",
		Tags:               []string{"poll"},
		PollOptions:        []string{"Option 1", "A", "Lrrr", "Zaphod Beeblebrox"},
		PollExpiry:         time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		ExpectedStatusCode: 400,
		ExpectedType:       "Error",
		ExpectedError:      "Option too short: 1",
	}
	testLong := createPollTest{
		Token:              token,
		Text:               "Which is the best option?",
		Tags:               []string{"poll"},
		PollOptions:        []string{"Option 1", "A really really really really really really really really really really long option.", "Really"},
		PollExpiry:         time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		ExpectedStatusCode: 400,
		ExpectedType:       "Error",
		ExpectedError:      "Option too long: 1",
	}

	tests := []createPollTest{testGood, testMissingExpiry, testExpiryPast, testTooSoon, testTooLate, testFewOptions, testManyOptions, testShort, testLong}
	for _, cpt := range tests {
		data := make(url.Values)
		data["id"] = []string{fmt.Sprintf("%d", cpt.Token.UserID)}
		data["token"] = []string{cpt.Token.Token}
		data["text"] = []string{cpt.Text}
		data["poll-options"] = []string{strings.Join(cpt.PollOptions, ",")}
		if len(cpt.PollExpiry) > 0 {
			data["poll-expiry"] = []string{cpt.PollExpiry}
		}
		data["tags"] = cpt.Tags

		resp, err := client.PostForm(baseURL+"posts", data)
		if err != nil {
			t.Fatal("Error making request:", err)
		}
		if resp.StatusCode != cpt.ExpectedStatusCode {
			t.Fatalf("Expected status code %d, got %d\n", cpt.ExpectedStatusCode, resp.StatusCode)
		}
		dec := json.NewDecoder(resp.Body)
		switch {
		case cpt.ExpectedType == "CreatedPost":
			post := gp.CreatedPost{}
			err = dec.Decode(&post)
			if err != nil {
				t.Fatalf("Failed to decode as %s: %v\n", cpt.ExpectedType, err)
			}
			if post.ID < 1 {
				t.Fatalf("Post.ID must be nonzero (%d)\n", post.ID)
			}
			if post.Pending == true {
				t.Fatalf("Post should not be pending")
			}
		case cpt.ExpectedType == "Error":
			errorResp := gp.APIerror{}
			err = dec.Decode(&errorResp)
			if err != nil {
				t.Fatalf("Failed to decode as %s: %v\n", cpt.ExpectedType, err)
			}
			if cpt.ExpectedError != errorResp.Reason {
				t.Fatalf("Expected error: %s but got: %s\n", cpt.ExpectedError, errorResp.Reason)
			}
		}
	}
}
