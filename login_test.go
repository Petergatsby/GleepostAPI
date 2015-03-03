package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

var baseUrl = "http://localhost:8083/api/v1/"

func TestLogin(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	type loginTest struct {
		Email              string
		Pass               string
		ExpectedStatusCode int
		ExpectedType       string
		ExpectedError      string
	}
	badLogin := loginTest{
		Email:              "patrick@fakestanford.edu",
		Pass:               "bad pass",
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedType:       "Error",
		ExpectedError:      "Bad username/password",
	}
	goodLogin := loginTest{
		Email:              "patrick@fakestanford.edu",
		Pass:               "TestingPass",
		ExpectedStatusCode: http.StatusOK,
		ExpectedType:       "Token",
	}
	tests := []loginTest{badLogin, goodLogin}
	for _, lt := range tests {
		resp, err := loginRequest(lt.Email, lt.Pass)
		if err != nil {
			t.Fatalf("Error logging in: %v\n", err)
		}
		if resp.StatusCode != lt.ExpectedStatusCode {
			t.Fatalf("Got status code %d, expected %d\n", resp.StatusCode, lt.ExpectedStatusCode)
		}
		dec := json.NewDecoder(resp.Body)
		switch {
		case lt.ExpectedType == "Error":
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Error parsing error: %v\n", err)
			}
			if errorValue.Reason != lt.ExpectedError {
				t.Fatalf("Expected %s, got %s\n", lt.ExpectedError, errorValue.Reason)
			}
		case lt.ExpectedType == "Token":
			token := gp.Token{}
			err = dec.Decode(&token)
			if err != nil {
				t.Fatalf("Error parsing %s: %v", lt.ExpectedType, err)
			}
			if token.UserID <= 0 {
				t.Fatalf("User ID is not valid: got %d\n", token.UserID)
			}
			if len(token.Token) != 64 {
				t.Fatalf("Token too short: expected %d but got %d\n", 64, len(token.Token))
			}
			if token.Expiry.AddDate(-1, 0, 0).After(time.Now().Add(1 * time.Minute)) {
				t.Fatalf("Token expiration longer than it should be!")
			}
			if token.Expiry.AddDate(-1, 0, 0).Before(time.Now().Add(-1 * time.Minute)) {
				t.Fatalf("Token expiration shorter than it should be!")
			}
		}
	}
}

func initDB() error {
	config := conf.GetConfig()
	db, err := sql.Open("mysql", config.Mysql.ConnectionString())
	if err != nil {
		return err
	}
	_, err = db.Exec("TRUNCATE TABLE `network`")
	if err != nil {
		return err
	}
	stmt, err := db.Prepare("INSERT INTO `network` (`name`, `is_university`, `privacy`, `user_group`) VALUES (?, ?, NULL, ?)")
	if err != nil {
		return err
	}
	res, err := stmt.Exec("Fake Stanford", true, true)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	_, err = db.Exec("TRUNCATE TABLE `net_rules`")
	if err != nil {
		return err
	}
	stmt, err = db.Prepare("INSERT INTO `net_rules` (network_id, rule_type, rule_value) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(id, "email", "fakestanford.edu")
	if err != nil {
		return err
	}
	_, err = db.Exec("TRUNCATE TABLE `users`")
	if err != nil {
		return err
	}
	return nil
}

func loginRequest(email, pass string) (resp *http.Response, err error) {
	data := make(url.Values)
	client := &http.Client{}
	data["email"] = []string{email}
	data["pass"] = []string{pass}
	resp, err = client.PostForm(baseUrl+"login", data)
	return
}
