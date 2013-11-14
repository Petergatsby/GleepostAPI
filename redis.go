package main

import (
	"encoding/json"
	"fmt"
	"github.com/draaglom/GleepostAPI/gp"
	"github.com/draaglom/GleepostAPI/db"
	"github.com/garyburd/redigo/redis"
	"log"
	"time"
)

/********************************************************************
		General
********************************************************************/

func RedisDial() (redis.Conn, error) {
	conf := gp.GetConfig()
	conn, err := redis.Dial(conf.Redis.Proto, conf.Redis.Address)
	return conn, err
}

var ErrEmptyCache = gp.APIerror{"Not in redis!"}

/********************************************************************
		Messages
********************************************************************/

func redisPublish(msg gp.RedisMessage) {
	log.Printf("Publishing message to redis: %d, %d", msg.Conversation, msg.Id)
	conn := pool.Get()
	defer conn.Close()
	participants := getParticipants(msg.Conversation)
	JSONmsg, _ := json.Marshal(msg)
	for _, user := range participants {
		conn.Send("PUBLISH", user.Id, JSONmsg)
	}
	conn.Flush()
}

func redisSubscribe(c chan []byte, userId gp.UserId) {
	conn := pool.Get()
	defer conn.Close()
	psc := redis.PubSubConn{Conn: conn}
	psc.Subscribe(userId)
	defer psc.Unsubscribe(userId)
	for {
		switch n := psc.Receive().(type) {
		case redis.Message:
			c <- n.Data
		case redis.Subscription:
			fmt.Printf("%s: %s %d\n", n.Channel, n.Kind, n.Count)
		}
	}
}

func redisMessageChan(userId gp.UserId) (c chan []byte) {
	go redisSubscribe(c, userId)
	return
}

func redisAddMessage(msg gp.Message, convId gp.ConversationId) {
	log.Printf("redis add message %d %d", convId, msg.Id)
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convId)
	conn.Send("ZADD", key, msg.Time.Unix(), msg.Id)
	conn.Flush()
	go redisSetMessage(msg)
}

func redisGetLastMessage(id gp.ConversationId) (message gp.Message, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", id)
	messageId, err := redis.Int(conn.Do("ZREVRANGE", key, 0, 0))
	if err != nil {
		return
	}
	BaseKey := fmt.Sprintf("messages:%d", messageId)
	reply, err := redis.Values(conn.Do("MGET", BaseKey+":by", BaseKey+":text", BaseKey+":time", BaseKey+":seen"))
	if err != nil {
		//should reach this if there is no last message
		log.Printf("error getting message in redis %v", err)
		return message, err
	}
	var by gp.UserId
	var timeString string
	if _, err = redis.Scan(reply, &by, &message.Text, &timeString, &message.Seen); err != nil {
		return message, err
	}
	if by != 0 {
		message.By, err = getUser(by)
		if err != nil {
			log.Printf("error getting user %d %v", by, err)
		}
	}
	message.Id = gp.MessageId(messageId)
	message.Time, err = time.Parse(time.RFC3339, timeString)
	return message, err
}

func redisAddMessages(convId gp.ConversationId, messages []gp.Message) {
	//expecting messages ordered b
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convId)
	for _, message := range messages {
		conn.Send("ZADD", key, message.Time.Unix(), message.Id)
		go redisSetMessage(message)
	}
	conn.Flush()
}

func redisSetMessage(message gp.Message) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("messages:%d", message.Id)
	conn.Send("MSET", key+":by", message.By.Id, key+":text", message.Text, key+":time", message.Time.Format(time.RFC3339), key+":seen", message.Seen)
	conn.Flush()
}

func redisMessageSeen(msgId gp.MessageId) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("messages:%d:seen", msgId)
	conn.Send("SET", key, true)
	conn.Flush()
}

func redisMarkConversationSeen(id gp.UserId, convId gp.ConversationId, upTo gp.MessageId) (err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convId)
	index := -1
	index, err = redis.Int(conn.Do("ZRANK", key, upTo))
	if err != nil {
		return
	}
	if index == 0 {
		return redis.Error("That message isn't in redis!")
	}
	values, err := redis.Values(conn.Do("ZRANGE", key, 0, index))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return redis.Error("No messages for this conversation in redis.")
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
			message, errGettingMessage := getMessage(gp.MessageId(curr))
			if errGettingMessage != nil {
				return errGettingMessage
			} else {
				if message.By.Id != id {
					go redisMessageSeen(message.Id)
				}
			}
		}
	}
	return
}

