package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/draaglom/GleepostAPI/lib/conf"
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

	db, err := sql.Open("mysql", conf.GetConfig().Mysql.ConnectionString())
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	_, err = db.Exec("INSERT INTO uploads (user_id, url) VALUES (?, ?)", token.UserID, "https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg")
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	file, err := os.Open("testdata/test_post1.json")
	dec = json.NewDecoder(file)
	expectedValue := gp.PostSmall{}
	err = dec.Decode(&expectedValue)
	if err != nil {
		t.Fatalf("Error parsing expected data: %v", err)
	}

	type viewPostTest struct {
		TestNumber         int
		ExpectedPost       gp.PostSmall
		UseCorrectToken    bool
		UseCorrectID       bool
		ExpectedStatusCode int
		ExpectedError      string
	}
	goodTest := viewPostTest{
		TestNumber:         0,
		ExpectedPost:       expectedValue,
		UseCorrectToken:    true,
		UseCorrectID:       true,
		ExpectedStatusCode: http.StatusOK,
	}
	badTest := viewPostTest{
		TestNumber:         1,
		UseCorrectToken:    false,
		UseCorrectID:       false,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	badToken := viewPostTest{
		TestNumber:         2,
		UseCorrectToken:    false,
		UseCorrectID:       true,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	badID := viewPostTest{
		TestNumber:         3,
		UseCorrectToken:    false,
		UseCorrectID:       true,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}

	tests := []viewPostTest{goodTest, badTest, badToken, badID}
	for _, vpt := range tests {

		var tags string

		for _, tag := range vpt.ExpectedPost.Categories {
			tags += tag.Tag + ", "
		}

		data := make(url.Values)
		data["token"] = []string{token.Token}
		data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
		data["text"] = []string{vpt.ExpectedPost.Text}
		data["url"] = vpt.ExpectedPost.Images
		data["tags"] = []string{tags}

		_, err = client.PostForm(baseURL+"posts", data)
		if err != nil {
			t.Fatalf("Test%v: Error with post request: %v", err)
		}

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
			respValue := []gp.PostSmall{}
			err = dec.Decode(&respValue)
			if err != nil {
				t.Fatalf("Error parsing response: %v\n", err)
			}
			fmt.Println(respValue[vpt.TestNumber])
			if respValue[vpt.TestNumber].By.ID != token.UserID {
				t.Fatalf("Wrong user found. Expecting: %v, Got: %v\n", token.UserID, respValue[vpt.TestNumber].By.ID)
			} else if respValue[vpt.TestNumber].Text != vpt.ExpectedPost.Text {
				t.Fatalf("Wrong value found. Expecting: %v, Got: %v\n", vpt.ExpectedPost.Text, respValue[vpt.TestNumber].Text)
			} else if len(respValue[vpt.TestNumber].Images) > 0 || len(vpt.ExpectedPost.Images) > 0 {
				if len(respValue[vpt.TestNumber].Images) != len(vpt.ExpectedPost.Images) {
					t.Fatalf("Wrong images found. Expecting: %v, Got: %v\n", vpt.ExpectedPost.Images, respValue[vpt.TestNumber].Images)
				} else {
					for index, image := range respValue[vpt.TestNumber].Images {
						if image != vpt.ExpectedPost.Images[index] {
							t.Fatalf("Wrong image found. Expecting: %v, Got: %v\n", vpt.ExpectedPost.Images, respValue[vpt.TestNumber].Images)
						}
					}
				}
			} else if len(respValue[vpt.TestNumber].Categories) > 0 || len(vpt.ExpectedPost.Categories) > 0 {
				if len(respValue[vpt.TestNumber].Categories) != len(vpt.ExpectedPost.Categories) {
					t.Fatalf("Wrong tags found. Expecting: %v, Got: %v\n", vpt.ExpectedPost.Categories, respValue[vpt.TestNumber].Categories)
				} else {
					for index, category := range respValue[vpt.TestNumber].Categories {
						if category.Tag != vpt.ExpectedPost.Categories[index].Tag {
							t.Fatalf("Wrong tag found. Expecting: %v, Got: %v\n", vpt.ExpectedPost.Categories, respValue[vpt.TestNumber].Categories)
						}
					}
				}
			}
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
