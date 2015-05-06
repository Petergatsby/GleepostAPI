package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/mail"
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

	config := conf.GetConfig()
	api = lib.New(*config)
	api.Mail = mail.NewMock()
	api.Start()
	server := httptest.NewServer(r)
	baseURL = server.URL + "/api/v1/"

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
	for testNumber, vt := range tests {

		data := make(url.Values)
		data["email"] = []string{vt.Email}
		data["pass"] = []string{vt.Pass}
		data["first"] = []string{vt.First}
		data["last"] = []string{vt.Last}
		resp, err := client.PostForm(baseURL+"register", data)

		if err != nil {
			t.Fatalf("Test%v: Error making http request: %v\n", testNumber, err)
		}

		var token string
		if vt.TestValidToken {
			token, err = getToken(db, vt.Email)
			if err != nil {
				t.Fatalf("Test%v: Error finding token: %v\n", testNumber, err)
			}
		} else {
			token = "12345lolololtest"
		}
		resp, err = client.PostForm(baseURL+"verify/"+token, make(url.Values))
		if err != nil {
			t.Fatalf("Test%v: Error with verification request: %v\n", testNumber, err)
		}
		if vt.VerifyTwice {
			resp, err = client.PostForm(baseURL+"verify/"+token, make(url.Values))
			if err != nil {
				t.Fatalf("Test%v: Error with verification request: %v\n", testNumber, err)
			}
		}

		if vt.ExpectedStatusCode != resp.StatusCode {
			dec := json.NewDecoder(resp.Body)
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			log.Println(errorValue.Reason)
			t.Fatalf("Test%v: Expected %v, got %v\n", testNumber, vt.ExpectedStatusCode, resp.StatusCode)
		}
		switch {
		case vt.ExpectedStatusCode == http.StatusOK:
		case vt.ExpectedStatusCode == http.StatusBadRequest:
			dec := json.NewDecoder(resp.Body)
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Test%v: Error parsing error: %v\n", testNumber, err)
			}
			if errorValue.Reason != vt.ExpectedError {
				t.Fatalf("Test%v: Expected %s, got %s\n", testNumber, vt.ExpectedError, errorValue.Reason)
			}
		default:
			t.Fatalf("Test%v: Something completely unexpected happened", testNumber)
		}

		_, err = testingGetSession(vt.Email, vt.Pass)
		if err != nil && vt.TestValidToken {
			t.Fatalf("Test%v: Error logging in: %v\n", testNumber, err)
		} else if err == nil && !vt.TestValidToken {
			t.Fatalf("Test%v: Should not have been able to log in.", testNumber)
		}
	}
}

func getToken(db *sql.DB, email string) (token string, err error) {
	err = db.QueryRow("SELECT token FROM verification JOIN users ON users.id = verification.user_id WHERE users.email = ?", email).Scan(&token)
	return
}
