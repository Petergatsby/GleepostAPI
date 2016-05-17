package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Petergatsby/GleepostAPI/lib/gp"
)

func TestPostComment(t *testing.T) {

	once.Do(setup)

	truncate("post_comments")

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatalf("Error logging in: %v", err)
	}

	var largeText string
	for x := 0; x <= 1024; x++ {
		largeText += "l"
	}

	type postCommentTest struct {
		UserID             gp.UserID
		Token              string
		Text               string
		PostID             gp.PostID
		ExpectedStatusCode int
		ExpectedError      string
	}

	goodTest := postCommentTest{
		UserID:             token.UserID,
		Token:              token.Token,
		Text:               "Lolololol this post was so funny",
		PostID:             1,
		ExpectedStatusCode: http.StatusCreated,
	}
	emptyTest := postCommentTest{
		UserID:             token.UserID,
		Token:              token.Token,
		Text:               "",
		PostID:             1,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Comment too short",
	}
	hugeTextTest := postCommentTest{
		UserID:             token.UserID,
		Token:              token.Token,
		Text:               largeText,
		PostID:             1,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Comment too long",
	}
	noPostTest := postCommentTest{
		UserID:             token.UserID,
		Token:              token.Token,
		Text:               "This one was not so funny, it didn't exist",
		PostID:             1123,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "No such post",
	}

	tests := []postCommentTest{goodTest, emptyTest, noPostTest, hugeTextTest}

	createSimplePost(token, "Simple test post")

	for testNumber, pct := range tests {
		data := make(url.Values)
		data["token"] = []string{pct.Token}
		data["id"] = []string{fmt.Sprintf("%d", pct.UserID)}
		data["text"] = []string{pct.Text}

		req, err := http.NewRequest("POST", baseURL+"posts/"+fmt.Sprintf("%d", pct.PostID)+"/comments", strings.NewReader(data.Encode()))
		if err != nil {
			t.Fatalf("Test%v: %v", testNumber, err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Close = true

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Test%v: %v", testNumber, err)
		}

		if resp.StatusCode != pct.ExpectedStatusCode {
			t.Fatalf("Test%v: Expected %v, got %v", testNumber, pct.ExpectedStatusCode, resp.StatusCode)
		}

		dec := json.NewDecoder(resp.Body)

		if pct.ExpectedStatusCode == http.StatusCreated {
			var created gp.Created
			err = dec.Decode(&created)
			if err != nil {
				t.Fatalf("Test%v: %v", testNumber, err)
			}
		} else if pct.ExpectedStatusCode == http.StatusBadRequest {
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if pct.ExpectedError != errorValue.Reason {
				t.Fatalf("Test%v: Expected %v, got %v\n", testNumber, pct.ExpectedError, errorValue.Reason)
			}
		}
	}
}

func createSimplePost(token gp.Token, text string) error {

	data := make(url.Values)
	data["token"] = []string{token.Token}
	data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
	data["text"] = []string{text}

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

	return nil
}
