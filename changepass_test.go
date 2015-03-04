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
		ExpectedError:      "crypto/bcrypt: hashedPassword is not the hash of the given password",
	}

	tests := []changePassTest{testGood, testWeakPass, testWrongOldPass}

	for _, cpt := range tests {
		loginResp, err := loginRequest(cpt.Email, cpt.Pass)
		if err != nil {
			t.Fatalf("Error logging in: %v\n", err)
		}
		if loginResp.StatusCode != http.StatusOK {
			t.Fatalf("Got status code %d, expected %d\n", loginResp.StatusCode, http.StatusOK)
		}

		dec := json.NewDecoder(loginResp.Body)
		loginToken := gp.Token{}
		err = dec.Decode(&loginToken)
		if err != nil {
			t.Fatalf("Error decoding login %v\n", err)
		}

		resp, err := changePassRequest(loginToken, cpt.OldPass, cpt.NewPass)
		switch {
		case cpt.ExpectedStatusCode == http.StatusNoContent:
			if cpt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Expected %v, got %v\n", cpt.ExpectedStatusCode, resp.StatusCode)
			}
		case cpt.ExpectedType == "Error":
			if cpt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Expected %v, got %v\n", cpt.ExpectedStatusCode, resp.StatusCode)
			}
			dec = json.NewDecoder(resp.Body)
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
