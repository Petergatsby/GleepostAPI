package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func TestFileHistory(t *testing.T) {
	once.Do(setup)

	truncate("conversations", "conversation_participants", "chat_messages")

	err := initDB()
	if err != nil {
		t.Fatal(err)
	}

	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatal("Error logging in:", err)
	}

	conv, err := createConversation(token)
	if err != nil {
		t.Fatal("Error creating conversation:", err)
	}
	msgID, err := sendMessage(token, conv.ID, "hey here's a file: <https://file.host|pdf>")
	if err != nil {
		t.Fatal("Error sending file:", err)
	}
	log.Println(msgID)
	resp, err := client.Get(fmt.Sprintf("%s%s/%d/files", baseURL, "conversations", conv.ID))
	if err != nil {
		t.Fatal("Error getting files list:", err)
	}
	if resp.StatusCode != 200 {
		t.Fatal("Expected status 200 but got:", resp.StatusCode)

	}

	//todo: check /files for this

}

func createConversation(token gp.Token) (conv gp.ConversationAndMessages, err error) {
	data := make(url.Values)
	data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
	data["token"] = []string{token.Token}
	data["participants"] = []string{"1,2"}
	req, _ := http.NewRequest("POST", baseURL+"conversations", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	dec := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		errResp := gp.APIerror{}
		log.Println(resp.Status)
		err = dec.Decode(&errResp)
		log.Println(errResp)
		return

	}
	err = dec.Decode(&conv)
	return
}

func sendMessage(token gp.Token, conv gp.ConversationID, msg string) (id gp.MessageID, err error) {
	data := make(url.Values)
	data["id"] = []string{fmt.Sprintf("%d", token.UserID)}
	data["token"] = []string{token.Token}
	data["text"] = []string{msg}
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s%s/%d/messages", baseURL, "conversations", conv), strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	dec := json.NewDecoder(resp.Body)
	created := gp.Created{}
	err = dec.Decode(&created)
	if err != nil {
		return
	}
	id = gp.MessageID(created.ID)

	return
}
