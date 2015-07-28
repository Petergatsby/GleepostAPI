package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

type rejectTest struct {
	token              gp.Token
	userID             gp.UserID
	groupID            gp.NetworkID
	expectedStatusCode int
	expectedError      string
}

func TestRejectGroupRequest(t *testing.T) {
	truncate("network_requests")
	err := initDB()
	if err != nil {
		t.Fatal("Error initializing db:", err)
	}
	once.Do(setup)

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatal("Error getting session:", err)
	}
	initNGroups(token, 5)
	initRequests()
	initMemberships()
	tests := []rejectTest{
		//No request
		{token: token, userID: 999999, groupID: 2, expectedStatusCode: 404, expectedError: "No such request"},
		//No network
		{token: token, userID: 123, groupID: 999999, expectedStatusCode: 404, expectedError: "No such network"},
		//Not staff
		{token: token, userID: 2, groupID: 3, expectedStatusCode: 403, expectedError: "You're not allowed to do that!"},
		//Not in the group at all
		{token: token, userID: 2, groupID: 4, expectedStatusCode: 403, expectedError: "You're not allowed to do that!"},
		//Already rejected
		{token: token, userID: 2, groupID: 5, expectedStatusCode: 403, expectedError: "Request is already rejected"},
		//Already accepted
		{token: token, userID: 2, groupID: 6, expectedStatusCode: 403, expectedError: "Request is already accepted"},
		//Good case
		{token: token, userID: 2, groupID: 2, expectedStatusCode: 204},
	}
	//Reject (good case)
	// -> 204
	for _, test := range tests {
		req, err := http.NewRequest("DELETE", fmt.Sprintf("%snetworks/%d/requests/%d", baseURL, test.groupID, test.userID), nil)
		if err != nil {
			t.Fatal("Error building DELETE request:", err)
		}
		req.Header.Set("X-GP-Auth", fmt.Sprintf("%d-%s", test.token.UserID, test.token.Token))
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal("Error doing DELETE request:", err)
		}
		if resp.StatusCode != test.expectedStatusCode {
			t.Fatal("Expected:", test.expectedStatusCode, "but got:", resp.StatusCode)
		}
		if test.expectedError != "" {
			errResp := gp.APIerror{}
			dec := json.NewDecoder(resp.Body)
			err = dec.Decode(&errResp)
			if err != nil {
				t.Fatal("Error parsing error json:", err)
			}
			if errResp.Reason != test.expectedError {
				t.Fatal("Expected error:", test.expectedError, "but got:", errResp.Reason)
			}
		}
	}
}

func initRequests() error {
	config := conf.GetConfig()
	db, err := sql.Open("mysql", config.Mysql.ConnectionString())
	if err != nil {
		return err
	}
	defer db.Close()
	s, err := db.Prepare("INSERT INTO network_requests(user_id, network_id, status) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer s.Close()
	s.Exec(2, 2, "pending")
	s.Exec(2, 3, "pending")
	s.Exec(2, 4, "pending")
	s.Exec(2, 5, "rejected")
	s.Exec(2, 6, "accepted")

	return nil
}

func initMemberships() error {
	config := conf.GetConfig()
	db, err := sql.Open("mysql", config.Mysql.ConnectionString())
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("UPDATE user_network SET `role` = 'member', `role_level` = 1 WHERE user_id = ? AND network_id = ?", 1, 3)
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM user_network WHERE user_id = ? AND network_id = ?", 1, 4)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO user_network (user_id, network_id) VALUES (?, ?)", 2, 6)
	return err
}
