package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
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
	createConversation(token)
	header := make(http.Header)
	header.Set("X-GP-Auth", fmt.Sprintf("%d-%s", token.UserID, token.Token))
	ws, resp, err := websocket.DefaultDialer.Dial("ws"+baseURL[4:]+"ws", header)
	if err != nil {
		t.Fatal("Couldn't acquire wss connection:", err)
	}
	defer ws.Close()
	defer ws.WriteControl(websocket.CloseMessage, []byte("bye"), time.Now().Add(1*time.Second))
	if resp.StatusCode != 101 {
		t.Fatal("Didn't get", http.StatusSwitchingProtocols)
	}
	action := action{Action: "presence", Form: "desktop"}
	message := wrappedAction{Data: action}
	err = ws.WriteJSON(message)
	if err != nil {
		t.Fatal("Error writing status to ws:", err)
	}
	evt := gp.WrappedEvent{}
	err = ws.ReadJSON(&evt)
	if err != nil {
		t.Fatal("Couldn't read from websocket:", err)
	}
	if evt.Data.Type != "presence" {
		t.Fatal("Expected `presence` but got:", evt.Data.Type)
	}
	if evt.Data.Location != fmt.Sprintf("/user/%d", token.UserID) {
		t.Fatal("Unexpected location, got:", evt.Data.Location)
	}
}