func redisGetMessages(convId gp.ConversationId, index int64, sel string, count int) (messages []gp.Message, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convId)
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
		finish = int(index) + count
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
		finish = int(index) - 1
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
			message, errGettingMessage := getMessage(gp.MessageId(curr))
			if errGettingMessage != nil {
				return messages, errGettingMessage
			} else {
				go redisSetMessage(message)
			}
			messages = append(messages, message)
		}
	}
	return
}

func redisGetMessage(msgId gp.MessageId) (message gp.Message, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("messages:%d", msgId)
	reply, err := redis.Values(conn.Do("MGET", key+":by", key+":text", key+":time", key+":seen"))
	if err != nil {
		return message, err
	}
	message.Id = msgId
	var timeString string
	var by gp.UserId
	if _, err = redis.Scan(reply, &by, &message.Text, &timeString, &message.Seen); err != nil {
		return message, err
	}
	if by != 0 {
		message.By, err = getUser(by)
		if err != nil {
			log.Printf("error getting user %d %v", by, err)
		}
	}
	message.Time, err = time.Parse(time.RFC3339, timeString)
	return message, err
}

func redisAddAllMessages(convId gp.ConversationId) {
	conf := gp.GetConfig()
	messages, err := db.GetMessages(convId, 0, "start", conf.MessageCache)
	if err != nil {
		log.Printf("%v", err)
	}
	conn := pool.Get()
	defer conn.Close()
	zkey := fmt.Sprintf("conversations:%d:messages", convId)
	for _, message := range messages {
		key := fmt.Sprintf("messages:%d", message.Id)
		conn.Send("ZADD", zkey, message.Time.Unix(), message.Id)
		conn.Send("MSET", key+":by", message.By.Id, key+":text", message.Text, key+":time", message.Time.Format(time.RFC3339), key+":seen", message.Seen)
		conn.Flush()
	}
}

/********************************************************************
		Posts
********************************************************************/

func redisAddPosts(net gp.NetworkId, posts []gp.PostSmall) {
	for _, post := range posts {
		go redisAddPost(post)
		go redisAddNetworkPost(net, post)
	}
}

func redisAddPost(post gp.PostSmall) {
	conn := pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("posts:%d", post.Id)
	conn.Send("MSET", baseKey+":by", post.By.Id, baseKey+":time", post.Time.Format(time.RFC3339), baseKey+":text", post.Text)
	conn.Flush()
}

func redisAddNewPost(userId gp.UserId, text string, postId gp.PostId) {
	var post gp.PostSmall
	post.Id = postId
	post.By, _ = getUser(userId)
	post.Time = time.Now().UTC()
	post.Text = text
	networks, err := getUserNetworks(userId)
	if err == nil {
		go redisAddPost(post)
		go redisAddNetworkPost(networks[0].Id, post)
	}
}

func redisAddNetworkPost(network gp.NetworkId, post gp.PostSmall) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("networks:%d:posts", network)
	exists, _ := redis.Bool(conn.Do("EXISTS", key))
	if !exists { //Without this we might get stuck with only recent posts in cache
		go redisAddAllPosts(network)
	} else {
		conn.Send("ZADD", key, post.Time.Unix(), post.Id)
		conn.Flush()
	}
}

func redisGetPost(postId gp.PostId) (post gp.PostSmall, err error) {
	conn := pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("posts:%d", postId)
	values, err := redis.Values(conn.Do("MGET", baseKey+":by", baseKey+":time", baseKey+":text"))
	if err != nil {
		return post, err
	}
	var by gp.UserId
	var t string
	if _, err = redis.Scan(values, &by, &t, &post.Post.Text); err != nil {
		return post, err
	}
	post.Post.Id = postId
	post.Post.By, err = getUser(by)
	if err != nil {
		return post, err
	}
	post.Post.Time, _ = time.Parse(time.RFC3339, t)
	post.Post.Images = getPostImages(postId)
	post.CommentCount = getCommentCount(postId)
	post.LikeCount, err = likeCount(postId)
	if err != nil {
		return
	}
	return post, nil
}

func redisGetNetworkPosts(id gp.NetworkId, index int64, sel string) (posts []gp.PostSmall, err error) {
	conn := pool.Get()
	defer conn.Close()
	conf := gp.GetConfig()

	key := fmt.Sprintf("networks:%d:posts", id)
	var start, finish int
	switch {
	case sel == "before":
		rindex := -1
		rindex, err = redis.Int(conn.Do("ZREVRANK", key, index))
		if err != nil {
			return
		}
		if rindex < 1 {
			return posts, ErrEmptyCache
		}
		start = rindex + 1
		finish = rindex + conf.MessagePageSize
	case sel == "after":
		rindex := -1
		rindex, err = redis.Int(conn.Do("ZREVRANK", key, index))
		if err != nil {
			return
		}
		if rindex < 1 {
			return posts, ErrEmptyCache
		}
		start = rindex - conf.PostPageSize
		if start < 0 {
			start = 0
		}
		finish = rindex - 1
	default:
		start = int(index)
		finish = int(index) + conf.PostPageSize - 1
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
		postId := gp.PostId(curr)
		post, err := redisGetPost(postId)
		if err != nil {
			return posts, err
		}
		posts = append(posts, post)
	}
	return
}

