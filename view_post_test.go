package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

var testStart = time.Now()

type viewPostTest struct {
	ExpectedPostIndex  int
	VideoID            string
	Token              string
	UserID             gp.UserID
	ExpectedStatusCode int
	ExpectedError      string
}

func TestViewPost(t *testing.T) {
	once.Do(setup)

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")

	goodTest := viewPostTest{
		ExpectedPostIndex:  0,
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusOK,
	}
	goodTestVideo := viewPostTest{
		ExpectedPostIndex:  1,
		VideoID:            "9989",
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusOK,
	}
	badTest := viewPostTest{
		ExpectedPostIndex:  0,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	badToken := viewPostTest{
		ExpectedPostIndex:  0,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	badID := viewPostTest{
		ExpectedPostIndex:  0,
		Token:              token.Token,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	tests := []viewPostTest{goodTest, goodTestVideo, badTest, badToken, badID}

	err = initPosts(tests)
	if err != nil {
		t.Fatalf("Error initialising posts: %v", err)
	}

	file, err := os.Open("testdata/test_post1.json")
	if err != nil {
		t.Fatalf("Error loading test file: %v", err)
	}
	dec := json.NewDecoder(file)
	expectedValues := []gp.PostSmall{}
	err = dec.Decode(&expectedValues)
	if err != nil {
		t.Fatalf("Error parsing expected data: %v", err)
	}

	for testNumber, vpt := range tests {

		req, err := http.NewRequest("GET", baseURL+"posts", nil)
		req.Header.Set("X-GP-Auth", fmt.Sprintf("%d", vpt.UserID)+"-"+vpt.Token)

		if err != nil {
			t.Fatalf("Test%v: Error with get request: %v", testNumber, err)
		}

		resp, err := client.Do(req)

		if err != nil {
			t.Fatalf("Test%v: Error with get post: %v", testNumber, err)
		}

		if vpt.ExpectedStatusCode != resp.StatusCode {
			t.Fatalf("Test%v: Expected %v, got %v\n", testNumber, vpt.ExpectedStatusCode, resp.StatusCode)
		}

		switch {
		case vpt.ExpectedStatusCode == http.StatusOK:

			expectedValue := expectedValues[vpt.ExpectedPostIndex]

			addTimeDuration, _ := time.ParseDuration(fmt.Sprintf("%d", expectedValue.Attribs["event-time"]))
			expectedValue.Attribs["event-time"] = fmt.Sprintf("%d", testStart.Add(addTimeDuration).Unix())

			dec = json.NewDecoder(resp.Body)
			respValue := []gp.PostSmall{}
			err = dec.Decode(&respValue)
			if err != nil {
				t.Fatalf("Test%v: Error parsing response: %v\n", testNumber, err)
			}

			respNumber := len(tests) - testNumber - 1

			verifyPost(respValue[respNumber], expectedValue)

			if err != nil {
				t.Fatalf("Test%v: %v", testNumber, err)
			}

		case vpt.ExpectedStatusCode == http.StatusBadRequest:
			dec := json.NewDecoder(resp.Body)
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Test%v: Error parsing error: %v\n", testNumber, err)
			}
			if errorValue.Reason != vpt.ExpectedError {
				t.Fatalf("Test%v: Expected %s, got %s\n", testNumber, vpt.ExpectedError, errorValue.Reason)
			}
		default:
			t.Fatalf("Test%v: Something completely unexpected happened", testNumber)
		}
	}
}

func verifyPost(currentPost gp.PostSmall, expectedPost gp.PostSmall) error {
	if currentPost.By.ID != expectedPost.By.ID {
		return gp.APIerror{Reason: fmt.Sprintf("Wrong user found. Expecting: %v, Got: %v\n", expectedPost.By.ID, currentPost.By.ID)}
	}

	if currentPost.Attribs["title"] != expectedPost.Attribs["title"] {
		return gp.APIerror{Reason: fmt.Sprintf("Wrong title found. Expecting: %v, Got: %v\n", expectedPost.Attribs["title"], currentPost.Attribs["title"])}
	}

	if currentPost.Attribs["location-desc"] != expectedPost.Attribs["location-desc"] {
		return gp.APIerror{Reason: fmt.Sprintf("Wrong location-desc found. Expecting: %v, Got: %v\n", expectedPost.Attribs["location-desc"], currentPost.Attribs["location-desc"])}
	}

	if currentPost.Attribs["location-name"] != expectedPost.Attribs["location-name"] {
		return gp.APIerror{Reason: fmt.Sprintf("Wrong location-name found. Expecting: %v, Got: %v\n", expectedPost.Attribs["location-name"], currentPost.Attribs["location-name"])}
	}

	if currentPost.Attribs["location-gps"] != expectedPost.Attribs["location-gps"] {
		return gp.APIerror{Reason: fmt.Sprintf("Wrong location-gps found. Expecting: %v, Got: %v\n", expectedPost.Attribs["location-gps"], currentPost.Attribs["location-gps"])}
	}

	if currentPost.Text != expectedPost.Text {
		return gp.APIerror{Reason: fmt.Sprintf("Wrong value found. Expecting: %v, Got: %v\n", expectedPost.Text, currentPost.Text)}
	}

	err := testPostImageMatch(currentPost.Images, expectedPost.Images)
	if err != nil {
		return gp.APIerror{Reason: fmt.Sprintf("Error with images: %v", err)}
	}

	err = testPostVideoMatch(currentPost.Videos, expectedPost.Videos)

	if err != nil {
		return gp.APIerror{Reason: fmt.Sprintf("Error with videos: %v", err)}
	}

	err = testPostTagMatch(currentPost.Categories, expectedPost.Categories)
	if err != nil {
		return gp.APIerror{Reason: fmt.Sprintf("Error with tags: %v", err)}
	}

	responseTime, err := time.Parse(time.RFC3339, currentPost.Attribs["event-time"].(string))
	if err != nil {
		return gp.APIerror{Reason: fmt.Sprintf("Error with time: %v", err)}
	}
	if fmt.Sprintf("%d", responseTime.Unix()) != expectedPost.Attribs["event-time"] {
		return gp.APIerror{Reason: fmt.Sprintf("Wrong time found. Expecting: %v, Got: %v\n", expectedPost.Attribs["event-time"], responseTime.Unix())}
	}

	return nil
}

func initPosts(tests []viewPostTest) error {

	err := initDB()
	if err != nil {
		return err
	}

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", conf.GetConfig().Mysql.ConnectionString())
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO uploads (user_id, url) VALUES (?, ?)", token.UserID, "https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg")
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO uploads (user_id, url, mp4_url, webm_url, upload_id, status) VALUES (?, ?, ?, ?, ?, ?)", token.UserID, "https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg", "https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4", "https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm", 9989, "ready")
	if err != nil {
		return err
	}

	file, err := os.Open("testdata/test_post1.json")
	if err != nil {
		return err
	}
	dec := json.NewDecoder(file)
	expectedValues := []gp.PostSmall{}
	err = dec.Decode(&expectedValues)
	if err != nil {
		return err
	}

	for _, vpt := range tests {

		expectedValue := expectedValues[vpt.ExpectedPostIndex]

		var tags string

		for _, tag := range expectedValue.Categories {
			tags += tag.Tag + ", "
		}

		addTimeDuration, _ := time.ParseDuration(fmt.Sprintf("%d", expectedValue.Attribs["event-time"]))
		expectedValue.Attribs["event-time"] = fmt.Sprintf("%d", testStart.Add(addTimeDuration).Unix())

		data := make(url.Values)
		data["token"] = []string{token.Token}
		data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
		data["text"] = []string{expectedValue.Text}
		data["url"] = expectedValue.Images
		data["tags"] = []string{tags}
		data["video"] = []string{vpt.VideoID}
		data["event-time"] = []string{expectedValue.Attribs["event-time"].(string)}
		data["title"] = []string{expectedValue.Attribs["title"].(string)}
		data["location-desc"] = []string{expectedValue.Attribs["location-desc"].(string)}
		data["location-name"] = []string{expectedValue.Attribs["location-name"].(string)}
		data["location-gps"] = []string{expectedValue.Attribs["location-gps"].(string)}

		_, err = client.PostForm(baseURL+"posts", data)
		if err != nil {
			return err
		}
	}
	return nil
}

func testPostTagMatch(currentValue []gp.PostCategory, expectedValue []gp.PostCategory) (err error) {
	if len(currentValue) > 0 || len(expectedValue) > 0 {
		if len(currentValue) != len(expectedValue) {
			return gp.APIerror{Reason: "Tag mismatch"}
		}
		for index, category := range currentValue {
			if category.Tag != expectedValue[index].Tag {
				return gp.APIerror{Reason: "Tag mistmatch"}
			}
		}
	}
	return
}

func testPostImageMatch(currentValue []string, expectedValue []string) (err error) {
	if len(currentValue) > 0 || len(expectedValue) > 0 {
		if len(currentValue) != len(expectedValue) {
			return gp.APIerror{Reason: "Image mismatch"}
		}
		for index, image := range currentValue {
			if image != expectedValue[index] {
				return gp.APIerror{Reason: "Image mistmatch"}
			}
		}
	}
	return
}

func testPostVideoMatch(currentValue []gp.Video, expectedValue []gp.Video) (err error) {
	if len(currentValue) > 0 || len(expectedValue) > 0 {
		if len(currentValue) != len(expectedValue) {
			return gp.APIerror{Reason: "Video mismatch"}
		}
		for index, video := range currentValue {
			switch {
			case video.ID != expectedValue[index].ID:
				return gp.APIerror{Reason: "Video mismatch"}
			case video.MP4 != expectedValue[index].MP4:
				return gp.APIerror{Reason: "Video mismatch"}
			case video.WebM != expectedValue[index].WebM:
				return gp.APIerror{Reason: "Video mismatch"}
			case len(video.Thumbs) != len(expectedValue[index].Thumbs):
				return gp.APIerror{Reason: "Video mismatch"}
			case len(video.Thumbs) == len(expectedValue[index].Thumbs):
				for thumbIndex, thumbnail := range video.Thumbs {
					if thumbnail != expectedValue[index].Thumbs[thumbIndex] {
						return gp.APIerror{Reason: "Video mismatch"}
					}
				}
			}
		}
	}
	return
}
