package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Petergatsby/GleepostAPI/lib/conf"
	"github.com/Petergatsby/GleepostAPI/lib/gp"
)

var baseURL = "http://localhost:8083/api/v1/"
var client = &http.Client{}

func TestLogin(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	once.Do(setup)

	type loginTest struct {
		Email              string
		Pass               string
		ExpectedStatusCode int
		ExpectedType       string
		ExpectedError      string
		ExpectedStatus     string
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
	unverifiedLogin := loginTest{
		Email:              "beetlebum@fakestanford.edu",
		Pass:               "TestingPass",
		ExpectedStatusCode: http.StatusForbidden,
		ExpectedType:       "Status",
		ExpectedStatus:     "unverified",
	}
	tests := []loginTest{badLogin, goodLogin, unverifiedLogin}
	for testNumber, lt := range tests {
		resp, err := loginRequest(lt.Email, lt.Pass)
		if err != nil {
			t.Fatalf("Test%v: Error logging in: %v\n", testNumber, err)
		}
		if resp.StatusCode != lt.ExpectedStatusCode {
			t.Fatalf("Test%v: Got status code %d, expected %d\n", testNumber, resp.StatusCode, lt.ExpectedStatusCode)
		}
		dec := json.NewDecoder(resp.Body)
		switch {
		case lt.ExpectedType == "Error":
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Test%v: Error parsing error: %v\n", testNumber, err)
			}
			if errorValue.Reason != lt.ExpectedError {
				t.Fatalf("Test%v: Expected %s, got %s\n", testNumber, lt.ExpectedError, errorValue.Reason)
			}
		case lt.ExpectedType == "Token":
			token := gp.Token{}
			err = dec.Decode(&token)
			if err != nil {
				t.Fatalf("Test%v: Error parsing %s: %v", testNumber, lt.ExpectedType, err)
			}
			if token.UserID <= 0 {
				t.Fatalf("Test%v: User ID is not valid: got %d\n", testNumber, token.UserID)
			}
			if len(token.Token) != 64 {
				t.Fatalf("Test%v: Token too short: expected %d but got %d\n", testNumber, 64, len(token.Token))
			}
			expectedMaxExpiry := time.Now().AddDate(1, 0, 0).Add(1 * time.Minute)
			if token.Expiry.After(expectedMaxExpiry) {
				t.Fatalf("Test%v: Token expiration longer than it should be: %v is after %v\n", testNumber, token.Expiry, expectedMaxExpiry)
			}
			if token.Expiry.AddDate(-1, 0, 0).Before(time.Now().Add(-1 * time.Minute)) {
				t.Fatalf("Test%v: Token expiration shorter than it should be!", testNumber)
			}
		case lt.ExpectedType == "Status":
			status := gp.Status{}
			err = dec.Decode(&status)
			if err != nil {
				t.Fatalf("Test%v: Error parsing status: %v\n", testNumber, err)
			}
			if status.Status != lt.ExpectedStatus {
				t.Fatalf("Test%v: Expected %s, got %s", testNumber, lt.ExpectedStatus, status.Status)
			}
		default:
			t.Fatalf("Test%v: Something completely unexpected happened", testNumber)
		}
	}
}

func BenchmarkLogin(b *testing.B) {
	b.ReportAllocs()
	err := initDB()
	if err != nil {
		b.FailNow()
	}
	once.Do(setup)
	email := "patrick@fakestanford.edu"
	pass := "TestingPass"
	for i := 0; i < b.N; i++ {
		_, err := loginRequest(email, pass)
		if err != nil {
			b.FailNow()
		}
	}
}

func initDB() error {
	config := conf.GetConfig()
	db, err := sql.Open("mysql", config.Mysql.ConnectionString())
	if err != nil {
		return err
	}
	defer db.Close()
	err = truncate("network", "net_rules", "users", "user_network", "uploads", "post_images")
	if err != nil {
		return err
	}
	res, err := db.Exec("INSERT INTO `network` (`name`, `is_university`, `privacy`, `user_group`) VALUES (?, ?, NULL, ?)", "Fake Stanford", true, true)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO `net_rules` (network_id, rule_type, rule_value) VALUES (?, ?, ?)", id, "email", "fakestanford.edu")
	if err != nil {
		return err
	}
	res, err = db.Exec("INSERT INTO `users` (`password`, `email`, `verified`, `firstname`, `lastname`) VALUES ('$2a$10$xLUmQbvrHAAOGuv4.uHAY.NmoLGEuEObENPiQ8kkh.Miyvdzhyge6', 'patrick@fakestanford.edu', 1, 'Patrick', 'Molgaard')")
	if err != nil {
		return err
	}
	uid, err := res.LastInsertId()
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO `users` (`password`, `email`, `verified`, `firstname`, `lastname`) VALUES ('$2a$10$xLUmQbvrHAAOGuv4.uHAY.NmoLGEuEObENPiQ8kkh.Miyvdzhyge6', 'beetlebum@fakestanford.edu', 0, 'Beetle', 'Bum')")
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO `user_network` (`user_id`, `network_id`) VALUES (?, ?)", uid, id)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO `user_network` (`user_id`, `network_id`) VALUES (?, ?)", uid+1, id)
	if err != nil {
		return err
	}
	return nil
}

func truncate(tables ...string) error {
	config := conf.GetConfig()
	db, err := sql.Open("mysql", config.Mysql.ConnectionString())
	if err != nil {
		return err
	}
	defer db.Close()
	if err != nil {
		return err
	}
	for _, t := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE `%s`", t))
		if err != nil {
			return err
		}
	}
	return nil
}

func loginRequest(email, pass string) (resp *http.Response, err error) {
	data := make(url.Values)
	data["email"] = []string{email}
	data["pass"] = []string{pass}
	req, err := http.NewRequest("POST", baseURL+"login", strings.NewReader(data.Encode()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Close = true

	resp, err = client.Do(req)
	return
}
