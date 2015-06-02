package main

import (
	"fmt"
	"log"
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
)

func TestSendPresence(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatal("Error initializing db:", err)
	}
	once.Do(setup)
	token, err := testingGetSession("patrick@fakestanford.edu", "TestingPass")
	if err != nil {
		t.Fatal("Error getting session:", err)
	}
	header := make(http.Header)
	header.Set("X-GP-Auth", fmt.Sprintf("%d-%s", token.UserID, token.Token))
	ws, resp, err := websocket.DefaultDialer.Dial("ws"+baseURL[4:]+"ws", header)
	if err != nil {
		t.Fatal("Couldn't acquire wss connection:", err)
	}
	defer ws.Close()
	if resp.StatusCode != 101 {
		t.Fatal("Didn't get", http.StatusSwitchingProtocols)

	}
	action := action{Action: "presence", Form: "desktop"}
	err = ws.WriteJSON(action)
	if err != nil {
		log.Println("Error writing status to ws:", err)
	}
	//check own presence
}
