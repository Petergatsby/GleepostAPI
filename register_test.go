package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/mail"
)

func init() {
	api.Mail = mail.NewMock()
	go main()
	time.Sleep(100 * time.Millisecond) //Time to spin up
}

func TestRegister(t *testing.T) {
	//Init
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

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
	tests := []registrationTest{testGood, testNoEmail}

	for _, r := range tests {
		data := make(url.Values)
		data["email"] = []string{r.Email}
		data["pass"] = []string{r.Pass}
		data["first"] = []string{r.First}
		data["last"] = []string{r.Last}
		resp, err := client.PostForm(baseUrl+"register", data)
		if err != nil {
			t.Fatalf("Error making http request: %v\n", err)
		}
		if resp.StatusCode != r.ExpectedStatusCode {
			t.Fatalf("Wrong status code: Got %v but was expecting %v", resp.StatusCode, r.ExpectedStatusCode)
		}
		dec := json.NewDecoder(resp.Body)
		switch {
		case r.ExpectedReturnType == "NewUser":
			created := gp.NewUser{}
			err = dec.Decode(&created)
			if err != nil {
				t.Fatalf("Error parsing registration response as %s: %v\n", r.ExpectedReturnType, err)
			}
			if created.Status != r.ExpectedRegStatus {
				t.Fatalf("Status should be %s, but is actually: %s\n", r.ExpectedRegStatus, created.Status)
			}
		case r.ExpectedReturnType == "Error":
			errorResp := gp.APIerror{}
			err = dec.Decode(&errorResp)
			if err != nil {
				t.Fatalf("Error parsing registration response as %s: %v\n", r.ExpectedReturnType, err)
			}
			if errorResp.Reason != r.ExpectedError {
				t.Fatalf("Saw error: %s, was expecting: %s\n", errorResp.Reason, r.ExpectedError)
			}
		}
	}
}
