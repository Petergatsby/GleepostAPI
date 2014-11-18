package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/db"
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
	cache.pool = redis.NewPool(getDialer(conf), 100)
	return
}

func getDialer(conf conf.RedisConfig) func() (redis.Conn, error) {
	f := func() (redis.Conn, error) {
		conn, err := redis.Dial(conf.Proto, conf.Address)
		return conn, err
	}
	return f
}

//ErrEmptyCache is a cache miss
var ErrEmptyCache = gp.APIerror{Reason: "Not in redis!"}

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

/********************************************************************
		Posts
********************************************************************/

//AddPosts adds all the posts to this network.
func (c *Cache) addPosts(net gp.NetworkID, posts []gp.Post) (err error) {
	for _, post := range posts {
		go c.AddPost(post)
		err = c.AddPostToNetwork(post, net)
		if err != nil {
			return
		}
	}
	return
}

//AddPost adds a post into the cache but doesn't record its membership in a network.
func (c *Cache) AddPost(post gp.Post) {
	conn := c.pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("posts:%d", post.ID)
	conn.Send("MSET", baseKey+":by", post.By.ID, baseKey+":time", post.Time.Format(time.RFC3339), baseKey+":text", post.Text)
	conn.Flush()
}

//AddPostToNetwork records that this post is in network.
func (c *Cache) AddPostToNetwork(post gp.Post, network gp.NetworkID) (err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("networks:%d:posts", network)
	exists, _ := redis.Bool(conn.Do("EXISTS", key))
	if !exists { //Without this we might get stuck with only recent posts in cache
		return ErrEmptyCache
	}
	conn.Send("ZADD", key, post.Time.Unix(), post.ID)
	conn.Flush()
	return nil
}

//GetPost fetches the core details of a post from the cache, or returns an error if it's not in the cache (maybe)
func (c *Cache) GetPost(postID gp.PostID) (post gp.PostCore, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("posts:%d", postID)
	values, err := redis.Values(conn.Do("MGET", baseKey+":by", baseKey+":time", baseKey+":text"))
	if err != nil {
		return post, err
	}
	var by gp.UserID
	var t string
	if _, err = redis.Scan(values, &by, &t, &post.Text); err != nil {
		return post, err
	}
	post.ID = postID
	post.By, err = c.GetUser(by)
	if err != nil {
		return post, err
	}
	post.Time, _ = time.Parse(time.RFC3339, t)
	return post, nil
}

//GetPosts returns posts in this network in a manner mirroring db.NewGetPosts.
//TODO: Return posts which don't embed a user
func (c *Cache) getPosts(id gp.NetworkID, mode int, index int64, count int) (posts []gp.PostCore, err error) {
	conn := c.pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("networks:%d:posts", id)
	var start, finish int
	switch {
	case mode == gp.OBEFORE:
		rindex := -1
		rindex, err = redis.Int(conn.Do("ZREVRANK", key, index))
		if err != nil {
			return
		}
		if rindex < 1 {
			return posts, ErrEmptyCache
		}
		start = rindex + 1
		finish = rindex + count
	case mode == gp.OAFTER:
		rindex := -1
		rindex, err = redis.Int(conn.Do("ZREVRANK", key, index))
		if err != nil {
			return
		}
		if rindex < 1 {
			return posts, ErrEmptyCache
		}
		start = rindex - count
		if start < 0 {
			start = 0
		}
		finish = rindex - 1
	default:
		start = int(index)
		finish = int(index) + count - 1
	}
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, finish))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return posts, ErrEmptyCache
	}
	for len(values) > 0 {
		curr := -1
		values, err = redis.Scan(values, &curr)
		if err != nil {
			return
		}
		if curr == -1 {
			return
		}
		postID := gp.PostID(curr)
		post, err := c.GetPost(postID)
		if err != nil {
			return posts, err
		}
		posts = append(posts, post)
	}
	return
}

//AddPostsFromDB refills an empty cache from the database.
func (c *Cache) addPostsFromDB(netID gp.NetworkID, db *db.DB) {
	posts, err := db.GetPosts(netID, 1, 0, c.config.PostCache, "")
	if err != nil {
		log.Println(err)
	}
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("networks:%d:posts", netID)
	for _, post := range posts {
		baseKey := fmt.Sprintf("posts:%d", post.ID)
		conn.Send("MSET", baseKey+":by", post.By.ID, baseKey+":time", post.Time.Format(time.RFC3339), baseKey+":text", post.Text)
		conn.Send("ZADD", key, post.Time.Unix(), post.ID)
		conn.Flush()
	}
}

//EventSubscribe subscribes to the channels in subscription, and returns them as a combined MsgQueue.
func (c *Cache) EventSubscribe(subscriptions []string) (events gp.MsgQueue) {
	commands := make(chan gp.QueueCommand)
	log.Println("Made a new command channel")
	messages := make(chan []byte)
	log.Println("Made a new message channel")
	events = gp.MsgQueue{Commands: commands, Messages: messages}
	conn := c.pool.Get()
	log.Println("Got a redis connection")
	psc := redis.PubSubConn{Conn: conn}
	for _, s := range subscriptions {
		psc.Subscribe(s)
	}
	log.Println("Subscribed to some stuff: ", subscriptions)
	go controller(&psc, events.Commands)
	log.Println("Launched a goroutine to listen for unsub")
	go messageReceiver(&psc, events.Messages)
	log.Println("Launched a goroutine to get messages")
	return events
}

func messageReceiver(psc *redis.PubSubConn, messages chan<- []byte) {
	for {
		switch n := psc.Receive().(type) {
		case redis.Message:
			log.Printf("Got a message: %s", n.Data)
			messages <- n.Data
		case redis.Subscription:
			log.Println("Saw a subscription event: ", n.Count)
			if n.Count == 0 {
				close(messages)
				psc.Conn.Close()
				return
			}
		case error:
			log.Println("Saw an error: ", n)
			log.Println(n)
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
		if command.Command == "UNSUBSCRIBE" && command.Value == "" {
			psc.Unsubscribe()
			return
		}
	}
}
