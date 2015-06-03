package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/gorilla/websocket"
)

func init() {
	base.HandleFunc("/ws", wsHandler)
}

type action struct {
	Action       string            `json:"action"`
	Channels     []int             `json:"posts"`
	Form         string            `json:"form"`
	Conversation gp.ConversationID `json:"conversation"`
	Typing       bool              `json:"typing"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	userID, err := authenticate(r)
	if err != nil {
		e := conn.WriteJSON(err)
		if e != nil {
			log.Println(e)
		}
		return
	}
	chans := lib.ConversationChannelKeys([]gp.UserPresence{{User: gp.User{ID: userID}}})
	chans = append(chans, lib.NotificationChannelKey(userID))
	events := api.EventSubscribe(chans)
	go wsReader(conn, events, userID)
	heartbeat := time.Tick(30 * time.Second)
	for {
		select {
		case message, ok := <-events.Messages:
			if !ok {
				log.Println("Message channel is closed...")
				return
			}
			err := conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				if err != websocket.ErrCloseSent {
					log.Println("Saw an error: ", err)
				}
				events.Commands <- gp.QueueCommand{Command: "UNSUBSCRIBE", Value: []string{}}
				close(events.Commands)
				return
			}
		case <-heartbeat:
			err := conn.WriteControl(websocket.PingMessage, []byte("hello"), time.Now().Add(1*time.Second))
			if err != nil {
				if err != websocket.ErrCloseSent {
					log.Println("Saw an error pinging: ", err)
				}
				events.Commands <- gp.QueueCommand{Command: "UNSUBSCRIBE", Value: []string{}}
				close(events.Commands)
				return
			}
		}
	}
}

func wsReader(ws *websocket.Conn, messages gp.MsgQueue, userID gp.UserID) {
	var c action
	for {
		if ws == nil {
			return
		}
		err := ws.ReadJSON(&c)
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading from websocket:", err)
			}
			ws.Close()
			return
		}
		switch {
		case c.Action == "presence":
			err := api.Presences.Broadcast(userID, c.Form)
			if err != nil {
				log.Println("Error broadcasting presence:", err)
			}
		case c.Action == "typing":
			api.UserIsTyping(userID, c.Conversation, c.Typing)
		default:
			var postChans []gp.PostID
			for _, i := range c.Channels {
				postChans = append(postChans, gp.PostID(i))
			}
			postChans, err = api.CanSubscribePosts(userID, postChans)
			if err != nil {
				log.Println(err)
				continue
			}
			var chans []string
			for _, i := range postChans {
				chans = append(chans, lib.PostChannel(i))
			}

			messages.Commands <- gp.QueueCommand{Command: c.Action, Value: chans}
		}
	}
}
