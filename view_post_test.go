package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func TestViewPost(t *testing.T) {
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

	data := make(url.Values)
	data["token"] = []string{token.Token}
	data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
	data["title"] = []string{"Test Post"}
	data["text"] = []string{"This post is for tests"}

	_, err = client.PostForm(baseURL+"posts", data)
	if err != nil {
		t.Fatalf("Test%v: Error with post request: %v", err)
	}

	type viewPostTest struct {
		TestNumber         int
		UseCorrectToken    bool
		UseCorrectID       bool
		ExpectedStatusCode int
		ExpectedError      string
	}
	goodTest := viewPostTest{
		TestNumber:         1,
		UseCorrectToken:    true,
		UseCorrectID:       true,
		ExpectedStatusCode: http.StatusOK,
	}
	badTest := viewPostTest{
		TestNumber:         2,
		UseCorrectToken:    false,
		UseCorrectID:       false,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	badToken := viewPostTest{
		TestNumber:         3,
		UseCorrectToken:    false,
		UseCorrectID:       true,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	badID := viewPostTest{
		TestNumber:         4,
		UseCorrectToken:    false,
		UseCorrectID:       true,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}

	tests := []viewPostTest{goodTest, badTest, badToken, badID}
	for _, vpt := range tests {

		var userToken string
		var userID string

		if vpt.UseCorrectToken {
			userToken = token.Token
		} else {
			userToken = "aasdfjan02t11adv0a9va9v2"
		}

		if vpt.UseCorrectID {
			userID = fmt.Sprintf("%d", token.UserID)
		} else {
			userID = "914783"
		}

		req, err := http.NewRequest("GET", baseURL+"posts", nil)
		req.Header.Set("X-GP-Auth", userID+"-"+userToken)

		if err != nil {
			t.Fatalf("Test%v: Error with get request: %v", vpt.TestNumber, err)
		}

		resp, err := client.Do(req)

		if err != nil {
			t.Fatalf("Test%v: Error with get post: %v", vpt.TestNumber, err)
		}

		switch {
		case vpt.ExpectedStatusCode == http.StatusOK:
			if vpt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Expected %v, got %v\n", vpt.ExpectedStatusCode, resp.StatusCode)
			}
			dec := json.NewDecoder(resp.Body)
			respValue := gp.PostSmall{}
			err = dec.Decode(&respValue)
		case vpt.ExpectedStatusCode == http.StatusBadRequest:
			if vpt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Expected %v, got %v\n", vpt.ExpectedStatusCode, resp.StatusCode)
			}

			dec := json.NewDecoder(resp.Body)
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Error parsing error: %v\n", err)
			}
			if errorValue.Reason != vpt.ExpectedError {
				t.Fatalf("Expected %s, got %s\n", vpt.ExpectedError, errorValue.Reason)
			}
		default:
			t.Fatalf("Something completely unexpected happened")
		}
	}
}
