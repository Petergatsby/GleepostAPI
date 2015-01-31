package cache

import (
	"fmt"
	"time"

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

//AddConversation records this conversation and its participants (but not its messages) in the cache
func (c *Cache) AddConversation(conv gp.Conversation) {
	conn := c.pool.Get()
	defer conn.Close()
	if len(conv.Read) > 0 {
		go c.SetReadStatus(conv.ID, conv.Read...)
	}
	for _, participant := range conv.Participants {
		key := fmt.Sprintf("users:%d:conversations", participant.ID)
		conn.Send("ZADD", key, conv.LastActivity.Unix(), conv.ID)
	}
	conn.Flush()
}

//MessageChan returns a channel containing events for userID (Contents are slices of byte containing JSON)
func (c *Cache) MessageChan(userID gp.UserID) (messages chan []byte) {
	messages = make(chan []byte)
	go c.Subscribe(messages, userID)
	return
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
func (c *Cache) AddMessages(convID gp.ConversationID, messages ...gp.Message) {
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

//SetReadStatus marks convId seen up to messageID for each read (userID:messageID pair).
func (c *Cache) SetReadStatus(convID gp.ConversationID, read ...gp.Read) {
	for _, r := range read {
		conn := c.pool.Get()
		defer conn.Close()
		key := fmt.Sprintf("conversations:%d:read", convID)
		conn.Send("HSET", key, r.UserID, r.LastRead)
		conn.Flush()
		return
	}
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
