package main

import (
	"database/sql"
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

func TestCreatePoll(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	config := conf.GetConfig()
	api = lib.New(*config)
	api.Mail = mail.NewMock()
	api.Start()
	server := httptest.NewServer(r)
	baseURL = server.URL + "/api/v1/"

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatalf("Couldn't log in: %v\n", err)
	}

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
		PollOptions:        []string{"Option 1", "", "Lrrr", "Zaphod Beeblebrox"},
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
	testUnix := createPollTest{
		Token:              token,
		Text:               "This poll was created with a UNIX timestamp",
		Tags:               []string{"poll"},
		PollOptions:        []string{"Why do I care?", "woo"},
		PollExpiry:         strconv.FormatInt(time.Now().Add(24*time.Hour).Unix(), 10),
		ExpectedStatusCode: 201,
		ExpectedType:       "CreatedPost",
	}

	tests := []createPollTest{testGood, testMissingExpiry, testExpiryPast, testTooSoon, testTooLate, testFewOptions, testManyOptions, testShort, testLong, testUnix}
	for testNumber, cpt := range tests {
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
			t.Fatalf("Test%v: Error making request: %s\n", testNumber, err)
		}
		if resp.StatusCode != cpt.ExpectedStatusCode {
			d := json.NewDecoder(resp.Body)
			errResp := gp.APIerror{}
			d.Decode(&errResp)
			log.Println(errResp.Reason)
			t.Fatalf("Test%v: Expected status code %d, got %d\n", testNumber, cpt.ExpectedStatusCode, resp.StatusCode)
		}
		dec := json.NewDecoder(resp.Body)
		switch {
		case cpt.ExpectedType == "CreatedPost":
			post := gp.CreatedPost{}
			err = dec.Decode(&post)
			if err != nil {
				t.Fatalf("Test%v: Failed to decode as %s: %v\n", testNumber, cpt.ExpectedType, err)
			}
			if post.ID < 1 {
				t.Fatalf("Test%v: Post.ID must be nonzero (%d)\n", testNumber, post.ID)
			}
			if post.Pending == true {
				t.Fatalf("Test%v: Post should not be pending", testNumber)
			}
		case cpt.ExpectedType == "Error":
			errorResp := gp.APIerror{}
			err = dec.Decode(&errorResp)
			if err != nil {
				t.Fatalf("Test%v: Failed to decode as %s: %v\n", testNumber, cpt.ExpectedType, err)
			}
			if cpt.ExpectedError != errorResp.Reason {
				t.Fatalf("Test%v: Expected error: %s but got: %s\n", testNumber, cpt.ExpectedError, errorResp.Reason)
			}
		}
	}
}

