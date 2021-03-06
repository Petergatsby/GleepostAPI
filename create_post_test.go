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

func TestCreatePost(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	once.Do(setup)

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatalf("Error logging in: %v\n", err)
	}

	type createPostTest struct {
		Title              string
		Text               string
		Tags               string
		Image              string
		Video              string
		Token              string
		UserID             gp.UserID
		ExpectedStatusCode int
		ExpectedError      string
	}
	textPost := createPostTest{
		Text:               "Hello my name is Patrick, how are you?",
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusCreated,
	}
	badPost := createPostTest{
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Post contains no content",
	}
	badToken := createPostTest{
		Text:               "Hey my name is Patrick, what up?",
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusUnauthorized,
		ExpectedError:      "Invalid credentials",
	}
	badID := createPostTest{
		Text:               "Yo yo me name's Pat, sup?",
		Token:              token.Token,
		ExpectedStatusCode: http.StatusUnauthorized,
		ExpectedError:      "Invalid credentials",
	}
	badImage := createPostTest{
		Image:              "https://www.fakeimage.com/lololol.jpg",
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "That is not a valid image",
	}
	badVideo := createPostTest{
		Video:              "12341",
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "That is not a valid video",
	}
	tests := []createPostTest{textPost, badPost, badToken, badID, badImage, badVideo}
	for testNumber, cpt := range tests {

		data := make(url.Values)
		data["token"] = []string{cpt.Token}
		data["id"] = []string{fmt.Sprintf("%d", cpt.UserID)}
		data["text"] = []string{cpt.Text}
		data["tags"] = []string{cpt.Tags}
		data["url"] = []string{cpt.Image}
		data["video"] = []string{cpt.Video}

		req, _ := http.NewRequest("POST", baseURL+"posts", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Close = true

		postResp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Test%v: Error with post request: %v", testNumber, err)
		}

		if cpt.ExpectedStatusCode != postResp.StatusCode {
			errorValue := gp.APIerror{}
			dec := json.NewDecoder(postResp.Body)
			err = dec.Decode(&errorValue)
			t.Fatalf("Test%v: Expected %v, got %v: %v\n", testNumber, cpt.ExpectedStatusCode, postResp.StatusCode, errorValue.Reason)
		} else if cpt.ExpectedStatusCode == http.StatusBadRequest {
			errorValue := gp.APIerror{}
			dec := json.NewDecoder(postResp.Body)
			err = dec.Decode(&errorValue)
			if cpt.ExpectedError != errorValue.Reason {
				t.Fatalf("Test%v: Expected %v, got %v\n", testNumber, cpt.ExpectedError, errorValue.Reason)
			}
		}
	}
}
