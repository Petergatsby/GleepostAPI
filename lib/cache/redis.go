package cache

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/garyburd/redigo/redis"
)

/********************************************************************
		General
********************************************************************/

//Cache represents a redis cache configuration + pool of connections to operate against.
type Cache struct {
	pool   *redis.Pool
	config conf.RedisConfig
}

//New constructs a new Cache from config.
func New(conf conf.RedisConfig) (cache *Cache) {
	cache = new(Cache)
	cache.config = conf
	cache.pool = redis.NewPool(GetDialer(conf), 100)
	return
}

//GetDialer enables dialing in a redis.Pool
func GetDialer(conf conf.RedisConfig) func() (redis.Conn, error) {
	f := func() (redis.Conn, error) {
		conn, err := redis.Dial(conf.Proto, conf.Address)
		return conn, err
	}
	return f
}

/********************************************************************
		Messages
********************************************************************/

//Publish takes a Message and publishes it to all participants (to be eventually consumed over websocket)
func (c *Cache) Publish(msg gp.Message, participants []gp.User, convID gp.ConversationID) {
	conn := c.pool.Get()
	defer conn.Close()
	JSONmsg, _ := json.Marshal(gp.RedisMessage{Message: msg, Conversation: convID})
	for _, user := range participants {
		conn.Send("PUBLISH", user.ID, JSONmsg)
	}
	conn.Flush()
}

//PublishEvent broadcasts an event of type etype with location "where" and a payload of data encoded as JSON to all of channels.
func (c *Cache) PublishEvent(etype string, where string, data interface{}, channels []string) {
	conn := c.pool.Get()
	defer conn.Close()
	event := gp.Event{Type: etype, Location: where, Data: data}
	JSONEvent, _ := json.Marshal(event)
	for _, channel := range channels {
		conn.Send("PUBLISH", channel, JSONEvent)
	}
	conn.Flush()
}

//Subscribe connects to userID's event channel and sends any messages over the messages chan.
//TODO: Delete Printf
func (c *Cache) Subscribe(messages chan []byte, userID gp.UserID) {
	conn := c.pool.Get()
	defer conn.Close()
	psc := redis.PubSubConn{Conn: conn}
	psc.Subscribe(userID)
	defer psc.Unsubscribe(userID)
	for {
		switch n := psc.Receive().(type) {
		case redis.Message:
			messages <- n.Data
		case redis.Subscription:
			fmt.Printf("%s: %s %d\n", n.Channel, n.Kind, n.Count)
		default:
			log.Printf("Other: %v", n)
		}
	}
}

//EventSubscribe subscribes to the channels in subscription, and returns them as a combined MsgQueue.
func (c *Cache) EventSubscribe(subscriptions []string) (events gp.MsgQueue) {
	commands := make(chan gp.QueueCommand)
	messages := make(chan []byte)
	events = gp.MsgQueue{Commands: commands, Messages: messages}
	conn := c.pool.Get()
	psc := redis.PubSubConn{Conn: conn}
	for _, s := range subscriptions {
		psc.Subscribe(s)
	}
	go controller(&psc, events.Commands)
	go messageReceiver(&psc, events.Messages)
	log.Println("New websocket connection created.")
	return events
}

func messageReceiver(psc *redis.PubSubConn, messages chan<- []byte) {
	for {
		switch n := psc.Receive().(type) {
		case redis.Message:
			messages <- n.Data
		case redis.Subscription:
			if n.Count == 0 {
				log.Println("Websocket client has unsubscribed from everything; closing connection.")
				close(messages)
				psc.Conn.Close()
				return
			}
		case error:
			log.Println("Saw an error: ", n)
			close(messages)
			return
		}
	}
}

func controller(psc *redis.PubSubConn, commands <-chan gp.QueueCommand) {
	for {
		command, ok := <-commands
		if !ok {
			return
		}
		log.Println("Got a command: ", command)
		switch {
		case command.Command == "UNSUBSCRIBE":
			channels := make([]interface{}, len(command.Value))
			for i, v := range command.Value {
				channels[i] = interface{}(v)
			}
			psc.Unsubscribe(channels...)
			if len(channels) == 0 {
				return
			}
		case command.Command == "SUBSCRIBE":
			channels := make([]interface{}, len(command.Value))
			for i, v := range command.Value {
				channels[i] = interface{}(v)
			}
			psc.Subscribe(channels...)
		}
	}
}
