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

func TestRequestPassReset(t *testing.T) {
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
	defer server.Close()
	baseURL = server.URL + "/api/v1/"

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
		ExpectedError:      "That user does not exist.",
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

	for testNumber, rprt := range tests {

		if rprt.RegisterAccount {
			data := make(url.Values)
			data["email"] = []string{rprt.Email}
			data["pass"] = []string{rprt.Pass}
			data["first"] = []string{rprt.First}
			data["last"] = []string{rprt.Last}
			_, err := client.PostForm(baseURL+"register", data)

			if err != nil {
				t.Fatalf("Test%v: Error making http request: %v\n", testNumber, err)
			}
		}

		if rprt.VerifyAccount {
			var token string
			err = db.QueryRow("SELECT token FROM verification JOIN users ON users.id = verification.user_id WHERE users.email = ?", rprt.Email).Scan(&token)

			if err != nil {
				t.Fatalf("Test%v: Error finding token: %v\n", testNumber, err)
			}
			if token == "" {
				t.Fatalf("Test%v: Incorrect token retrieved: %v\n", testNumber, token)
			}

			_, err = client.PostForm(baseURL+"verify/"+token, make(url.Values))

			if err != nil {
				t.Fatalf("Test%v: Error with verification request: %v\n", testNumber, err)
			}
		}

		resetData := make(url.Values)
		resetData["email"] = []string{rprt.Email}
		resp, err := client.PostForm(baseURL+"profile/request_reset", resetData)

		switch {
		case rprt.ExpectedStatusCode == http.StatusNoContent:
			if rprt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Test%v: Expected %v, got %v\n", testNumber, rprt.ExpectedStatusCode, resp.StatusCode)
			}
		case rprt.ExpectedStatusCode == http.StatusBadRequest:
			if rprt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Test%v: Expected %v, got %v\n", testNumber, rprt.ExpectedStatusCode, resp.StatusCode)
			}

			dec := json.NewDecoder(resp.Body)
			errorValue := gp.APIerror{}
			err = dec.Decode(&errorValue)
			if err != nil {
				t.Fatalf("Test%v: Error parsing error: %v\n", testNumber, err)
			}
			if errorValue.Reason != rprt.ExpectedError {
				t.Fatalf("Test%v: Expected %s, got %s\n", testNumber, rprt.ExpectedError, errorValue.Reason)
			}
		default:
			t.Fatalf("Test%v: Something completely unexpected happened", testNumber)
		}
	}
}