func redisAddAllPosts(netId gp.NetworkId) {
	conf := gp.GetConfig()
	posts, err := db.GetPosts(netId, 0, conf.PostCache, "start")
	if err != nil {
		log.Println(err)
	}
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("networks:%d:posts", netId)
	for _, post := range posts {
		baseKey := fmt.Sprintf("posts:%d", post.Id)
		conn.Send("MSET", baseKey+":by", post.By.Id, baseKey+":time", post.Time.Format(time.RFC3339), baseKey+":text", post.Text)
		conn.Send("ZADD", key, post.Time.Unix(), post.Id)
		conn.Flush()
	}
}

/********************************************************************
		Conversations
********************************************************************/

func redisUpdateConversation(id gp.ConversationId) {
	conn := pool.Get()
	defer conn.Close()
	participants := getParticipants(id)
	for _, user := range participants {
		key := fmt.Sprintf("users:%d:conversations", user.Id)
		//nb: this means that the last activity time for a conversation will
		//differ slightly from the db to the cache (and even from user to user)
		//but I think this is okay because it's only for ordering purposes
		//(the actual last message timestamp will be consistent)
		conn.Send("ZADD", key, time.Now().Unix(), id)
	}
	conn.Flush()
}

func redisGetConversationMessageCount(convId gp.ConversationId) (count int, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convId)
	count, err = redis.Int(conn.Do("ZCARD", key))
	if err != nil {
		return 0, err
	}
	return count, nil
}

func redisSetConversationParticipants(convId gp.ConversationId, participants []gp.User) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:participants", convId)
	for _, user := range participants {
		conn.Send("SADD", key, user.Id)
	}
	conn.Flush()
}

func redisGetConversationParticipants(convId gp.ConversationId) (participants []gp.User, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:participants", convId)
	values, err := redis.Values(conn.Do("SMEMBERS", key))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return participants, ErrEmptyCache
	}
	for len(values) > 0 {
		user := gp.User{}
		values, err = redis.Scan(values, &user.Id)
		if err != nil {
			return
		}
		user, err = getUser(user.Id)
		if err != nil {
			return
		}
		participants = append(participants, user)
	}
	return
}

func redisGetConversations(id gp.UserId, start int64) (conversations []gp.ConversationSmall, err error) {
	conf := gp.GetConfig()
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:conversations", id)
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, start+int64(conf.ConversationPageSize)-1, "WITHSCORES"))
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
		conv.Id = gp.ConversationId(curr)
		conv.LastActivity = time.Unix(int64(unix), 0).UTC()
		conv.Conversation.Participants = getParticipants(conv.Id)
		expiry, err := conversationExpiry(conv.Id)
		if err == nil {
			conv.Expiry = &expiry
		}
		LastMessage, err := getLastMessage(conv.Id)
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		conversations = append(conversations, conv)
	}
	return
}

func redisConversationExpiry(convId gp.ConversationId) (expiry gp.Expiry, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:expiry", convId)
	t, err := redis.Int(conn.Do("GET", key))
	if err != nil {
		return
	}
	expiry.Time = time.Unix(int64(t), 0).UTC()
	return
}

func redisSetConversationExpiry(conv gp.Conversation) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:expiry", conv.Id)
	conn.Send("SET", key, conv.Expiry.Time.Unix())
	conn.Flush()
}

func redisAddConversation(conv gp.Conversation) {
	conn := pool.Get()
	defer conn.Close()
	go redisSetConversationExpiry(conv)
	for _, participant := range conv.Participants {
		key := fmt.Sprintf("users:%d:conversations", participant.Id)
		conn.Send("ZADD", key, conv.LastActivity.Unix(), conv.Id)
	}
	conn.Flush()
}

/********************************************************************
		Comments
********************************************************************/

func redisGetCommentCount(id gp.PostId) (count int, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("posts:%d:comments", id)
	count, err = redis.Int(conn.Do("ZCARD", key))
	if err != nil {
		return 0, err
	} else {
		return count, nil
	}
}

func redisAddComment(id gp.PostId, comment gp.Comment) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("posts:%d:comments", id)
	baseKey := fmt.Sprintf("comments:%d", comment.Id)
	conn.Send("ZADD", key, comment.Time.Unix(), comment.Id)
	conn.Send("MSET", baseKey+":by", comment.By.Id, baseKey+":text", comment.Text, baseKey+":time", comment.Time.Format(time.RFC3339))
	conn.Flush()
}

