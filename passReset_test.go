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

func TestPassReset(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	db, err := sql.Open("mysql", conf.GetConfig().Mysql.ConnectionString())
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	client := &http.Client{}

	type passResetTest struct {
		Email              string
		Pass               string
		NewPass            string
		First              string
		Last               string
		VerifyAccount      bool
		BadResetToken      bool
		ExpectedStatusCode int
		ExpectedError      string
	}
	testGood := passResetTest{
		Email:              "pass_reset_test1@fakestanford.edu",
		Pass:               "TestingPass",
		NewPass:            "NewTestingPass",
		First:              "Resetpass",
		Last:               "Test1",
		VerifyAccount:      true,
		BadResetToken:      false,
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
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "Password too weak!",
	}
	tests := []passResetTest{testGood, testBad, testUnverified, testWeakPass}

	for _, prt := range tests {

		data := make(url.Values)
		data["email"] = []string{prt.Email}
		data["pass"] = []string{prt.Pass}
		data["first"] = []string{prt.First}
		data["last"] = []string{prt.Last}
		_, err := client.PostForm(baseURL+"register", data)

		if err != nil {
			t.Fatalf("Error making http request: %v\n", err)
		}

		if prt.VerifyAccount {
			var token string
			err = db.QueryRow("SELECT token FROM verification JOIN users ON users.id = verification.user_id WHERE users.email = ?", prt.Email).Scan(&token)

			if err != nil {
				t.Fatalf("Error finding token: %v\n", err)
			}
			if token == "" {
				t.Fatalf("Incorrect token retrieved: %v\n", token)
			}

			_, err = client.PostForm(baseURL+"verify/"+token, make(url.Values))

			if err != nil {
				t.Fatalf("Error with verification request: %v\n", err)
			}
		}

		requestResetData := make(url.Values)
		requestResetData["email"] = []string{prt.Email}
		_, err = client.PostForm(baseURL+"profile/request_reset", requestResetData)
		if err != nil {
			t.Fatalf("Error making http request: %v\n", err)
		}

		var userId string
		err = db.QueryRow("SELECT users.id FROM users WHERE users.email = ?", prt.Email).Scan(&userId)
		if err != nil {
			t.Fatalf("Error finding reset token: %v\n", err)
		}

		var resetToken string
		err = db.QueryRow("SELECT token FROM password_recovery WHERE password_recovery.user = ?", userId).Scan(&resetToken)
		if err != nil {
			t.Fatalf("Error finding reset token: %v\n", err)
		}
		if resetToken == "" {
			t.Fatalf("Incorrect reset token retrieved: %v\n", resetToken)
		}

		if prt.BadResetToken {
			resetToken += "123"
		}

		resetData := make(url.Values)
		resetData["user-id"] = []string{userId}
		resetData["reset-token"] = []string{resetToken}
		resetData["pass"] = []string{prt.NewPass}
		resp, err := client.PostForm(baseURL+"profile/reset/"+userId+"/"+resetToken, resetData)

		if err != nil {
			t.Fatalf("Error with reset request: %v\n", err)
		}

		switch {
		case prt.ExpectedStatusCode == http.StatusNoContent:
			if prt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("%v: Expected %v, got %v\n", prt.Last, prt.ExpectedStatusCode, resp.StatusCode)
			}
		case prt.ExpectedStatusCode == http.StatusBadRequest:
			if prt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("%v: Expected %v, got %v\n", prt.Last, prt.ExpectedStatusCode, resp.StatusCode)
			}

			dec := json.NewDecoder(resp.Body)
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Error parsing error: %v\n", err)
			}
			if errorValue.Reason != prt.ExpectedError {
				t.Fatalf("Expected %s, got %s\n", prt.ExpectedError, errorValue.Reason)
			}
		default:
			t.Fatalf("Something completely unexpected happened")
		}
	}
}
