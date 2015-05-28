package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func TestNotificationPagination(t *testing.T) {
	once.Do(setup)

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatalf("Error logging in: %v", err)
	}

	err = initManyNotifications(100)
	if err != nil {
		t.Fatalf("Error intialising notificationss: %v", err)
	}

	type notificationPaginationTest struct {
		Command                   string
		ExpectedPosts             int
		ExpectedStartingPostIndex int
		ExpectedEndingPostIndex   int
	}
	beforeTest := notificationPaginationTest{
		Command:                   "?before=50",
		ExpectedPosts:             20,
		ExpectedStartingPostIndex: 49,
		ExpectedEndingPostIndex:   30,
	}
	afterTest := notificationPaginationTest{
		Command:                   "?after=25",
		ExpectedPosts:             20,
		ExpectedStartingPostIndex: 45,
		ExpectedEndingPostIndex:   26,
	}
	beforeEmptyTest := notificationPaginationTest{
		Command:                   "?before=1",
		ExpectedPosts:             0,
		ExpectedStartingPostIndex: 0,
		ExpectedEndingPostIndex:   0,
	}
	afterEmptyTest := notificationPaginationTest{
		Command:                   "?after=100",
		ExpectedPosts:             0,
		ExpectedStartingPostIndex: 0,
		ExpectedEndingPostIndex:   0,
	}
	emptyTest := notificationPaginationTest{
		Command:                   "",
		ExpectedPosts:             20,
		ExpectedStartingPostIndex: 100,
		ExpectedEndingPostIndex:   81,
	}
	tests := []notificationPaginationTest{beforeTest, afterTest, beforeEmptyTest, afterEmptyTest, emptyTest}
	for testNumber, npt := range tests {
		respValue, err := getPaginatedNotifications(token, npt.Command)
		if err != nil {
			t.Fatalf("Test%v: Failed to get paginated notifications", err)
		}

		if len(respValue) != npt.ExpectedPosts {
			t.Fatalf("Test%v: Incorrect number of responses, expected %v, got %v", testNumber, npt.ExpectedPosts, len(respValue))
		}

		if len(respValue) > 0 {
			if !checkNotificationsDescending(respValue) {
				t.Fatalf("Test%v: Posts are not in order", testNumber)
			}

			if respValue[0].ID != gp.NotificationID(npt.ExpectedStartingPostIndex) {
				t.Fatalf("Test%v: Incorrect starting notification number. Expected %v, got %v", testNumber, npt.ExpectedStartingPostIndex, respValue[0].ID)
			}

			if respValue[npt.ExpectedPosts-1].ID != gp.NotificationID(npt.ExpectedEndingPostIndex) {
				t.Fatalf("Test%v: Incorrect ending notificaition number. Expected %v, got %v", testNumber, npt.ExpectedEndingPostIndex, respValue[npt.ExpectedPosts-1].ID)
			}
		}
	}
}

func getPaginatedNotifications(token gp.Token, command string) ([]gp.Notification, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", baseURL+"notifications"+command, nil)
	req.Header.Set("X-GP-Auth", fmt.Sprintf("%d", token.UserID)+"-"+token.Token)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(resp.Body)
	respValue := []gp.Notification{}
	err = dec.Decode(&respValue)
	if err != nil {
		return nil, err
	}
	return respValue, nil
}

func checkNotificationsDescending(notifications []gp.Notification) bool {
	lastID := notifications[0].ID + 1
	for _, notificaiton := range notifications {
		if lastID != notificaiton.ID+1 {
			return false
		}
		lastID = notificaiton.ID
	}
	return true
}

func initManyNotifications(tests int) error {
	err := initDB()
	if err != nil {
		return err
	}

	truncate("notifications")

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", config.Mysql.ConnectionString())
	if err != nil {
		return err
	}
	defer db.Close()

	for i := 0; i < tests; i++ {
		_, err = db.Exec("INSERT INTO notifications (type, time, `by`, recipient, post_id, network_id, preview_text) VALUES (?, NOW(), ?, ?, ?, ?, ?)", "commented", token.UserID, token.UserID, 1, 1, i)
		if err != nil {
			return err
		}
	}

	return nil
}
