package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func TestLogin(t *testing.T) {

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
	//Bad password
	resp, err = loginRequest(email, "bad pass")
	if err != nil {
		t.Fatalf("Error logging in: %v\n", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("Got status code %d, expected %d\n", resp.StatusCode, 400)
	}
	errorValue := gp.APIerror{}
	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&errorValue)
	if err != nil {
		t.Fatalf("Error parsing error: %v\n", err)
	}
	if errorValue.Reason != "Bad username/password" {
		t.Fatalf("Expected %s, got %s\n", "Bad username/password", errorValue.Reason)
	}

}

func loginRequest(email, pass string) (resp *http.Response, err error) {
	baseUrl := "https://dev.gleepost.com/api/v1/"
	data := make(url.Values)
	client := &http.Client{}
	data["email"] = []string{email}
	data["pass"] = []string{pass}
	resp, err = client.PostForm(baseUrl+"login", data)
	return
}
