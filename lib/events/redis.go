package events

import (
	"encoding/json"
	"log"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/garyburd/redigo/redis"
)

//Broker represents a redis cache configuration + pool of connections to operate against.
type Broker struct {
	pool   *redis.Pool
	config conf.RedisConfig
}

//New constructs a new Broker from config.
func New(conf conf.RedisConfig) (cache *Broker) {
	cache = new(Broker)
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

//PublishEvent broadcasts an event of type etype with location "where" and a payload of data encoded as JSON to all of channels.
func (b *Broker) PublishEvent(etype string, where string, data interface{}, channels []string) {
	conn := b.pool.Get()
	defer conn.Close()
	event := gp.Event{Type: etype, Location: where, Data: data}
	JSONEvent, _ := json.Marshal(event)
	for _, channel := range channels {
		conn.Send("PUBLISH", channel, JSONEvent)
	}
	conn.Flush()
}

//EventSubscribe subscribes to the channels in subscription, and returns them as a combined MsgQueue.
func (b *Broker) EventSubscribe(subscriptions []string) (events gp.MsgQueue) {
	commands := make(chan gp.QueueCommand)
	messages := make(chan []byte)
	events = gp.MsgQueue{Commands: commands, Messages: messages}
	conn := b.pool.Get()
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
