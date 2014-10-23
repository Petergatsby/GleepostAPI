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

//MessageChan returns a channel containing events for userID (Contents are slices of byte containing JSON)
func (c *Cache) MessageChan(userID gp.UserID) (messages chan []byte) {
	messages = make(chan []byte)
	go c.Subscribe(messages, userID)
	return
}

//AddMessage records msg.ID in the ZSET for convID (with its unix timestamp as ZSCORE)
func (c *Cache) AddMessage(msg gp.Message, convID gp.ConversationID) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convID)
	conn.Send("ZADD", key, msg.Time.Unix(), msg.ID)
	conn.Flush()
	go c.SetMessage(msg)
}

//GetLastMessage returns the most recent message in this conversation if available,
//or an error (not sure what exactly) if it's not in the cache.
func (c *Cache) GetLastMessage(id gp.ConversationID) (message gp.Message, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", id)
	messageID, err := redis.Int(conn.Do("ZREVRANGE", key, 0, 0))
	if err != nil {
		return
	}
	message, err = c.GetMessage(gp.MessageID(messageID))
	return message, err
}

//AddMessages - Identical to addMessage, except it can do several messages at once.
func (c *Cache) AddMessages(convID gp.ConversationID, messages []gp.Message) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convID)
	for _, message := range messages {
		conn.Send("ZADD", key, message.Time.Unix(), message.ID)
		go c.SetMessage(message)
	}
	conn.Flush()
}

//SetMessage records the actual content of the message at messages:id:by, message:id:text, messages:id:time (RFC3339)
func (c *Cache) SetMessage(message gp.Message) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("messages:%d", message.ID)
	conn.Send("MSET", key+":by", message.By.ID, key+":text", message.Text, key+":time", message.Time.Format(time.RFC3339))
	conn.Flush()
}

//MarkConversationSeen registers the id:upTo (last read) pair in redis for convId
func (c *Cache) MarkConversationSeen(id gp.UserID, convID gp.ConversationID, upTo gp.MessageID) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:read", convID)
	conn.Send("HSET", key, id, upTo)
	conn.Flush()
	return
}

//SetReadStatus marks convId seen up to messageID for each read (userID:messageID pair).
//I don't know what this is for?
func (c *Cache) SetReadStatus(convID gp.ConversationID, read []gp.Read) {
	for _, r := range read {
		c.MarkConversationSeen(r.UserID, convID, r.LastRead)
	}
}

//GetMessages returns this conversation's messages, in a manner specified by sel; "before" specifies messages earler than index, "after" specifies messages newer than index, and "start" returns messages that are after the start-th in a chronological order (ie, pagination starting from oldest)
func (c *Cache) GetMessages(convID gp.ConversationID, index int64, sel string, count int) (messages []gp.Message, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convID)
	var start, finish int
	switch {
	case sel == "before":
		rindex := -1
		rindex, err = redis.Int(conn.Do("ZREVRANK", key, index))
		if err != nil {
			return
		}
		if rindex <= 0 {
			return messages, ErrEmptyCache
		}
		start = rindex + 1
		finish = int(rindex) + count
	case sel == "after":
		rindex := -1
		rindex, err = redis.Int(conn.Do("ZREVRANK", key, index))
		if err != nil {
			return
		}
		if rindex <= 0 {
			return messages, ErrEmptyCache
		}
		start = rindex - count
		if start < 0 {
			start = 0
		}
		finish = int(rindex) - 1
	default:
		start = int(index)
		finish = int(index) + count - 1
	}
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, finish))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return messages, ErrEmptyCache
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
		if curr != 0 {
			message, errGettingMessage := c.GetMessage(gp.MessageID(curr))
			if errGettingMessage != nil {
				return messages, errGettingMessage
			}
			go c.SetMessage(message)
			messages = append(messages, message)
		}
	}
	return
}

