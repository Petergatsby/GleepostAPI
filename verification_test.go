package main

import (
	"database/sql"
	"net/http"
	"net/url"
	"testing"

	"github.com/draaglom/GleepostAPI/lib/conf"
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

		if vt.ExpectedStatusCode != resp.StatusCode {
			t.Fatalf("Expected %v, got %v\n", vt.ExpectedStatusCode, resp.StatusCode)
		}

		_, err = testingGetSession(vt.Email, vt.Pass)
		if err != nil {
			t.Fatalf("Error logging in: %v\n", err)
		}
	}
}