func TestVote(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	config := conf.GetConfig()
	api = lib.New(*config)
	api.Mail = mail.NewMock()
	api.Start()
	server := httptest.NewServer(r)
	baseURL = server.URL + "/api/v1/"

	err = initPolls()
	if err != nil {
		t.Fatalf("Error clearing poll tables: %v\n", err)
	}
	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatalf("Couldn't log in: %v\n", err)
	}

	type voteTest struct {
		Token              gp.Token
		Post               gp.PostID
		Option             string
		ExpectedStatusCode int
		ExpectedType       string
		ExpectedError      string
	}
	testNoSuchPost := voteTest{
		Token:              token,
		Option:             strconv.Itoa(1),
		Post:               12341234214123423421,
		ExpectedStatusCode: 403,
		ExpectedType:       "Error",
		ExpectedError:      "You're not allowed to do that!",
	}
	id, err := createPoll(token.UserID, "Test Poll", 1, time.Now().UTC().Add(24*time.Hour), []string{"Option A", "Option 2", "Option Gamma"})
	if err != nil {
		t.Fatalf("Error creating test poll: %v\n", err)
	}
	testVote := voteTest{
		Token:              token,
		Option:             strconv.Itoa(1),
		Post:               gp.PostID(id),
		ExpectedStatusCode: 204,
	}
	expired, err := createPoll(token.UserID, "Another Poll", 1, time.Now().UTC().Add(-1*time.Hour), []string{"Option Foo", "Option Bar", "Option Baz"})
	if err != nil {
		t.Fatalf("Error creating test poll: %v\n", err)
	}
	testExpired := voteTest{
		Token:              token,
		Option:             strconv.Itoa(2),
		Post:               gp.PostID(expired),
		ExpectedStatusCode: 400,
		ExpectedType:       "Error",
		ExpectedError:      "Poll has already ended",
	}
	testVoteTwice := voteTest{
		Token:              token,
		Option:             strconv.Itoa(0),
		Post:               gp.PostID(id),
		ExpectedStatusCode: 400,
		ExpectedType:       "Error",
		ExpectedError:      "You already voted",
	}
	another, err := createPoll(token.UserID, "Third poll", 1, time.Now().UTC().Add(24*time.Hour), []string{"Alien Kang", "Alien Kodos", "Richard Nixon's Head"})
	if err != nil {
		t.Fatalf("Error creating test poll: %v\n", err)
	}
	testBadOption := voteTest{
		Token:              token,
		Option:             strconv.Itoa(3),
		Post:               gp.PostID(another),
		ExpectedStatusCode: 400,
		ExpectedType:       "Error",
		ExpectedError:      "Invalid option",
	}
	tests := []voteTest{testNoSuchPost, testVote, testExpired, testVoteTwice, testBadOption}
	for _, vt := range tests {
		data := make(url.Values)
		data["id"] = []string{fmt.Sprintf("%d", vt.Token.UserID)}
		data["token"] = []string{vt.Token.Token}
		data["option"] = []string{vt.Option}

		resp, err := client.PostForm(fmt.Sprintf("%sposts/%d/votes", baseURL, vt.Post), data)
		if err != nil {
			t.Fatal("Error making request:", err)
		}
		if resp.StatusCode != vt.ExpectedStatusCode {
			d := json.NewDecoder(resp.Body)
			errResp := gp.APIerror{}
			d.Decode(&errResp)
			log.Println(errResp.Reason)
			t.Fatalf("Expected status code %d, got %d\n", vt.ExpectedStatusCode, resp.StatusCode)
		}
		switch {
		case vt.ExpectedStatusCode == http.StatusNoContent:
		case vt.ExpectedType == "Error":
			dec := json.NewDecoder(resp.Body)
			errResp := gp.APIerror{}
			err = dec.Decode(&errResp)
			if err != nil {
				t.Fatalf("Couldn't parse error: %v\n", err)
			}
			if errResp.Reason != vt.ExpectedError {
				t.Fatalf("Expected error %s, but got %s\n", vt.ExpectedError, errResp.Reason)
			}
		}
	}
}

func initPolls() (err error) {
	config := conf.GetConfig()
	db, err := sql.Open("mysql", config.Mysql.ConnectionString())
	if err != nil {
		return
	}
	_, err = db.Exec("TRUNCATE TABLE `wall_posts`")
	if err != nil {
		return
	}
	_, err = db.Exec("TRUNCATE TABLE `post_categories`")
	if err != nil {
		return
	}
	_, err = db.Exec("TRUNCATE TABLE `post_attribs`")
	if err != nil {
		return
	}
	_, err = db.Exec("TRUNCATE TABLE `post_polls`")
	if err != nil {
		return
	}
	_, err = db.Exec("TRUNCATE TABLE `poll_options`")
	if err != nil {
		return
	}
	_, err = db.Exec("TRUNCATE TABLE `poll_votes`")
	if err != nil {
		return
	}
	return err
}

func createPoll(user gp.UserID, text string, netID gp.NetworkID, expiry time.Time, options []string) (id int64, err error) {
	config := conf.GetConfig()
	db, err := sql.Open("mysql", config.Mysql.ConnectionString())
	if err != nil {
		return
	}
	res, err := db.Exec("INSERT INTO wall_posts (`by`, text, network_id) VALUES(?, ?, ?)", user, text, netID)
	if err != nil {
		return
	}
	id, err = res.LastInsertId()
	if err != nil {
		return
	}
	_, err = db.Exec("INSERT INTO post_categories(post_id, category_id) VALUES(?, 15)", id)
	if err != nil {
		return
	}
	_, err = db.Exec("INSERT INTO post_polls(post_id, expiry_time) VALUES (?, ?)", id, expiry)
	if err != nil {
		return
	}
	s, err := db.Prepare("INSERT INTO poll_options(post_id, option_id, `option`) VALUES(?, ?, ?)")
	if err != nil {
		return
	}
	for i, o := range options {
		_, err = s.Exec(id, i, o)
		if err != nil {
			return
		}
	}
	return
}