//GetMessage attempts to retrieve the message with id msgId from cache. If it doesn't exist in the cache it returns an error. Maybe.
//TODO: get a message which doesn't embed a gp.User
//TODO: return an APIerror when the message doesn't exist.
func (c *Cache) GetMessage(msgID gp.MessageID) (message gp.Message, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("messages:%d", msgID)
	reply, err := redis.Values(conn.Do("MGET", key+":by", key+":text", key+":time"))
	if err != nil {
		return message, err
	}
	message.ID = msgID
	var timeString string
	var by gp.UserID
	if _, err = redis.Scan(reply, &by, &message.Text, &timeString); err != nil {
		return message, err
	}
	if by != 0 {
		message.By, err = c.GetUser(by)
		if err != nil {
			return
		}
	}
	message.Time, err = time.Parse(time.RFC3339, timeString)
	return message, err
}

//AddMessagesFromDB takes up to config.MessageCache messages from the database and adds them to the cache.
func (c *Cache) AddMessagesFromDB(convID gp.ConversationID, db db.DB) (err error) {
	messages, err := db.GetMessages(convID, 0, "start", c.config.MessageCache)
	if err != nil {
		return
	}
	conn := c.pool.Get()
	defer conn.Close()
	zkey := fmt.Sprintf("conversations:%d:messages", convID)
	for _, message := range messages {
		key := fmt.Sprintf("messages:%d", message.ID)
		conn.Send("ZADD", zkey, message.Time.Unix(), message.ID)
		conn.Send("MSET", key+":by", message.By.ID, key+":text", message.Text, key+":time", message.Time.Format(time.RFC3339))
		conn.Flush()
	}
	return nil
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

/********************************************************************
		Conversations
********************************************************************/

//UpdateConversationLists bumps this ConversationID up to the top of each participant's conversation list.
func (c *Cache) UpdateConversationLists(participants []gp.User, id gp.ConversationID) {
	conn := c.pool.Get()
	defer conn.Close()
	for _, user := range participants {
		key := fmt.Sprintf("users:%d:conversations", user.ID)
		//nb: this means that the last activity time for a conversation will
		//differ slightly from the db to the cache (and even from user to user)
		//but I think this is okay because it's only for ordering purposes
		//(the actual last message timestamp will be consistent)
		conn.Send("ZADD", key, time.Now().Unix(), id)
	}
	conn.Flush()
}

func (c *Cache) getConversationMessageCount(convID gp.ConversationID) (count int, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convID)
	count, err = redis.Int(conn.Do("ZCARD", key))
	if err != nil {
		return 0, err
	}
	return count, nil
}

//SetConversationParticipants records all of these users in a redis set at conversations:convID:participants
func (c *Cache) SetConversationParticipants(convID gp.ConversationID, participants []gp.User) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:participants", convID)
	for _, user := range participants {
		conn.Send("SADD", key, user.ID)
	}
	conn.Flush()
}

//GetParticipants returns this conversation's participants.
//TODO: Return []gp.UserId.
func (c *Cache) GetParticipants(convID gp.ConversationID) (participants []gp.User, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:participants", convID)
	values, err := redis.Values(conn.Do("SMEMBERS", key))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return participants, ErrEmptyCache
	}
	for len(values) > 0 {
		user := gp.User{}
		values, err = redis.Scan(values, &user.ID)
		if err != nil {
			return
		}
		user, err = c.GetUser(user.ID)
		if err != nil {
			return
		}
		participants = append(participants, user)
	}
	return
}

//TODO: return []gp.ConversationId.
func (c *Cache) getConversations(id gp.UserID, start int64, count int) (conversations []gp.ConversationSmall, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:conversations", id)
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, start+int64(count)-1, "WITHSCORES"))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return conversations, redis.Error("No conversations for this user in redis.")
	}
	for len(values) > 0 {
		curr := -1
		unix := -1
		values, err = redis.Scan(values, &curr, &unix)
		if err != nil {
			return
		}
		if curr == -1 || unix == -1 {
			return
		}
		conv := gp.ConversationSmall{}
		conv.ID = gp.ConversationID(curr)
		conv.LastActivity = time.Unix(int64(unix), 0).UTC()
		conv.Participants, err = c.GetParticipants(conv.ID)
		if err != nil {
			return
		}
		conv.Read, err = c.GetRead(conv.ID)
		if err != nil {
			return
		}
		LastMessage, err := c.GetLastMessage(conv.ID)
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		conversations = append(conversations, conv)
	}
	return
}

