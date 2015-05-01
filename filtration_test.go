package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func TestFiltration(t *testing.T) {
	client := &http.Client{}

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatalf("Error logging in", err)
	}

	type filtrationTest struct {
		Filter          string
		ExpectedMatches int
	}

	newsTest := filtrationTest{
		Filter:          "news",
		ExpectedMatches: 3,
	}
	jobsTest := filtrationTest{
		Filter:          "jobs",
		ExpectedMatches: 2,
	}
	badTest := filtrationTest{
		Filter:          "lies",
		ExpectedMatches: 0,
	}
	tests := []filtrationTest{newsTest, jobsTest, badTest}

	truncate("wall_posts")
	truncate("post_categories")
	initPostFromJSON("testdata/filtration_test_posts.json")

	for testNumber, ft := range tests {
		req, err := http.NewRequest("GET", baseURL+"posts?filter="+ft.Filter, nil)
		req.Header.Set("X-GP-Auth", fmt.Sprintf("%d", token.UserID)+"-"+token.Token)
		req.Close = true

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
		if err != nil {
			t.Fatalf("Error parsing expected data: %v", err)
		}

		if len(respValue) != ft.ExpectedMatches {
			t.Fatalf("Test%v: Did not return the correct number of posts. Expected %v posts, got %v posts", testNumber, ft.ExpectedMatches, len(respValue))
		}

		categoryMatches := 0
		for _, post := range respValue {
			for _, category := range post.Categories {
				if category.Tag == ft.Filter {
					categoryMatches++
				}
			}
		}
		if categoryMatches != ft.ExpectedMatches {
			t.Fatalf("Test%v: Did not correctly filter posts. Expected %v matches, got %v matches", testNumber, ft.ExpectedMatches, categoryMatches)
		}
	}
}

func initPostFromJSON(fileLocation string) error {

	err := initDB()
	if err != nil {
		return err
	}

	client := &http.Client{}

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		return err
	}

	file, err := os.Open(fileLocation)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(file)
	expectedValues := []gp.PostSmall{}
	err = dec.Decode(&expectedValues)
	if err != nil {
		return err
	}

	for _, expectedValue := range expectedValues {

		var tags string

		for _, tag := range expectedValue.Categories {
			tags += tag.Tag + ", "
		}

		data := make(url.Values)
		data["token"] = []string{token.Token}
		data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
		data["text"] = []string{expectedValue.Text}
		data["tags"] = []string{tags}
		data["title"] = []string{expectedValue.Attribs["title"].(string)}

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
