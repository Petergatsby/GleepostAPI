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

type groupListTest struct {
	token      gp.Token
	start      int
	groupCount int
}

func TestGroupList(t *testing.T) {
	once.Do(setup)
	truncate("networks", "user_network")
	initDB()

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatal("Error logging in", err)
	}
	initNGroups(token, 25)
	tests := []groupListTest{{token: token, start: 0, groupCount: 20}, {token: token, start: 15, groupCount: 10}}

	for i, test := range tests {
		req, err := http.NewRequest("GET", fmt.Sprintf("%snetworks?start=%d", baseURL, test.start), nil)
		req.Header.Set("X-GP-Auth", fmt.Sprintf("%d-%s", test.token.UserID, test.token.Token))
		if err != nil {
			t.Fatalf("Test%v: Error building get request: %v", i, err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Test%v: Error doing get request: %v", i, err)
		}
		dec := json.NewDecoder(resp.Body)
		respValue := []gp.Group{}
		err = dec.Decode(&respValue)
		if err != nil {
			t.Fatalf("Error parsing response: %v", err)
		}
		if len(respValue) != test.groupCount {
			t.Fatalf("Got %d groups back but expected %d\n", len(respValue), test.groupCount)
		}
	}
}

func initNGroups(token gp.Token, n int) {
	for i := 0; i < n; i++ {
		data := make(url.Values)
		data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
		data["token"] = []string{token.Token}
		data["university"] = []string{fmt.Sprintf("%t", false)}
		data["name"] = []string{fmt.Sprintf("Group %d", i)}
		req, _ := http.NewRequest("POST", baseURL+"networks", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client.Do(req)
	}
}