//GetRead returns the point which participants have read up to in conversation convId.
func (c *Cache) GetRead(convID gp.ConversationID) (read []gp.Read, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:read", convID)
	values, err := redis.Values(conn.Do("HGETALL", key))
	if err != nil {
		return
	}
	if len(values) < 1 {
		err = ErrEmptyCache
		return
	}
	for len(values) > 0 {
		var r gp.Read
		values, err = redis.Scan(values, &r.UserID, &r.LastRead)
		if err != nil {
			return
		}
		read = append(read, r)
	}
	return
}

//ConversationExpiry returns the Expiry of this conversation, or an error if it's missing from the cache.
func (c *Cache) ConversationExpiry(convID gp.ConversationID) (expiry gp.Expiry, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d", convID)
	t, err := redis.Int(conn.Do("GET", key+":expiry"))
	if err != nil {
		return
	}
	expiry.Ended, err = redis.Bool(conn.Do("GET", key+":ended"))
	expiry.Time = time.Unix(int64(t), 0).UTC()
	return
}

//SetConversationExpiry records this conversation's expiry in the cache (NB: expiry meaning "conversation end time" not cache-expiry).
func (c *Cache) SetConversationExpiry(convID gp.ConversationID, expiry gp.Expiry) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d", convID)
	conn.Send("MSET", key+":expiry", expiry.Time.Unix(), key+":ended", expiry.Ended)
	conn.Flush()
}

//DelConversationExpiry removes an expiry (ie, it will now no longer end).
//TODO: think of something better than a cache miss
func (c *Cache) DelConversationExpiry(convID gp.ConversationID) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d", convID)
	conn.Send("DEL", key+":expiry", key+":ended")
	conn.Flush()
}

//AddConversation records this conversation and its participants (but not its messages) in the cache
func (c *Cache) AddConversation(conv gp.Conversation) {
	conn := c.pool.Get()
	defer conn.Close()
	if conv.Expiry != nil {
		go c.SetConversationExpiry(conv.ID, *conv.Expiry)
	}
	if len(conv.Read) > 0 {
		go c.SetReadStatus(conv.ID, conv.Read)
	}
	for _, participant := range conv.Participants {
		key := fmt.Sprintf("users:%d:conversations", participant.ID)
		conn.Send("ZADD", key, conv.LastActivity.Unix(), conv.ID)
	}
	conn.Flush()
}

//TerminateConversation marks a conversation as ended in the cache.
func (c *Cache) TerminateConversation(convID gp.ConversationID) (err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:ended", convID)
	conn.Send("SET", key, true)
	participants, err := c.GetParticipants(convID)
	if err != nil {
		for _, p := range participants {
			conn.Send("ZREM", fmt.Sprintf("users:%d:conversations", p.ID), convID)
		}
	}
	conn.Flush()
	return
}

/********************************************************************
		Comments
********************************************************************/

//GetCommentCount returns the total number of comments on this post.
func (c *Cache) GetCommentCount(id gp.PostID) (count int, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("posts:%d:comments", id)
	count, err = redis.Int(conn.Do("ZCARD", key))
	if err != nil {
		return 0, err
	}
	return count, nil
}

//AddComment places this comment in the cache.
func (c *Cache) AddComment(id gp.PostID, comment gp.Comment) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("posts:%d:comments", id)
	baseKey := fmt.Sprintf("comments:%d", comment.ID)
	conn.Send("ZADD", key, comment.Time.Unix(), comment.ID)
	conn.Send("MSET", baseKey+":by", comment.By.ID, baseKey+":text", comment.Text, baseKey+":time", comment.Time.Format(time.RFC3339))
	conn.Flush()
}

//AddAllCommentsFromDB pulls the most recent cache.config.CommentCache comments from the database.
func (c *Cache) AddAllCommentsFromDB(postID gp.PostID, db *db.DB) {
	comments, err := db.GetComments(postID, 0, c.config.CommentCache)
	if err != nil {
		log.Println(err)
	}
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("posts:%d:comments", postID)
	for _, comment := range comments {
		baseKey := fmt.Sprintf("comments:%d", comment.ID)
		conn.Send("ZADD", key, comment.Time.Unix(), comment.ID)
		conn.Send("MSET", baseKey+":by", comment.By.ID, baseKey+":text", comment.Text, baseKey+":time", comment.Time.Format(time.RFC3339))
		conn.Flush()
	}
}

