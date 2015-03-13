package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func TestVerification(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	db, err := sql.Open("mysql", conf.GetConfig().Mysql.ConnectionString())
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	client := &http.Client{}

	type verificationTest struct {
		Email              string
		Pass               string
		First              string
		Last               string
		VerifyTwice        bool
		TestValidToken     bool
		ExpectedStatusCode int
		ExpectedError      string
	}
	testGood := verificationTest{
		Email:              "verification_test1@fakestanford.edu",
		Pass:               "TestingPass",
		First:              "Verification",
		Last:               "Test1",
		VerifyTwice:        false,
		TestValidToken:     true,
		ExpectedStatusCode: http.StatusOK,
	}
	testTwice := verificationTest{
		Email:              "verification_test2@fakestanford.edu",
		Pass:               "TestingPass",
		First:              "Verification",
		Last:               "Test2",
		VerifyTwice:        true,
		TestValidToken:     true,
		ExpectedStatusCode: http.StatusOK,
	}
	testBad := verificationTest{
		Email:              "verification_test3@fakestanford.edu",
		Pass:               "TestingPass",
		First:              "Verification",
		Last:               "Test3",
		VerifyTwice:        true,
		TestValidToken:     false,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Bad verification token",
	}
	tests := []verificationTest{testGood, testTwice, testBad}
	for _, vt := range tests {

		data := make(url.Values)
		data["email"] = []string{vt.Email}
		data["pass"] = []string{vt.Pass}
		data["first"] = []string{vt.First}
		data["last"] = []string{vt.Last}
		resp, err := client.PostForm(baseURL+"register", data)

		if err != nil {
			t.Fatalf("Error making http request: %v\n", err)
		}

		if vt.TestValidToken {
			var token string
			err = db.QueryRow("SELECT token FROM verification JOIN users ON users.id = verification.user_id WHERE users.email = ?", vt.Email).Scan(&token)

			if err != nil {
				t.Fatalf("Error finding token: %v\n", err)
			}
			if token == "" {
				t.Fatalf("Incorrect token retrieved: %v\n", token)
			}

			resp, err = client.PostForm(baseURL+"verify/"+token, make(url.Values))
			if err != nil {
				t.Fatalf("Error with verification request: %v\n", err)
			}
			if vt.VerifyTwice {
				resp, err = client.PostForm(baseURL+"verify/"+token, make(url.Values))
				if err != nil {
					t.Fatalf("Error with verification request: %v\n", err)
				}
			}
		} else {
			resp, err = client.PostForm(baseURL+"verify/12345lolololtest", make(url.Values))
			if err != nil {
				t.Fatalf("Error with verification request: %v\n", err)
			}
		}

		switch {
		case vt.ExpectedStatusCode == http.StatusOK:
			if vt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Expected %v, got %v\n", vt.ExpectedStatusCode, resp.StatusCode)
			}
		case vt.ExpectedStatusCode == http.StatusBadRequest:
			if vt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Expected %v, got %v\n", vt.ExpectedStatusCode, resp.StatusCode)
			}

			dec := json.NewDecoder(resp.Body)
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Error parsing error: %v\n", err)
			}
			if errorValue.Reason != vt.ExpectedError {
				t.Fatalf("Expected %s, got %s\n", vt.ExpectedError, errorValue.Reason)
			}
		}

		_, err = testingGetSession(vt.Email, vt.Pass)
		if err != nil && vt.TestValidToken {
			t.Fatalf("Error logging in: %v\n", err)
		} else if err == nil && !vt.TestValidToken {
			t.Fatalf("Should not have been able to log in.")
		}
	}
}
