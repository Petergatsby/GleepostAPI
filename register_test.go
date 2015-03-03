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
	time.Sleep(200 * time.Millisecond) //Time to spin up
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
	}
	testGood := registrationTest{
		Email:              "patrick@fakestanford.edu",
		Pass:               "TestingPass",
		First:              "Patrick",
		Last:               "Molgaard",
		ExpectedStatusCode: http.StatusCreated,
		ExpectedReturnType: "NewUser",
	}

	tests := []registrationTest{testGood}

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
			if created.Status != "unverified" {
				t.Fatalf("Status should be 'unverified', but is actually: %s\n", created.Status)
			}
		}
	}
}
