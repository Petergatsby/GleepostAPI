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
	ExpectedPost       string
	VideoID            string
	Token              string
	UserID             gp.UserID
	ExpectedStatusCode int
	ExpectedError      string
}

func TestViewPost(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	client := &http.Client{}

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")

	goodTest := viewPostTest{
		ExpectedPost:       "testdata/test_post1.json",
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusOK,
	}
	goodTestVideo := viewPostTest{
		ExpectedPost:       "testdata/test_post2.json",
		VideoID:            "9989",
		Token:              token.Token,
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusOK,
	}
	badTest := viewPostTest{
		ExpectedPost:       "testdata/test_post1.json",
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	badToken := viewPostTest{
		ExpectedPost:       "testdata/test_post1.json",
		UserID:             token.UserID,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	badID := viewPostTest{
		ExpectedPost:       "testdata/test_post1.json",
		Token:              token.Token,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Invalid credentials",
	}
	tests := []viewPostTest{goodTest, goodTestVideo, badTest, badToken, badID}

	initPosts(tests)

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
			file, err := os.Open(vpt.ExpectedPost)
			dec := json.NewDecoder(file)
			expectedValue := gp.PostSmall{}
			err = dec.Decode(&expectedValue)
			if err != nil {
				t.Fatalf("Test%v: Error parsing expected data: %v", testNumber, err)
			}

			addTimeDuration, _ := time.ParseDuration(fmt.Sprintf("%d", expectedValue.Attribs["event-time"]))
			expectedValue.Attribs["event-time"] = fmt.Sprintf("%d", testStart.Add(addTimeDuration).Unix())

			dec = json.NewDecoder(resp.Body)
			respValue := []gp.PostSmall{}
			err = dec.Decode(&respValue)
			if err != nil {
				t.Fatalf("Test%v: Error parsing response: %v\n", testNumber, err)
			}

			respNumber := len(tests) - testNumber - 1

			if respValue[respNumber].By.ID != token.UserID {
				t.Fatalf("Test%v: Wrong user found. Expecting: %v, Got: %v\n", testNumber, token.UserID, respValue[respNumber].By.ID)
			}

			if respValue[respNumber].Attribs["title"] != expectedValue.Attribs["title"] {
				t.Fatalf("Test%v: Wrong title found. Expecting: %v, Got: %v\n", testNumber, expectedValue.Attribs["title"], respValue[respNumber].Attribs["title"])
			}

			if respValue[respNumber].Text != expectedValue.Text {
				t.Fatalf("Test%v: Wrong value found. Expecting: %v, Got: %v\n", testNumber, expectedValue.Text, respValue[respNumber].Text)
			}

			err = testPostImageMatch(respValue[respNumber].Images, expectedValue.Images)
			if err != nil {
				t.Fatalf("Test%v: Error with images: %v", testNumber, err)
			}

			err = testPostVideoMatch(respValue[respNumber].Videos, expectedValue.Videos)

			if err != nil {
				t.Fatalf("Test%v: Error with videos: %v", testNumber, err)
			}

			err = testPostTagMatch(respValue[respNumber].Categories, expectedValue.Categories)
			if err != nil {
				t.Fatalf("Test%v: Error with tags: %v", testNumber, err)
			}

			responseTime, err := time.Parse(time.RFC3339, respValue[respNumber].Attribs["event-time"].(string))
			if err != nil {
				t.Fatalf("Test%v: Error with time: %v", testNumber, err)
			}
			if fmt.Sprintf("%d", responseTime.Unix()) != expectedValue.Attribs["event-time"] {
				t.Fatalf("Test%v: Wrong time found. Expecting: %v, Got: %v\n", testNumber, expectedValue.Attribs["event-time"], responseTime.Unix())
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
			t.Fatalf("Test%v: Something completely unexpected happened")
		}
	}
}

func initPosts(tests []viewPostTest) error {
	client := &http.Client{}

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

	for _, vpt := range tests {
		file, err := os.Open(vpt.ExpectedPost)
		dec := json.NewDecoder(file)
		expectedValue := gp.PostSmall{}
		err = dec.Decode(&expectedValue)
		if err != nil {
			return err
		}

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
			fmt.Println("current %v, expected %v", len(currentValue), len(expectedValue))
			return gp.APIerror{Reason: "Video mismatch"}
		}
		for index, video := range currentValue {
			if video.ID != expectedValue[index].ID {
				return gp.APIerror{Reason: "Video mismatch"}
			} else if video.MP4 != expectedValue[index].MP4 {
				return gp.APIerror{Reason: "Video mismatch"}
			} else if video.WebM != expectedValue[index].WebM {
				return gp.APIerror{Reason: "Video mismatch"}
			} else if len(video.Thumbs) != len(expectedValue[index].Thumbs) {
				return gp.APIerror{Reason: "Video mismatch"}
			} else if len(video.Thumbs) == len(expectedValue[index].Thumbs) {
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
