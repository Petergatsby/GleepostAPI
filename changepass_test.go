package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func TestChangePass(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	config := conf.GetConfig()
	api = lib.New(*config)
	api.Start()
	server := httptest.NewServer(r)
	defer server.Close()
	baseURL = server.URL + "/api/v1/"

	type changePassTest struct {
		Email              string
		Pass               string
		OldPass            string
		NewPass            string
		ExpectedStatusCode int
		ExpectedType       string
		ExpectedError      string
	}
	testGood := changePassTest{
		Email:              "patrick@fakestanford.edu",
		Pass:               "TestingPass",
		OldPass:            "TestingPass",
		NewPass:            "TestingPass2",
		ExpectedStatusCode: http.StatusNoContent,
	}
	testWeakPass := changePassTest{
		Email:              "patrick@fakestanford.edu",
		Pass:               "TestingPass2",
		OldPass:            "TestingPass2",
		NewPass:            "hi",
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedType:       "Error",
		ExpectedError:      "Password too weak!",
	}
	testWrongOldPass := changePassTest{
		Email:              "patrick@fakestanford.edu",
		Pass:               "TestingPass2",
		OldPass:            "TestingPass",
		NewPass:            "TestingPassword",
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedType:       "Error",
		ExpectedError:      "The password you have provided is incorrect",
	}

	tests := []changePassTest{testGood, testWeakPass, testWrongOldPass}

	for testNumber, cpt := range tests {
		token, err := testingGetSession(cpt.Email, cpt.Pass)
		if err != nil {
			t.Fatalf("Test%v: Error logging in: %s\n", testNumber, err)
		}

		resp, err := changePassRequest(token, cpt.OldPass, cpt.NewPass)
		if cpt.ExpectedStatusCode != resp.StatusCode {
			t.Fatalf("Test%v: Expected %v, got %v\n", testNumber, cpt.ExpectedStatusCode, resp.StatusCode)
		}
		switch {
		case cpt.ExpectedStatusCode == http.StatusNoContent:
			//All done
		case cpt.ExpectedType == "Error":
			dec := json.NewDecoder(resp.Body)
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Test%v: Error parsing error: %v\n", testNumber, err)
			}
			if errorValue.Reason != cpt.ExpectedError {
				t.Fatalf("Test%v: Expected %s, got %s\n", testNumber, cpt.ExpectedError, errorValue.Reason)
			}
		default:
			t.Fatalf("Test%v: Something completely unexpected happened", testNumber)
		}
	}
}

func changePassRequest(token gp.Token, oldPass string, newPass string) (resp *http.Response, err error) {
	data := make(url.Values)
	data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
	data["token"] = []string{token.Token}
	data["old"] = []string{oldPass}
	data["new"] = []string{newPass}
	req, err := http.NewRequest("POST", baseURL+"profile/change_pass", strings.NewReader(data.Encode()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Close = true

	resp, err = client.Do(req)
	return
}

func testingGetSession(email, pass string) (token gp.Token, err error) {
	data := make(url.Values)
	data["email"] = []string{email}
	data["pass"] = []string{pass}

	req, err := http.NewRequest("POST", baseURL+"login", strings.NewReader(data.Encode()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Close = true

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&token)
	if token.UserID <= 0 || len(token.Token) != 64 {
		err = errors.New("Invalid token")
	}
	return
}
