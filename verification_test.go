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
		ExpectedStatusCode int
	}
	testGood := verificationTest{
		Email:              "verification_test1@fakestanford.edu",
		Pass:               "TestingPass",
		First:              "Verification",
		Last:               "Test1",
		ExpectedStatusCode: http.StatusCreated,
	}
	tests := []verificationTest{testGood}
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
		if resp.StatusCode != vt.ExpectedStatusCode {
			t.Fatalf("Wrong status code: Got %v but was expecting %v", resp.StatusCode, vt.ExpectedStatusCode)
		}

		var token string = ""
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

		switch {
		case vt.ExpectedStatusCode == http.StatusNoContent:
			if vt.ExpectedStatusCode != resp.StatusCode {
				t.Fatalf("Expected %v, got %v\n", vt.ExpectedStatusCode, resp.StatusCode)
			}
		}
	}
}
