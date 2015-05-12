package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/mail"
)

func TestGetNotifications(t *testing.T) {
	config := conf.GetConfig()
	api = lib.New(*config)
	api.Mail = mail.NewMock()
	api.Start()
	server := httptest.NewServer(r)
	defer server.Close()
	baseURL = server.URL + "/api/v1/"

	truncate("notifications")

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatalf("Error logging in: %v", err)
	}

	respValue, err := getNotifications(token, "false")
	if err != nil {
		t.Fatalf("Error getting notifications: %v", err)
	}

	if len(respValue) != 0 {
		t.Fatalf("Got %v notifications when expected 0", len(respValue))
	}

	createNotification("commented", token.UserID, token.UserID, 1, 1, "1")
	createNotification("commented", token.UserID, token.UserID, 1, 1, "2")
	createNotification("commented", token.UserID, token.UserID, 1, 1, "3")
	createNotification("commented", token.UserID, token.UserID, 1, 1, "4")

	respValue, err = getNotifications(token, "false")
	if err != nil {
		t.Fatalf("Error getting notifications: %v", err)
	}

	if len(respValue) != 4 {
		t.Fatalf("Got %v notifications when expected 4", len(respValue))
	} else if checkNotificationValidity(respValue) == false {
		t.Fatalf("Notifications are incorrect", len(respValue))
	}

	err = readNotifications(token, 2)
	if err != nil {
		t.Fatalf("Error reading notifications: %v", err)
	}

	respValue, err = getNotifications(token, "false")
	if err != nil {
		t.Fatalf("Error getting notifications: %v", err)
	}

	if len(respValue) != 2 {
		t.Fatalf("Got %v notifications when expected 2", len(respValue))
	} else if checkNotificationValidity(respValue) == false {
		t.Fatalf("Notifications are incorrect", len(respValue))
	}

	respValue, err = getNotifications(token, "true")
	if err != nil {
		t.Fatalf("Error getting notifications: %v", err)
	}

	if len(respValue) != 4 {
		t.Fatalf("Got %v notifications when expected 4", len(respValue))
	} else if checkNotificationValidity(respValue) == false {
		t.Fatalf("Notifications are incorrect", len(respValue))
	}
}

func getNotifications(token gp.Token, include_seen string) ([]gp.Notification, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", baseURL+"notifications?include_seen="+include_seen, nil)
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

func createNotification(ntype string, by gp.UserID, recipient gp.UserID, postID gp.PostID, netID gp.NetworkID, preview string) error {
	db, err := sql.Open("mysql", config.Mysql.ConnectionString())
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("INSERT INTO notifications (type, time, `by`, recipient, post_id, network_id, preview_text) VALUES (?, NOW(), ?, ?, ?, ?, ?)", ntype, by, recipient, postID, netID, preview)
	if err != nil {
		return err
	}
	return nil
}

func readNotifications(token gp.Token, seen int) error {
	client := &http.Client{}

	req, err := http.NewRequest("PUT", baseURL+"notifications?seen="+fmt.Sprintf("%d", seen), nil)
	req.Header.Set("X-GP-Auth", fmt.Sprintf("%d", token.UserID)+"-"+token.Token)
	if err != nil {
		return err
	}

	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}

func checkNotificationValidity(notifications []gp.Notification) bool {
	for _, notification := range notifications {
		if notification.ID <= 0 || notification.By.ID <= 0 || len(notification.Type) <= 0 {
			return false
		}
	}
	return true
}
