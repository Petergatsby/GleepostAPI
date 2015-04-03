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

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")

	db, err := sql.Open("mysql", conf.GetConfig().Mysql.ConnectionString())
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	_, err = db.Exec("INSERT INTO uploads (user_id, url) VALUES (?, ?)", token.UserID, "https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg")
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	type viewPostTest struct {
		TestNumber         int
		ExpectedPost       string
		Token              string
		UserID             gp.UserID
		ExpectedStatusCode int
		ExpectedError      string
	}
	goodTest := viewPostTest{
		TestNumber:         0,
		ExpectedPost:       "testdata/test_post1.json",
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusOK,
	}
	badTag := viewPostTest{
		TestNumber:         1,
		ExpectedPost:       "testdata/test_post2.json",
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusOK,
		ExpectedError:      "Tag mismatch",
	}
	badImage := viewPostTest{
		TestNumber:         2,
		ExpectedPost:       "testdata/test_post3.json",
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusOK,
		ExpectedError:      "Image mismatch",
	}
	badTest := viewPostTest{
		TestNumber:         3,
		ExpectedPost:       "testdata/test_post1.json",
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	badToken := viewPostTest{
		TestNumber:         4,
		ExpectedPost:       "testdata/test_post1.json",
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	badID := viewPostTest{
		TestNumber:         5,
		ExpectedPost:       "testdata/test_post1.json",
		Token:              token.Token,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	tests := []viewPostTest{goodTest, badTag, badImage, badTest, badToken, badID}
	for _, vpt := range tests {

		file, err := os.Open(vpt.ExpectedPost)
		dec := json.NewDecoder(file)
		expectedValue := gp.PostSmall{}
		err = dec.Decode(&expectedValue)
		if err != nil {
			t.Fatalf("Test%v: Error parsing expected data: %v", vpt.TestNumber, err)
		}

		var tags string

		for _, tag := range expectedValue.Categories {
			tags += tag.Tag + ", "
		}

		data := make(url.Values)
		data["token"] = []string{token.Token}
		data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
		data["text"] = []string{expectedValue.Text}
		data["url"] = expectedValue.Images
		data["tags"] = []string{tags}

		_, err = client.PostForm(baseURL+"posts", data)
		if err != nil {
			t.Fatalf("Test%v: Error with post request: %v", vpt.TestNumber, err)
		}

		req, err := http.NewRequest("GET", baseURL+"posts", nil)
		req.Header.Set("X-GP-Auth", fmt.Sprintf("%d", vpt.UserID)+"-"+vpt.Token)

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
				t.Fatalf("Test%v: Expected %v, got %v\n", vpt.TestNumber, vpt.ExpectedStatusCode, resp.StatusCode)
			}

			dec := json.NewDecoder(resp.Body)
			respValue := []gp.PostSmall{}
			err = dec.Decode(&respValue)
			if err != nil {
				t.Fatalf("Test%v: Error parsing response: %v\n", vpt.TestNumber, err)
			}
			fmt.Println(respValue[0])
			if respValue[0].By.ID != token.UserID {
				t.Fatalf("Test%v: Wrong user found. Expecting: %v, Got: %v\n", vpt.TestNumber, token.UserID, respValue[0].By.ID)
			}

			if respValue[0].Text != expectedValue.Text {
				t.Fatalf("Test%v: Wrong value found. Expecting: %v, Got: %v\n", vpt.TestNumber, expectedValue.Text, respValue[0].Text)
			}

			err = testPostImageMatch(respValue[0].Images, expectedValue.Images)
			if vpt.ExpectedError == "Image mismatch" {
				if err == nil || err.Error() != "Image mismatch" {
					t.Fatalf("Test%v: Expected image mismatch, but did not get error", vpt.TestNumber)
				}
			} else {
				if err != nil {
					t.Fatalf("Test%v: Error with images: %v", vpt.TestNumber, err)
				}
			}

			err = testPostTagMatch(respValue[0].Categories, expectedValue.Categories)
			if vpt.ExpectedError == "Tag mismatch" {
				if err == nil || err.Error() != "Tag mismatch" {
					t.Fatalf("Test%v: Expected tag mismatch, but did not get error", vpt.TestNumber)
				}
			} else {
				if err != nil {
					t.Fatalf("Test%v: Error with tags: %v", vpt.TestNumber, err)
				}
			}

		case vpt.ExpectedStatusCode == http.StatusBadRequest:
			if vpt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Test%v: Expected %v, got %v\n", vpt.TestNumber, vpt.ExpectedStatusCode, resp.StatusCode)
			}

			dec := json.NewDecoder(resp.Body)
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Test%v: Error parsing error: %v\n", vpt.TestNumber, err)
			}
			if errorValue.Reason != vpt.ExpectedError {
				t.Fatalf("Test%v: Expected %s, got %s\n", vpt.TestNumber, vpt.ExpectedError, errorValue.Reason)
			}
		default:
			t.Fatalf("Test%v: Something completely unexpected happened")
		}
	}
}

func testPostTagMatch(currentValue []gp.PostCategory, expectedValue []gp.PostCategory) (err error) {
	if len(currentValue) > 0 || len(expectedValue) > 0 {
		if len(currentValue) != len(expectedValue) {
			return gp.APIerror{Reason: "Tag mismatch"}
		} else {
			for index, category := range currentValue {
				if category.Tag != expectedValue[index].Tag {
					return gp.APIerror{Reason: "Tag mistmatch"}
				}
			}
		}
	}
	return
}

func testPostImageMatch(currentValue []string, expectedValue []string) (err error) {
	if len(currentValue) > 0 || len(expectedValue) > 0 {
		if len(currentValue) != len(expectedValue) {
			return gp.APIerror{Reason: "Image mismatch"}
		} else {
			for index, image := range currentValue {
				if image != expectedValue[index] {
					return gp.APIerror{Reason: "Image mistmatch"}
				}
			}
		}
	}
	return
}
