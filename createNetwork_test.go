package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func initCreateNetwork() error {
	err := initDB()
	if err != nil {
		return err
	}
	err = initAdmin()
	if err != nil {
		return err
	}
	return nil
}

func TestCreateNetwork(t *testing.T) {
	err := initCreateNetwork()
	if err != nil {
		t.Fatalf("Error initializing test state: %v\n", err)
	}

	config := conf.GetConfig()
	api = lib.New(*config)
	server := httptest.NewServer(r)
	defer server.Close()
	baseURL = server.URL + "/api/v1/"

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
	testNonAdmin := netCreationTest{
		Email:              "bonnie@fakestanford.edu",
		Pass:               "TestingPass",
		Name:               "University of I'm 3 yo, I just wanna watch Octonauts.",
		University:         true,
		Domains:            "gremlin.ac.uk",
		ExpectedStatusCode: http.StatusForbidden,
		ExpectedType:       "Error",
		ExpectedError:      "You're not allowed to do that!",
	}
	testGroup := netCreationTest{
		Email:              "patrick@fakestanford.edu",
		Pass:               "TestingPass",
		Name:               "Cool Group",
		University:         false,
		ExpectedStatusCode: http.StatusCreated,
		ExpectedType:       "Group",
	}

	tests := []netCreationTest{testAdmin, testNonAdmin, testGroup}
	for testNumber, nct := range tests {
		token, err := testingGetSession(nct.Email, nct.Pass)
		if err != nil {
			t.Fatalf("Test%v: Error logging in: %s\n", testNumber, err)
		}
		data := make(url.Values)
		data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
		data["token"] = []string{token.Token}
		data["university"] = []string{fmt.Sprintf("%t", nct.University)}
		data["name"] = []string{nct.Name}
		req, _ := http.NewRequest("POST", baseURL+"networks", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Close = true

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Test%v: Error making request: %s\n", testNumber, err)
		}
		if resp.StatusCode != nct.ExpectedStatusCode {
			t.Fatalf("Test%v: Expected status code %d, got %d\n", testNumber, nct.ExpectedStatusCode, resp.StatusCode)
		}
		dec := json.NewDecoder(resp.Body)
		switch {
		case nct.ExpectedType == "Network":
			network := gp.Network{}
			err = dec.Decode(&network)
			if err != nil {
				t.Fatalf("Test%v: Failed to decode as %s: %v\n", testNumber, nct.ExpectedType, err)
			}
			if network.ID < 1 {
				t.Fatalf("Test%v: Network.ID must be nonzero (%d)\n", testNumber, network.ID)
			}
			if network.Name != nct.Name {
				t.Fatalf("Test%v: Network name was not as expected: %s vs %s\n", testNumber, network.Name, nct.Name)
			}
		case nct.ExpectedType == "Group":
			group := gp.Group{}
			err = dec.Decode(&group)
			if err != nil {
				t.Fatalf("Test%v: Failed to decode as %s: %v\n", testNumber, nct.ExpectedType, err)
			}
			if group.ID < 1 {
				t.Fatalf("Test%v: Group.ID must be nonzero (%d)\n", testNumber, group.ID)
			}
			if group.Name != nct.Name {
				t.Fatalf("Test%v: Group name was not as expected: %s vs %s\n", testNumber, group.Name, nct.Name)
			}
		case nct.ExpectedType == "Error":
			errorResp := gp.APIerror{}
			err = dec.Decode(&errorResp)
			if err != nil {
				t.Fatalf("Test%v: Failed to decode as %s: %v\n", testNumber, nct.ExpectedType, err)
			}
			if errorResp.Reason != nct.ExpectedError {
				t.Fatalf("Test%v: Wrong error: Expected %s but got %s\n", testNumber, nct.ExpectedError, errorResp.Reason)
			}
		default:
			t.Fatalf("Test%v: Something completely unexpected happened", testNumber)
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
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO `users` (`password`, `email`, `verified`, `firstname`, `lastname`) VALUES ('$2a$10$xLUmQbvrHAAOGuv4.uHAY.NmoLGEuEObENPiQ8kkh.Miyvdzhyge6', 'bonnie@fakestanford.edu', 1, 'Bonnie', 'Molgaard')")
	return err
}