func redisAddAllComments(postId gp.PostId) {
	conf := gp.GetConfig()
	comments, err := db.GetComments(postId, 0, conf.CommentCache)
	if err != nil {
		log.Println(err)
	}
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("posts:%d:comments", postId)
	for _, comment := range comments {
		baseKey := fmt.Sprintf("comments:%d", comment.Id)
		conn.Send("ZADD", key, comment.Time.Unix(), comment.Id)
		conn.Send("MSET", baseKey+":by", comment.By.Id, baseKey+":text", comment.Text, baseKey+":time", comment.Time.Format(time.RFC3339))
		conn.Flush()
	}
}

func redisGetComments(postId gp.PostId, start int64) (comments []gp.Comment, err error) {
	conn := pool.Get()
	defer conn.Close()
	conf := gp.GetConfig()
	key := fmt.Sprintf("posts:%d:comments", postId)
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, start+int64(conf.CommentPageSize)-1))
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
		comment, e := redisGetComment(gp.CommentId(curr))
		if e != nil {
			return comments, e
		}
		comments = append(comments, comment)
	}
	return
}

func redisGetComment(commentId gp.CommentId) (comment gp.Comment, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("comments:%d", commentId)
	reply, err := redis.Values(conn.Do("MGET", key+":by", key+":text", key+":time"))
	if err != nil {
		return
	}
	var timeString string
	var by gp.UserId
	if _, err = redis.Scan(reply, &by, &comment.Text, &timeString); err != nil {
		return
	}
	comment.Id = commentId
	comment.By, err = getUser(by)
	if err != nil {
		return
	}
	comment.Time, _ = time.Parse(time.RFC3339, timeString)
	return
}

/********************************************************************
		Networks
********************************************************************/

func redisGetUserNetwork(userId gp.UserId) (networks []gp.Network, err error) {
	/* Part 1 of the transition to one network per user (why did I ever allow more :| */
	//this returns a slice of 1 network to keep compatible with dbGetNetworks
	conn := pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("users:%d:network", userId)
	reply, err := redis.Values(conn.Do("MGET", baseKey+":id", baseKey+":name"))
	if err != nil {
		return networks, err
	}
	net := gp.Network{}
	if _, err = redis.Scan(reply, &net.Id, &net.Name); err != nil {
		return networks, err
	} else if net.Id == 0 {
		//there must be a neater way?
		err = redis.Error("Cache miss")
		return networks, err
	}
	networks = append(networks, net)
	return networks, nil
}

func redisSetUserNetwork(userId gp.UserId, network gp.Network) {
	conn := pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("users:%d:network", userId)
	conn.Send("MSET", baseKey+":id", network.Id, baseKey+":name", network.Name)
	conn.Flush()
}

/********************************************************************
		Users
********************************************************************/

func redisSetUser(user gp.User) {
	conn := pool.Get()
	defer conn.Close()
	BaseKey := fmt.Sprintf("users:%d", user.Id)
	conn.Send("MSET", BaseKey+":name", user.Name, BaseKey+":profile_image", user.Avatar)
	conn.Flush()
}

func redisGetUser(id gp.UserId) (user gp.User, err error) {
	conn := pool.Get()
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
	user.Id = id
	return user, nil
}

func redisSetProfileImage(id gp.UserId, url string) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:profile_image", id)
	conn.Send("SET", key, url)
	conn.Flush()
}

func redisSetBusyStatus(id gp.UserId, busy bool) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:busy", id)
	conn.Send("SET", key, busy)
	conn.Flush()
}

func redisUserPing(id gp.UserId) {
	conf := gp.GetConfig()
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:busy", id)
	conn.Send("SETEX", key, conf.OnlineTimeout, 1)
	conn.Flush()
}

func redisUserIsOnline(id gp.UserId) (online bool) {
	conn := pool.Get()
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

func redisPutToken(token gp.Token) {
	/* Set a session token in redis.
		We use the token value as part of the redis key
	        so that a user may have more than one concurrent session
		(eg: signed in on the web and mobile at once */
	conn := pool.Get()
	defer conn.Close()
	expiry := int(token.Expiry.Sub(time.Now()).Seconds())
	key := fmt.Sprintf("users:%d:token:%s", token.UserId, token.Token)
	conn.Send("SETEX", key, expiry, token.Expiry)
	conn.Flush()
}

func redisTokenExists(id gp.UserId, token string) bool {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:token:%s", id, token)
	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false
	}
	return exists
}
