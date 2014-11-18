package cache

import (
	"fmt"
	"time"

	"github.com/draaglom/GleepostAPI/lib/db"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/garyburd/redigo/redis"
)

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
	messages = make([]gp.Message, 0)
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
