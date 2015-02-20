package main

import (
	"log"
	"net/http"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/cache"
	"github.com/draaglom/GleepostAPI/lib/gp"
	gwebsocket "github.com/gorilla/websocket"
)

func init() {
	base.Handle("/ws", websocket.Handler(jsonServer))
	base.HandleFunc("/ws2", gorillaWS)
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
	go websocketReader(ws, events, userID)
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
			events.Commands <- gp.QueueCommand{Command: "UNSUBSCRIBE", Value: []string{}}
			close(events.Commands)
			return
		}
		log.Println("Sent bytes: ", n)
	}
}

func websocketReader(ws *websocket.Conn, events gp.MsgQueue, userID gp.UserID) {
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
			chans = append(chans, cache.PostViewChannel(i))
		}
		log.Println(c)
		log.Println(chans)

		events.Commands <- gp.QueueCommand{Command: c.Action, Value: chans}
	}
}

type postSubscriptionAction struct {
	Action   string `json:"action"`
	Channels []int  `json:"posts"`
}

var upgrader = gwebsocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func gorillaWS(w http.ResponseWriter, r *http.Request) {
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
	chans := lib.ConversationChannelKeys([]gp.User{{ID: userID}})
	chans = append(chans, lib.NotificationChannelKey(userID))
	events := api.EventSubscribe(chans)
	go gWebsocketReader(conn, events, userID)
	heartbeat := time.Tick(30 * time.Second)
	for {
		select {
		case message, ok := <-events.Messages:
			if !ok {
				log.Println("Message channel is closed...")
				return
			}
			err := conn.WriteMessage(gwebsocket.TextMessage, message)
			if err != nil {
				log.Println("Saw an error: ", err)
				events.Commands <- gp.QueueCommand{Command: "UNSUBSCRIBE", Value: []string{}}
				close(events.Commands)
				return
			}
		case <-heartbeat:
			err := conn.WriteControl(gwebsocket.PingMessage, []byte("hello"), time.Now().Add(1*time.Second))
			if err != nil {
				log.Println("Saw an error pinging: ", err)
				events.Commands <- gp.QueueCommand{Command: "UNSUBSCRIBE", Value: []string{}}
				close(events.Commands)
				return
			}
		}
	}
}

func gWebsocketReader(ws *gwebsocket.Conn, events gp.MsgQueue, userID gp.UserID) {
	var c postSubscriptionAction
	for {
		if ws == nil {
			return
		}
		err := ws.ReadJSON(&c)
		if err != nil {
			ws.Close()
			log.Println(err)
			return
		}
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
			chans = append(chans, cache.PostViewChannel(i))
		}
		log.Println(c)
		log.Println(chans)

		events.Commands <- gp.QueueCommand{Command: c.Action, Value: chans}
	}
}
