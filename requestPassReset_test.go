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

func TestRequestPassReset(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	db, err := sql.Open("mysql", conf.GetConfig().Mysql.ConnectionString())
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	client := &http.Client{}

	type requestPassResetTest struct {
		Email              string
		Pass               string
		First              string
		Last               string
		RegisterAccount    bool
		VerifyAccount      bool
		ExpectedStatusCode int
		ExpectedError      string
	}
	testGood := requestPassResetTest{
		Email:              "request_pass_reset_test1@fakestanford.edu",
		Pass:               "TestingPass",
		First:              "Resetpass",
		Last:               "Test1",
		RegisterAccount:    true,
		VerifyAccount:      true,
		ExpectedStatusCode: http.StatusNoContent,
	}
	testBad := requestPassResetTest{
		Email:              "request_pass_reset_test2@fakestanford.edu",
		RegisterAccount:    false,
		VerifyAccount:      false,
		ExpectedStatusCode: http.StatusBadRequest,
		ExpectedError:      "sql: no rows in result set",
	}
	testMixedCap := requestPassResetTest{
		Email:              "reQUeSt_pASS_ResET_teSt1@fakesTanFORD.eDu",
		RegisterAccount:    false,
		VerifyAccount:      false,
		ExpectedStatusCode: http.StatusNoContent,
	}
	testUnverified := requestPassResetTest{
		Email:              "request_pass_reset_test3@fakestanford.edu",
		Pass:               "TestingPass",
		First:              "Resetpass",
		Last:               "Test3",
		RegisterAccount:    true,
		VerifyAccount:      false,
		ExpectedStatusCode: http.StatusNoContent,
	}
	tests := []requestPassResetTest{testGood, testBad, testMixedCap, testUnverified}

	for _, rprt := range tests {

		if rprt.RegisterAccount {
			data := make(url.Values)
			data["email"] = []string{rprt.Email}
			data["pass"] = []string{rprt.Pass}
			data["first"] = []string{rprt.First}
			data["last"] = []string{rprt.Last}
			_, err := client.PostForm(baseURL+"register", data)

			if err != nil {
				t.Fatalf("Error making http request: %v\n", err)
			}
		}

		if rprt.VerifyAccount {
			var token string
			err = db.QueryRow("SELECT token FROM verification JOIN users ON users.id = verification.user_id WHERE users.email = ?", rprt.Email).Scan(&token)

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

		resetData := make(url.Values)
		resetData["email"] = []string{rprt.Email}
		resp, err := client.PostForm(baseURL+"profile/request_reset", resetData)

		switch {
		case rprt.ExpectedStatusCode == http.StatusOK:
			if rprt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Expected %v, got %v\n", rprt.ExpectedStatusCode, resp.StatusCode)
			}
		case rprt.ExpectedStatusCode == http.StatusBadRequest:
			if rprt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Expected %v, got %v\n", rprt.ExpectedStatusCode, resp.StatusCode)
			}

			dec := json.NewDecoder(resp.Body)
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Error parsing error: %v\n", err)
			}
			if errorValue.Reason != rprt.ExpectedError {
				t.Fatalf("Expected %s, got %s\n", rprt.ExpectedError, errorValue.Reason)
			}
		}
	}
}