//GetComments returns the comments on this post, ordered from oldest to newest, starting from start.
func (c *Cache) GetComments(postID gp.PostID, start int64, count int) (comments []gp.Comment, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("posts:%d:comments", postID)
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, start+int64(count)-1))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return comments, redis.Error("No conversations for this user in redis.")
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
		comment, e := c.GetComment(gp.CommentID(curr))
		if e != nil {
			return comments, e
		}
		comments = append(comments, comment)
	}
	return
}

//GetComment - a particular comment in the cache.
func (c *Cache) GetComment(commentID gp.CommentID) (comment gp.Comment, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("comments:%d", commentID)
	reply, err := redis.Values(conn.Do("MGET", key+":by", key+":text", key+":time"))
	if err != nil {
		return
	}
	var timeString string
	var by gp.UserID
	if _, err = redis.Scan(reply, &by, &comment.Text, &timeString); err != nil {
		return
	}
	comment.ID = commentID
	comment.By, err = c.GetUser(by)
	if err != nil {
		return
	}
	comment.Time, _ = time.Parse(time.RFC3339, timeString)
	return
}

/********************************************************************
		Users
********************************************************************/

//SetUser - cache a copy of this user.
func (c *Cache) SetUser(user gp.User) {
	conn := c.pool.Get()
	defer conn.Close()
	BaseKey := fmt.Sprintf("users:%d", user.ID)
	conn.Send("MSET", BaseKey+":name", user.Name, BaseKey+":profile_image", user.Avatar)
	conn.Flush()
}

//GetUser - retrieve a cached User, or a redis.Error if they're not in the cache.
func (c *Cache) GetUser(id gp.UserID) (user gp.User, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("users:%d", id)
	values, err := redis.Values(conn.Do("MGET", baseKey+":name", baseKey+":profile_image"))
	if err != nil {
		return user, err
	}
	if len(values) < 2 {
		return user, redis.Error("That user isn't cached!")
	}
	if _, err := redis.Scan(values, &user.Name, &user.Avatar); err != nil {
		return user, err
	}
	if user.Name == "" {
		return user, redis.Error("That user isn't cached!")
	}
	user.ID = id
	return user, nil
}

//SetProfileImage records your avatar in the cache.
func (c *Cache) SetProfileImage(id gp.UserID, url string) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:profile_image", id)
	conn.Send("SET", key, url)
	conn.Flush()
}

//SetBusyStatus records if this user is busy or not.
func (c *Cache) SetBusyStatus(id gp.UserID, busy bool) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:busy", id)
	conn.Send("SET", key, busy)
	conn.Flush()
}

//UserPing marks this user as busy for the next timeout seconds.
func (c *Cache) UserPing(id gp.UserID, timeout int) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:busy", id)
	conn.Send("SETEX", key, timeout, 1)
	conn.Flush()
}

//UserIsOnline returns true if this user is online.
//Should this use users:userID:busy??
func (c *Cache) UserIsOnline(id gp.UserID) (online bool) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:busy", id)
	online, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false
	}
	return
}

/********************************************************************
		Tokens
********************************************************************/

//PutToken records this token in the cache until it expires.
func (c *Cache) PutToken(token gp.Token) {
	/* Set a session token in redis.
		We use the token value as part of the redis key
	        so that a user may have more than one concurrent session
		(eg: signed in on the web and mobile at once */
	conn := c.pool.Get()
	defer conn.Close()
	expiry := int(token.Expiry.Sub(time.Now()).Seconds())
	key := fmt.Sprintf("users:%d:token:%s", token.UserID, token.Token)
	conn.Send("SETEX", key, expiry, token.Expiry)
	conn.Flush()
}

//TokenExists returns true if this id:token pair exists.
func (c *Cache) TokenExists(id gp.UserID, token string) bool {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:token:%s", id, token)
	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false
	}
	return exists
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
