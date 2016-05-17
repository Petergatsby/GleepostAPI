package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/Petergatsby/GleepostAPI/lib/conf"
	"github.com/Petergatsby/GleepostAPI/lib/gp"
)

func TestMuteConversation(t *testing.T) {
	initConvs()
	once.Do(setup)
	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatalf("Error logging in: %v", err)
	}
	type muteTest struct {
		Token          gp.Token
		ConversationID gp.ConversationID
		Mute           bool
		ExpectedStatus int
		ExpectedType   string
		ExpectedError  string
	}
	tests := []muteTest{
		{ //Nonexistent conversation
			Token:          token,
			ConversationID: 9999,
			Mute:           true,
			ExpectedStatus: 403,
			ExpectedType:   "gp.APIerror",
			ExpectedError:  "You're not allowed to do that!",
		},
		{ //Conversation you don't participate in
			Token:          token,
			ConversationID: 2,
			Mute:           true,
			ExpectedStatus: 403,
			ExpectedType:   "gp.APIerror",
			ExpectedError:  "You're not allowed to do that!",
		},
		{ //not muted -> not muted
			Token:          token,
			ConversationID: 1,
			Mute:           false,
			ExpectedStatus: 200,
			ExpectedType:   "gp.ConversationAndMessages",
		},
		{ //not muted -> muted
			Token:          token,
			ConversationID: 1,
			Mute:           true,
			ExpectedStatus: 200,
			ExpectedType:   "gp.ConversationAndMessages",
		},
		{ //muted -> muted
			Token:          token,
			ConversationID: 1,
			Mute:           true,
			ExpectedStatus: 200,
			ExpectedType:   "gp.ConversationAndMessages",
		},
		{ //muted -> not muted
			Token:          token,
			ConversationID: 1,
			Mute:           false,
			ExpectedStatus: 200,
			ExpectedType:   "gp.ConversationAndMessages",
		},
	}
	for _, test := range tests {
		data := make(url.Values)
		data["id"] = []string{fmt.Sprintf("%d", test.Token.UserID)}
		data["token"] = []string{test.Token.Token}
		data["muted"] = []string{strconv.FormatBool(test.Mute)}

		req, err := http.NewRequest("PUT", fmt.Sprintf("%sconversations/%d", baseURL, test.ConversationID), strings.NewReader(data.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		if err != nil {
			t.Fatal("Couldn't make request:", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal("Couldn't make request:", err)
		}
		if resp.StatusCode != test.ExpectedStatus {
			t.Fatalf("Got incorrect status code: expected %d but got %d.\n", test.ExpectedStatus, resp.StatusCode)
		}
		switch {
		case test.ExpectedType == "gp.APIerror":
			var errResp gp.APIerror
			dec := json.NewDecoder(resp.Body)
			dec.Decode(&errResp)
			if errResp.Reason != test.ExpectedError {
				t.Fatalf("Got incorrect error message: expected %s but got %s.\n", test.ExpectedError, errResp.Reason)
			}
		case test.ExpectedType == "gp.ConversationAndMessages":
			var conv gp.ConversationAndMessages
			dec := json.NewDecoder(resp.Body)
			err = dec.Decode(&conv)
			if err != nil {
				t.Fatal("Error decoding conversation json:", err)
			}
			if conv.Muted != test.Mute {
				t.Fatalf("Test set mute = %t but conversation had mute = %t\n", test.Mute, conv.Muted)
			}
		}
	}
}

func initConvs() error {
	err := initDB()
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", conf.GetConfig().Mysql.ConnectionString())
	if err != nil {
		return err
	}
	defer db.Close()

	truncate("conversations", "conversation_participants")

	_, err = db.Exec("INSERT INTO `conversations` (initiator, primary_conversation) VALUES (1, 1), (1, 1)")
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO `conversation_participants` (conversation_id, participant_id) VALUES (1, 1), (1, 2), (2, 2)")
	if err != nil {
		return err
	}
	return nil
}
