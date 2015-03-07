package main

import (
	"encoding/json"
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
			t.Fatalf("Error logging in:", err)
		}

		resp, err := changePassRequest(token, cpt.OldPass, cpt.NewPass)
		switch {
		case cpt.ExpectedStatusCode == http.StatusNoContent:
			if cpt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Expected %v, got %v\n", cpt.ExpectedStatusCode, resp.StatusCode)
			}
		case cpt.ExpectedType == "Error":
			if cpt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Expected %v, got %v\n", cpt.ExpectedStatusCode, resp.StatusCode)
			}
			dec := json.NewDecoder(resp.Body)
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Error parsing error: %v\n", err)
			}
			if errorValue.Reason != cpt.ExpectedError {
				t.Fatalf("Expected %s, got %s\n", cpt.ExpectedError, errorValue.Reason)
			}
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
	resp, err = client.PostForm(baseUrl+"profile/change_pass", data)
	return
}

func testingGetSession(email, pass string) (token gp.Token, err error) {
	data := make(url.Values)
	client := &http.Client{}
	data["email"] = []string{email}
	data["pass"] = []string{pass}
	resp, err := client.PostForm(baseUrl+"login", data)
	if err != nil {
		return
	}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&token)
	return
}
