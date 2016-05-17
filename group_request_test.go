package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Petergatsby/GleepostAPI/lib/conf"
	"github.com/Petergatsby/GleepostAPI/lib/gp"
)

func TestRequestAccess(t *testing.T) {
	err := initGroups()
	if err != nil {
		t.Fatal("Init error:", err)
	}
	once.Do(setup)
	type accesstest struct {
		token          gp.Token
		groupID        gp.NetworkID
		expectedStatus int
		expectedType   string
		expectedError  string
	}
	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatal("Auth error:", err)
	}
	tests := []accesstest{
		{ //A non-group network
			token:          token,
			groupID:        2,
			expectedStatus: http.StatusNotFound,
			expectedType:   "Error",
			expectedError:  "No such network",
		},
		{ //a group in another university
			token:          token,
			groupID:        6,
			expectedStatus: http.StatusNotFound,
			expectedType:   "Error",
			expectedError:  "No such network",
		},
		{ //A nonexistent group
			token:          token,
			groupID:        999,
			expectedStatus: http.StatusNotFound,
			expectedType:   "Error",
			expectedError:  "No such network",
		},
		{ //A secret group
			token:          token,
			groupID:        5,
			expectedStatus: http.StatusNotFound,
			expectedType:   "Error",
			expectedError:  "No such network",
		},
		{ //A public group
			token:          token,
			groupID:        3,
			expectedStatus: http.StatusForbidden,
			expectedType:   "Error",
			expectedError:  "You're not allowed to do that!",
		},
		{ //a group you're a member of already
			token:          token,
			groupID:        7,
			expectedStatus: http.StatusForbidden,
			expectedType:   "Error",
			expectedError:  "You're not allowed to do that!",
		},
		{ //good case
			token:          token,
			groupID:        4,
			expectedStatus: http.StatusNoContent,
		},
		{ //request again: idempotent case
			token:          token,
			groupID:        4,
			expectedStatus: http.StatusNoContent,
		},
	}
	for n, test := range tests {
		data := make(url.Values)
		data["id"] = []string{fmt.Sprintf("%d", test.token.UserID)}
		data["token"] = []string{test.token.Token}
		req, _ := http.NewRequest("POST", baseURL+"networks/"+fmt.Sprintf("%d", test.groupID)+"/requests", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Close = true
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Test %d: Couldn't make http request: %s\n", n, err)
		}
		if resp.StatusCode != test.expectedStatus {
			t.Fatalf("Test %d: Expected status code %d, got %d\n", n, test.expectedStatus, resp.StatusCode)
		}
		dec := json.NewDecoder(resp.Body)
		switch {
		case test.expectedType == "Error":
			errResp := gp.APIerror{}
			err = dec.Decode(&errResp)
			if err != nil {
				t.Fatalf("Test %d: Failed to decode as %s: %v\n", n, test.expectedType, err)
			}
			if errResp.Reason != test.expectedError {
				t.Fatalf("Test %d: Wrong error: Expected %s but got %s\n", n, test.expectedError, errResp.Reason)
			}
		}
	}
	truncate("networks", "user_network")
}

func initGroups() error {
	err := initDB()
	if err != nil {
		return err
	}
	config := conf.GetConfig()
	db, err := sql.Open("mysql", config.Mysql.ConnectionString())
	if err != nil {
		return err
	}
	q := "INSERT INTO `network` (`name`, `parent`, `is_university`, `privacy`, `user_group`) VALUES (?, ?, ?, ?, ?)"
	db.Exec(q, "University of Leeds", 0, 1, "", 0)
	db.Exec(q, "Public group", 1, 0, "public", 1)
	db.Exec(q, "Private group", 1, 0, "private", 1)
	db.Exec(q, "Secret group", 1, 0, "secret", 1)
	db.Exec(q, "Other network group", 2, 0, "private", 1)
	db.Exec(q, "Group you're in", 1, 0, "private", 1)
	db.Exec("INSERT INTO user_network (user_id, network_id) VALUES (1, 7)")
	return err
}
