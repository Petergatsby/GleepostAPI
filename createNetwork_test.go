package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"testing"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func TestCreateNetwork(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}
	err = initAdmin()
	if err != nil {
		t.Fatalf("Error initializing admin status: %v\n", err)
	}

	client := &http.Client{}

	type netCreationTest struct {
		Email              string //To get a session
		Pass               string //To get a session
		Name               string //Name of new network
		University         bool   //Is this a university network?
		Domains            string //Domains a user can register against
		ExpectedStatusCode int
		ExpectedType       string
		ExpectedError      string
	}
	testAdmin := netCreationTest{
		Email:              "patrick@fakestanford.edu",
		Pass:               "TestingPass",
		Name:               "University of Leeds",
		University:         true,
		Domains:            "leeds.ac.uk",
		ExpectedStatusCode: http.StatusCreated,
		ExpectedType:       "Network",
	}
	tests := []netCreationTest{testAdmin}
	for _, uct := range tests {
		token, err := testingGetSession(uct.Email, uct.Pass)
		if err != nil {
			t.Fatal("Error logging in:", err)
		}
		data := make(url.Values)
		data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
		data["token"] = []string{token.Token}
		data["university"] = []string{fmt.Sprintf("%t", uct.University)}
		data["name"] = []string{uct.Name}
		resp, err := client.PostForm(baseURL+"networks", data)
		if err != nil {
			t.Fatal("Error making request:", err)
		}
		if resp.StatusCode != uct.ExpectedStatusCode {
			t.Fatalf("Expected status code %d, got %d\n", uct.ExpectedStatusCode, resp.StatusCode)
		}
		dec := json.NewDecoder(resp.Body)
		switch {
		case uct.ExpectedType == "Network":
			network := gp.Network{}
			err = dec.Decode(&network)
			log.Println(network)
			if err != nil {
				t.Fatalf("Failed to decode as %s: %v\n", uct.ExpectedType, err)
			}
			if network.ID < 1 {
				t.Fatalf("Network.ID must be nonzero (%d)\n", network.ID)
			}
			if network.Name != uct.Name {
				t.Fatalf("Network name was not as expected: %s vs %s\n", network.Name, uct.Name)
			}
		default:
		}
	}
}

func initAdmin() error {
	config := conf.GetConfig()
	db, err := sql.Open("mysql", config.Mysql.ConnectionString())
	if err != nil {
		return err
	}
	_, err = db.Exec("UPDATE users SET is_admin = 1 WHERE email = 'patrick@fakestanford.edu'")
	return err
}
