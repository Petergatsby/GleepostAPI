package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func TestCreatePost(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	client := &http.Client{}

	loginResp, err := loginRequest("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatalf("Error logging in: %v\n", err)
	}

	dec := json.NewDecoder(loginResp.Body)

	token := gp.Token{}
	err = dec.Decode(&token)
	if err != nil {
		t.Fatalf("Error getting login token: %v", err)
	}

	type createPostTest struct {
		TestNumber         int
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
		TestNumber:         1,
		Text:               "Hello my name is Patrick, how are you?",
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusCreated,
	}
	badPost := createPostTest{
		TestNumber:         2,
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Post contains no content",
	}
	badToken := createPostTest{
		TestNumber:         3,
		Text:               "Hey my name is Patrick, what up?",
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	badID := createPostTest{
		TestNumber:         4,
		Text:               "Yo yo me name's Pat, sup?",
		Token:              token.Token,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	badImage := createPostTest{
		TestNumber:         5,
		Image:              "https://www.fakeimage.com/lololol.jpg",
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "That is not a valid image",
	}
	badVideo := createPostTest{
		TestNumber:         6,
		Video:              "12341",
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "That is not a valid video",
	}
	tests := []createPostTest{textPost, badPost, badToken, badID, badImage, badVideo}
	for _, cpt := range tests {

		data := make(url.Values)
		data["token"] = []string{cpt.Token}
		data["id"] = []string{fmt.Sprintf("%d", cpt.UserID)}
		data["text"] = []string{cpt.Text}
		data["tags"] = []string{cpt.Tags}
		data["url"] = []string{cpt.Image}
		data["video"] = []string{cpt.Video}

		postResp, err := client.PostForm(baseURL+"posts", data)
		if err != nil {
			t.Fatalf("Test%v: Error with post request: %v", cpt.TestNumber, err)
		}

		if cpt.ExpectedStatusCode != postResp.StatusCode {
			errorValue := gp.APIerror{}
			dec = json.NewDecoder(postResp.Body)
			err = dec.Decode(&errorValue)
			t.Fatalf("Test%v: Expected %v, got %v: %v\n", cpt.TestNumber, cpt.ExpectedStatusCode, postResp.StatusCode, errorValue.Reason)
		} else if cpt.ExpectedStatusCode == http.StatusBadRequest {
			errorValue := gp.APIerror{}
			dec = json.NewDecoder(postResp.Body)
			err = dec.Decode(&errorValue)
			if cpt.ExpectedError != errorValue.Reason {
				t.Fatalf("Test%v: Expected %v, got %v\n", cpt.TestNumber, cpt.ExpectedError, errorValue.Reason)
			}
		}
	}

}
