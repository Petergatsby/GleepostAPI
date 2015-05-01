package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/mail"
)

func TestPostPagination(t *testing.T) {

	client := &http.Client{}

	config := conf.GetConfig()
	api = lib.New(*config)
	api.Mail = mail.NewMock()
	api.Start()
	server := httptest.NewServer(r)
	baseURL = server.URL + "/api/v1/"

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatalf("Error logging in: %v", err)
	}

	err = initManyPosts(100)
	if err != nil {
		t.Fatalf("Error intialising posts: %v", err)
	}

	type postPaginationTest struct {
		Command                   string
		ExpectedPosts             int
		ExpectedStartingPostIndex int
		ExpectedEndingPostIndex   int
	}
	beforeTest := postPaginationTest{
		Command:                   "?before=50",
		ExpectedPosts:             20,
		ExpectedStartingPostIndex: 49,
		ExpectedEndingPostIndex:   30,
	}
	afterTest := postPaginationTest{
		Command:                   "?after=25",
		ExpectedPosts:             20,
		ExpectedStartingPostIndex: 45,
		ExpectedEndingPostIndex:   26,
	}
	startTest := postPaginationTest{
		Command:                   "?start=25",
		ExpectedPosts:             20,
		ExpectedStartingPostIndex: 75,
		ExpectedEndingPostIndex:   56,
	}
	beforeEmptyTest := postPaginationTest{
		Command:                   "?before=1",
		ExpectedPosts:             0,
		ExpectedStartingPostIndex: 0,
		ExpectedEndingPostIndex:   0,
	}
	startEmptyTest := postPaginationTest{
		Command:                   "?start=150",
		ExpectedPosts:             0,
		ExpectedStartingPostIndex: 0,
		ExpectedEndingPostIndex:   0,
	}
	afterEmptyTest := postPaginationTest{
		Command:                   "?after=100",
		ExpectedPosts:             0,
		ExpectedStartingPostIndex: 0,
		ExpectedEndingPostIndex:   0,
	}
	tests := []postPaginationTest{beforeTest, afterTest, startTest, beforeEmptyTest, afterEmptyTest, startEmptyTest}
	for testNumber, ppt := range tests {
		req, err := http.NewRequest("GET", baseURL+"posts"+ppt.Command, nil)
		req.Header.Set("X-GP-Auth", fmt.Sprintf("%d", token.UserID)+"-"+token.Token)
		if err != nil {
			t.Fatalf("Test%v: Error with get request: %v", testNumber, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Test%v: Error with get post: %v", testNumber, err)
		}
		dec := json.NewDecoder(resp.Body)
		respValue := []gp.PostSmall{}
		err = dec.Decode(&respValue)

		if len(respValue) != ppt.ExpectedPosts {
			t.Fatalf("Test%v: Incorrect number of responses, expected %v, got %v", testNumber, ppt.ExpectedPosts, len(respValue))
		}

		if len(respValue) > 0 {
			if !checkPostsDescending(respValue) {
				t.Fatalf("Test%v: Posts are not in order", testNumber)
			}

			if respValue[0].ID != gp.PostID(ppt.ExpectedStartingPostIndex) {
				t.Fatalf("Test%v: Incorrect starting post number. Expected %v, got %v", testNumber, ppt.ExpectedStartingPostIndex, respValue[0].ID)
			}

			if respValue[ppt.ExpectedPosts-1].ID != gp.PostID(ppt.ExpectedEndingPostIndex) {
				t.Fatalf("Test%v: Incorrect ending post number. Expected %v, got %v", testNumber, ppt.ExpectedEndingPostIndex, respValue[ppt.ExpectedPosts-1].ID)
			}
		}
	}
}

func checkPostsDescending(posts []gp.PostSmall) bool {
	lastID := posts[0].ID + 1
	for _, post := range posts {
		if lastID != post.ID+1 {
			return false
		}
		lastID = post.ID
	}
	return true
}

func initManyPosts(tests int) error {
	err := initDB()
	if err != nil {
		return err
	}

	truncate("wall_posts")

	client := &http.Client{}

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		return err
	}

	for i := 0; i < tests; i++ {
		data := make(url.Values)
		data["token"] = []string{token.Token}
		data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
		data["text"] = []string{fmt.Sprintf("This is the test post %v", i)}
		data["title"] = []string{fmt.Sprintf("Test post %v", i)}

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
	}

	return nil
}
