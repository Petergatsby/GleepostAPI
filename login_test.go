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

func TestInit(t *testing.T) {
	//Init
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}
	go main()
	time.Sleep(500 * time.Millisecond) //Time to spin up
}

func TestRegister(t *testing.T) {
	data := make(url.Values)
	client := &http.Client{}
	data["email"] = []string{"patrick@fakestanford.edu"}
	data["pass"] = []string{"TestingPass"}
	data["first"] = []string{"Patrick"}
	data["last"] = []string{"Molgaard"}
	resp, err := client.PostForm(baseUrl+"register", data)
	if err != nil {
		t.Fatalf("Error making http request: %v\n", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Wrong status code: Got %v but was expecting %v", resp.StatusCode, http.StatusCreated)
	}
	created := gp.NewUser{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&created)
	if err != nil {
		t.Fatalf("Error parsing registration response: %v\n", err)
	}
	if created.Status != "unverified" {
		t.Fatalf("Status should be 'unverified', but is actually: %s\n", created.Status)
	}
}

func TestLoginBadPass(t *testing.T) {
	//Bad password
	resp, err := loginRequest("patrick@fakestanford.edu", "bad pass")
	if err != nil {
		t.Fatalf("Error logging in: %v\n", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("Got status code %d, expected %d\n", resp.StatusCode, 400)
	}
	errorValue := gp.APIerror{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&errorValue)
	if err != nil {
		t.Fatalf("Error parsing error: %v\n", err)
	}
	if errorValue.Reason != "Bad username/password" {
		t.Fatalf("Expected %s, got %s\n", "Bad username/password", errorValue.Reason)
	}
}

func TestLoginGood(t *testing.T) {
	//Good user
	email := "patrick@fakestanford.edu"
	pass := "TestingPass"

	resp, err := loginRequest(email, pass)
	if err != nil {
		t.Fatalf("Error logging in: %v\n", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Got status code %d, expected %d\n", resp.StatusCode, 200)
	}
	token := gp.Token{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&token)
	if err != nil {
		t.Fatalf("Error parsing token: %v\n", err)
	}
	if token.UserID != 2909 {
		t.Fatalf("Got user %d, was expecting %d\n", token.UserID, 2909)
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
