package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func TestChangePass(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

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

	for _, cpt := range tests {
		token, err := testingGetSession(cpt.Email, cpt.Pass)
		if err != nil {
			t.Fatal("Error logging in:", err)
		}

		resp, err := changePassRequest(token, cpt.OldPass, cpt.NewPass)
		if cpt.ExpectedStatusCode != resp.StatusCode {
			t.Fatalf("Expected %v, got %v\n", cpt.ExpectedStatusCode, resp.StatusCode)
		}
		switch {
		case cpt.ExpectedStatusCode == http.StatusNoContent:
			//All done
		case cpt.ExpectedType == "Error":
			dec := json.NewDecoder(resp.Body)
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Error parsing error: %v\n", err)
			}
			if errorValue.Reason != cpt.ExpectedError {
				t.Fatalf("Expected %s, got %s\n", cpt.ExpectedError, errorValue.Reason)
			}
		default:
			t.Fatalf("Something completely unexpected happened")
		}
	}
}

func changePassRequest(token gp.Token, oldPass string, newPass string) (resp *http.Response, err error) {
	data := make(url.Values)
	client := &http.Client{}
	data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
	data["token"] = []string{token.Token}
	data["old"] = []string{oldPass}
	data["new"] = []string{newPass}
	resp, err = client.PostForm(baseURL+"profile/change_pass", data)
	return
}

func testingGetSession(email, pass string) (token gp.Token, err error) {
	data := make(url.Values)
	client := &http.Client{}
	data["email"] = []string{email}
	data["pass"] = []string{pass}
	resp, err := client.PostForm(baseURL+"login", data)
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
