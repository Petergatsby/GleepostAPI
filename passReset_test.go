package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/mail"
)

func passResetInit(db *sql.DB, tests []passResetTest) (err error) {
	err = initDB()
	if err != nil {
		return
	}

	for _, t := range tests {
		data := make(url.Values)
		data["email"] = []string{t.Email}
		data["pass"] = []string{t.Pass}
		data["first"] = []string{t.First}
		data["last"] = []string{t.Last}
		_, err = client.PostForm(baseURL+"register", data)

		if err != nil {
			return
		}

		if t.VerifyAccount {
			_, err = db.Exec("UPDATE users SET verified = 1 WHERE email = ?", t.Email)
			if err != nil {
				return
			}
		}
	}
	return
}

type passResetTest struct {
	Email              string
	Pass               string
	NewPass            string
	First              string
	Last               string
	VerifyAccount      bool
	BadResetToken      bool
	ResetTwice         bool
	RequestTwice       bool
	ExpectedStatusCode int
	ExpectedError      string
}

func TestPassReset(t *testing.T) {
	db, err := sql.Open("mysql", conf.GetConfig().Mysql.ConnectionString())
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	config := conf.GetConfig()
	api = lib.New(*config)
	api.Mail = mail.NewMock()
	api.Start()
	server := httptest.NewServer(r)
	defer server.Close()
	baseURL = server.URL + "/api/v1/"

	testGood := passResetTest{
		Email:              "pass_reset_test1@fakestanford.edu",
		Pass:               "TestingPass",
		NewPass:            "NewTestingPass",
		First:              "Resetpass",
		Last:               "Test1",
		VerifyAccount:      true,
		BadResetToken:      false,
		ResetTwice:         false,
		RequestTwice:       false,
		ExpectedStatusCode: http.StatusNoContent,
	}
	testBad := passResetTest{
		Email:              "pass_reset_test2@fakestanford.edu",
		Pass:               "TestingPass",
		NewPass:            "NewTestingPass",
		First:              "Resetpass",
		Last:               "Test2",
		VerifyAccount:      true,
		BadResetToken:      true,
		ResetTwice:         false,
		RequestTwice:       false,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Bad password recovery token.",
	}
	testUnverified := passResetTest{
		Email:              "pass_reset_test3@fakestanford.edu",
		Pass:               "TestingPass",
		NewPass:            "NewTestingPass",
		First:              "Resetpass",
		Last:               "Test3",
		VerifyAccount:      false,
		BadResetToken:      false,
		ResetTwice:         false,
		RequestTwice:       false,
		ExpectedStatusCode: http.StatusNoContent,
	}
	testWeakPass := passResetTest{
		Email:              "pass_reset_test4@fakestanford.edu",
		Pass:               "TestingPass",
		NewPass:            "weak",
		First:              "Resetpass",
		Last:               "Test4",
		VerifyAccount:      true,
		BadResetToken:      false,
		ResetTwice:         false,
		RequestTwice:       false,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Password too weak!",
	}
	testResetTwice := passResetTest{
		Email:              "pass_reset_test5@fakestanford.edu",
		Pass:               "TestingPass",
		NewPass:            "NewTestingPass",
		First:              "Resetpass",
		Last:               "Test5",
		VerifyAccount:      true,
		BadResetToken:      false,
		ResetTwice:         true,
		RequestTwice:       false,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Bad password recovery token.",
	}
	testTokenAfterWeak := passResetTest{
		Email:              "pass_reset_test6@fakestanford.edu",
		Pass:               "TestingPass",
		NewPass:            "weak",
		First:              "Resetpass",
		Last:               "Test6",
		VerifyAccount:      true,
		BadResetToken:      false,
		ResetTwice:         true,
		RequestTwice:       false,
		ExpectedStatusCode: http.StatusNoContent,
	}
	testRequestTwice := passResetTest{
		Email:              "pass_reset_test7@fakestanford.edu",
		Pass:               "TestingPass",
		NewPass:            "weak",
		First:              "Resetpass",
		Last:               "Test7",
		VerifyAccount:      true,
		BadResetToken:      false,
		ResetTwice:         true,
		RequestTwice:       true,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Bad password recovery token.",
	}
	tests := []passResetTest{testGood, testBad, testUnverified, testWeakPass, testResetTwice, testTokenAfterWeak, testRequestTwice}

	err = passResetInit(db, tests)
	if err != nil {
		t.Fatal("Problem initializing test state:", err)
	}
	for testNumber, prt := range tests {

		requestResetData := make(url.Values)
		requestResetData["email"] = []string{prt.Email}
		_, err = client.PostForm(baseURL+"profile/request_reset", requestResetData)
		if err != nil {
			t.Fatalf("Test%v: Error making http request: %v\n", testNumber, err)
		}

		userID, resetToken, err := getResetToken(db, prt.Email)
		if err != nil {
			t.Fatalf("Test%v: Error finding reset token: %v\n", testNumber, err)
		}

		if prt.RequestTwice {
			_, err = client.PostForm(baseURL+"profile/request_reset", requestResetData)
			if err != nil {
				t.Fatalf("Test%v: Error making http request: %v\n", testNumber, err)
			}
		}

		if prt.BadResetToken {
			resetToken += "123"
		}

		resetData := make(url.Values)
		resetData["user-id"] = []string{userID}
		resetData["reset-token"] = []string{resetToken}
		resetData["pass"] = []string{prt.NewPass}

		resp, err := client.PostForm(baseURL+"profile/reset/"+userID+"/"+resetToken, resetData)
		if err != nil {
			t.Fatalf("Test%v: Error with reset request: %v\n", testNumber, err)
		}

		if prt.ResetTwice {
			resetData["pass"] = []string{prt.Pass}
			resp, err = client.PostForm(baseURL+"profile/reset/"+userID+"/"+resetToken, resetData)
			if err != nil {
				t.Fatalf("Test%v: Error with reset request: %v\n", testNumber, err)
			}
		}

		if prt.ExpectedStatusCode != resp.StatusCode {
			t.Fatalf("Test%v: %v: Expected %v, got %v\n", testNumber, prt.Last, prt.ExpectedStatusCode, resp.StatusCode)
		}
		switch {
		case prt.ExpectedStatusCode == http.StatusNoContent:
			//Nothing to do
		case prt.ExpectedStatusCode == http.StatusBadRequest:
			dec := json.NewDecoder(resp.Body)
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Test%v: Error parsing error: %v\n", testNumber, err)
			}
			if errorValue.Reason != prt.ExpectedError {
				t.Fatalf("Test%v: Expected %s, got %s\n", testNumber, prt.ExpectedError, errorValue.Reason)
			}
		default:
			t.Fatalf("Test%v: Something completely unexpected happened", testNumber)
		}
	}
}

func getResetToken(db *sql.DB, email string) (userID string, token string, err error) {
	err = db.QueryRow("SELECT users.id FROM users WHERE users.email = ?", email).Scan(&userID)
	if err != nil {
		return
	}

	err = db.QueryRow("SELECT token FROM password_recovery WHERE password_recovery.user = ?", userID).Scan(&token)
	return
}
