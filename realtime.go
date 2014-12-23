package main

import (
	"log"
	"net/http"

	"code.google.com/p/go.net/websocket"
	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func init() {
	base.HandleFunc("/longpoll", longPollHandler)
	base.Handle("/ws", websocket.Handler(jsonServer))
}

func longPollHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, err := authenticate(r)
	switch {
	case err != nil:
		jsonResponse(w, &EBADTOKEN, 400)
	case r.Method != "GET":
		jsonResponse(w, &EUNSUPPORTED, 405)
	default:
		//awaitOneMessage will block until a message arrives over redis
		message := api.AwaitOneMessage(userID)
		w.Write(message)
	}
}

func jsonServer(ws *websocket.Conn) {
	r := ws.Request()
	defer ws.Close()
	userID, err := authenticate(r)
	if err != nil {
		ws.Write([]byte(err.Error()))
		return
	}
	//Change this. 12/12/13
	chans := lib.ConversationChannelKeys([]gp.User{{ID: userID}})
	chans = append(chans, lib.NotificationChannelKey(userID))
	events := api.EventSubscribe(chans)
	go websocketReader(ws, events)
	for {
		message, ok := <-events.Messages
		if !ok {
			log.Println("Message channel is closed...")
			ws.Close()
			return
		}
		n, err := ws.Write(message)
		if err != nil {
			log.Println("Saw an error: ", err)
			events.Commands <- gp.QueueCommand{Command: "UNSUBSCRIBE", Value: ""}
			close(events.Commands)
			return
		}
		log.Println("Sent bytes: ", n)
	}
}

func websocketReader(ws *websocket.Conn, events gp.MsgQueue) {
	var c postSubscriptionAction
	for {
		if ws == nil {
			return
		}
		err := websocket.JSON.Receive(ws, &c)
		if err != nil {
			log.Println(err)
			return
		}
		//TODO: Check you're actually allowed to see these.
		var chans string
		for _, i := range c.Channels {
			chans += " " + string(i)
		}
		events.Commands <- gp.QueueCommand{Command: c.Action, Value: chans}
	}
}

type postSubscriptionAction struct {
	Action   string `json:"action"`
	Channels []int  `json:"posts"`
}
