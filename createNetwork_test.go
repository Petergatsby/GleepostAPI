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
		case uct.ExpectedType == "Group":
			group := gp.Group{}
			err = dec.Decode(&group)
			if err != nil {
				t.Fatalf("Failed to decode as %s: %v\n", uct.ExpectedType, err)
			}
			if group.ID < 1 {
				t.Fatalf("Group.ID must be nonzero (%d)\n", group.ID)
			}
			if group.Name != uct.Name {
				t.Fatalf("Group name was not as expected: %s vs %s\n", group.Name, uct.Name)
			}
		case uct.ExpectedType == "Error":
			errorResp := gp.APIerror{}
			err = dec.Decode(&errorResp)
			if err != nil {
				t.Fatalf("Failed to decode as %s: %v\n", uct.ExpectedType, err)
			}
			if errorResp.Reason != uct.ExpectedError {
				t.Fatalf("Wrong error: Expected %s but got %s\n", uct.ExpectedError, errorResp.Reason)
			}
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
