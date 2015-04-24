package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func TestRegister(t *testing.T) {
	//Init
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	config := conf.GetConfig()
	api = lib.New(*config)
	api.Start()
	server := httptest.NewServer(r)
	baseURL = server.URL + "/api/v1/"

	client := &http.Client{}

	type registrationTest struct {
		Email              string
		Pass               string
		First              string
		Last               string
		ExpectedStatusCode int
		ExpectedReturnType string
		ExpectedError      string
		ExpectedRegStatus  string
	}
	testGood := registrationTest{
		Email:              "dominic@fakestanford.edu",
		Pass:               "TestingPass",
		First:              "Dominic",
		Last:               "Mortlock",
		ExpectedStatusCode: http.StatusCreated,
		ExpectedReturnType: "NewUser",
		ExpectedRegStatus:  "unverified",
	}
	testNoEmail := registrationTest{
		Email:              "",
		Pass:               "TestingPass",
		First:              "Patrick",
		Last:               "Molgaard",
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedReturnType: "Error",
		ExpectedError:      "Missing parameter: email",
	}
	testInvalidEmail := registrationTest{
		Email:              "cheese@realstanford.edu",
		Pass:               "TestingPass",
		First:              "Cheese",
		Last:               "Pizza",
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedReturnType: "Error",
		ExpectedError:      "Invalid Email",
	}
	testExistingUser := registrationTest{
		Email:              "beetlebum@fakestanford.edu",
		Pass:               "TestingPass",
		First:              "Beetle",
		Last:               "Bum",
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedReturnType: "Error",
		ExpectedError:      "Username or email address already taken",
	}
	testWeakPass := registrationTest{
		Email:              "cow@fakestanford.edu",
		Pass:               "pass",
		First:              "Cow",
		Last:               "Beef",
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedReturnType: "Error",
		ExpectedError:      "Password too weak!",
	}
	tests := []registrationTest{testGood, testNoEmail, testInvalidEmail, testExistingUser, testWeakPass}

	for testNumber, r := range tests {
		data := make(url.Values)
		data["email"] = []string{r.Email}
		data["pass"] = []string{r.Pass}
		data["first"] = []string{r.First}
		data["last"] = []string{r.Last}
		resp, err := client.PostForm(baseURL+"register", data)
		if err != nil {
			t.Fatalf("Test%v: Error making http request: %v\n", testNumber, err)
		}
		if resp.StatusCode != r.ExpectedStatusCode {
			t.Fatalf("Test%v: Wrong status code: Got %v but was expecting %v", testNumber, resp.StatusCode, r.ExpectedStatusCode)
		}
		dec := json.NewDecoder(resp.Body)
		switch {
		case r.ExpectedReturnType == "NewUser":
			created := gp.NewUser{}
			err = dec.Decode(&created)
			if err != nil {
				t.Fatalf("Test%v: Error parsing registration response as %s: %v\n", testNumber, r.ExpectedReturnType, err)
			}
			if created.Status != r.ExpectedRegStatus {
				t.Fatalf("Test%v: Status should be %s, but is actually: %s\n", testNumber, r.ExpectedRegStatus, created.Status)
			}
		case r.ExpectedReturnType == "Error":
			errorResp := gp.APIerror{}
			err = dec.Decode(&errorResp)
			if err != nil {
				t.Fatalf("Test%v: Error parsing registration response as %s: %v\n", testNumber, r.ExpectedReturnType, err)
			}
			if errorResp.Reason != r.ExpectedError {
				t.Fatalf("Test%v: Saw error: %s, was expecting: %s\n", testNumber, errorResp.Reason, r.ExpectedError)
			}
		default:
			t.Fatalf("Test%v: Something completely unexpected happened", testNumber)
		}
	}
}
