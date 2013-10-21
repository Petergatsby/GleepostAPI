package main

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"strconv"
	"time"
)

/********************************************************************
		General
********************************************************************/

func RedisDial() (redis.Conn, error) {
	conf := GetConfig()
	conn, err := redis.Dial(conf.RedisProto, conf.RedisAddress)
	return conn, err
}

/********************************************************************
		Messages
********************************************************************/

func redisPublish(msg RedisMessage) {
	log.Printf("Publishing message to redis: %d, %d", msg.Conversation, msg.Id)
	conn := pool.Get()
	defer conn.Close()
	participants := getParticipants(msg.Conversation)
	JSONmsg, _ := json.Marshal(msg)
	for _, user := range participants {
		if user.Id != msg.By.Id {
			conn.Send("PUBLISH", user.Id, JSONmsg)
		}
	}
	conn.Flush()
}

func redisAddMessage(msg Message, convId ConversationId) {
	log.Printf("redis add message %d %d", convId, msg.Id)
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convId)
	conn.Send("ZADD", key, msg.Time.Unix(), msg.Id)
	conn.Flush()
	go redisSetMessage(msg)
}

func redisGetMessagesAfter(convId ConversationId, after int64) (messages []Message, err error) {
	conf := GetConfig()
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convId)
	index := -1
	index, err = redis.Int(conn.Do("ZREVRANK", key, after))
	if err != nil {
		return
	}
	if index == 0 {
		return messages, redis.Error("That message isn't in redis!")
	}
	start := index - conf.MessagePageSize
	if start < 0 {
		start = 0
	}
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, index-1))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return messages, redis.Error("No messages for this conversation in redis.")
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
			message, errGettingMessage := getMessage(MessageId(curr))
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

func redisGetLastMessage(id ConversationId) (message Message, err error) {
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
	var by UserId
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
	message.Id = MessageId(messageId)
	message.Time, err = time.Parse(time.RFC3339, timeString)
	return message, err
}

func redisAddMessages(convId ConversationId, messages []Message) {
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

func redisSetMessage(message Message) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("messages:%d", message.Id)
	conn.Send("MSET", key+":by", message.By.Id, key+":text", message.Text, key+":time", message.Time.Format(time.RFC3339), key+":seen", message.Seen)
	conn.Flush()
}

func redisGetMessages(convId ConversationId, start int64) (messages []Message, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convId)
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, start+19))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return messages, redis.Error("No messages for this conversation in redis.")
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
			message, errGettingMessage := getMessage(MessageId(curr))
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

func redisGetMessage(msgId MessageId) (message Message, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("messages:%d", msgId)
	reply, err := redis.Values(conn.Do("MGET", key+":by", key+":text", key+":time", key+":seen"))
	if err != nil {
		return message, err
	}
	message.Id = msgId
	var timeString string
	var by UserId
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

func redisAddAllMessages(convId ConversationId) {
	conf := GetConfig()
	rows, err := messageSelectStmt.Query(convId, 0, conf.MessageCache)
	defer rows.Close()
	log.Println("DB hit: allMessages convid, start (message.id, message.by, message.text, message.time, message.seen)")
	if err != nil {
		log.Printf("%v", err)
	}
	conn := pool.Get()
	defer conn.Close()
	zkey := fmt.Sprintf("conversations:%d:messages", convId)
	for rows.Next() {
		var message Message
		var timeString string
		var by UserId
		err := rows.Scan(&message.Id, &by, &message.Text, &timeString, &message.Seen)
		if err != nil {
			log.Printf("%v", err)
		}
		message.Time, err = time.Parse(MysqlTime, timeString)
		if err != nil {
			log.Printf("%v", err)
		}
		message.By, err = getUser(by)
		if err != nil {
			//should only happen if a message is from a non-existent user
			//(or the db is fucked :))
			log.Println(err)
		}
		key := fmt.Sprintf("messages:%d", message.Id)
		conn.Send("ZADD", zkey, message.Time.Unix(), message.Id)
		conn.Send("MSET", key+":by", message.By.Id, key+":text", message.Text, key+":time", message.Time.Format(time.RFC3339), key+":seen", message.Seen)
		conn.Flush()
	}
}

/********************************************************************
		Posts
********************************************************************/

func redisAddPosts(net NetworkId, posts []PostSmall) {
	for _, post := range posts {
		go redisAddPost(post)
		go redisAddNetworkPost(net, post)
	}
}

func redisAddPost(post PostSmall) {
	conn := pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("posts:%d", post.Id)
	conn.Send("MSET", baseKey+":by", post.By.Id, baseKey+":time", post.Time.Format(time.RFC3339), baseKey+":text", post.Text)
	conn.Flush()
}

func redisAddNewPost(userId UserId, text string, postId PostId) {
	var post PostSmall
	post.Id = postId
	post.By, _ = getUser(userId)
	post.Time = time.Now().UTC()
	post.Text = text
	networks := getUserNetworks(userId)
	go redisAddPost(post)
	go redisAddNetworkPost(networks[0].Id, post)
}

func redisAddNetworkPost(network NetworkId, post PostSmall) {
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

func redisGetPost(postId PostId) (post PostSmall, err error) {
	conn := pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("posts:%d", postId)
	values, err := redis.Values(conn.Do("MGET", baseKey+":by", baseKey+":time", baseKey+":text"))
	if err != nil {
		return post, err
	}
	var by UserId
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
	return post, nil
}

func redisGetNetworkPosts(id NetworkId, start int64) (posts []PostSmall, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("networks:%d:posts", id)
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, start+19))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return posts, redis.Error("No posts for this network in redis.")
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
		postId := PostId(curr)
		post, err := redisGetPost(postId)
		if err != nil {
			return posts, err
		}
		posts = append(posts, post)
	}
	return
}

func redisAddAllPosts(netId NetworkId) {
	conf := GetConfig()
	posts, err := dbGetPosts(netId, 0, conf.PostCache)
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

func redisUpdateConversation(id ConversationId) {
	conn := pool.Get()
	defer conn.Close()
	participants := getParticipants(id)
	for _, user := range participants {
		key := "users:" + strconv.FormatUint(uint64(user.Id), 10) + ":conversations"
		//nb: this means that the last activity time for a conversation will
		//differ slightly from the db to the cache (and even from user to user)
		//but I think this is okay because it's only for ordering purposes
		//(the actual last message timestamp will be consistent)
		conn.Send("ZADD", key, time.Now().Unix(), id)
	}
	conn.Flush()
}

func redisGetConversationMessageCount(convId ConversationId) (count int, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:messages", convId)
	count, err = redis.Int(conn.Do("ZCARD", key))
	if err != nil {
		return 0, err
	}
	return count, nil
}

func redisSetConversationParticipants(convId ConversationId, participants []User) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:participants", convId)
	for _, user := range participants {
		conn.Send("HSET", key, user.Id, user.Name)
	}
	conn.Flush()
}

func redisGetConversationParticipants(convId ConversationId) (participants []User, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("conversations:%d:participants", convId)
	values, err := redis.Values(conn.Do("HGETALL", key))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return participants, redis.Error("Nothing in redis")
	}
	for len(values) > 0 {
		user := User{}
		values, err = redis.Scan(values, &user.Id, &user.Name)
		if err != nil {
			return
		}
		participants = append(participants, user)
	}
	return
}

func redisGetConversations(id UserId, start int64) (conversations []ConversationSmall, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:conversations", id)
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, start+19))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return conversations, redis.Error("No conversations for this user in redis.")
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
		conv := ConversationSmall{}
		conv.Id = ConversationId(curr)
		conv.Conversation.Participants = getParticipants(conv.Id)
		LastMessage, err := getLastMessage(conv.Id)
		if err == nil {
			conv.LastMessage = &LastMessage
		}
		conversations = append(conversations, conv)
	}
	return
}

func redisAddConversation(conv ConversationSmall) {
	conn := pool.Get()
	defer conn.Close()
	for _, participant := range conv.Participants {
		key := fmt.Sprintf("users:%d:conversations", participant.Id)
		conn.Send("ZADD", key, conv.LastActivity.Unix(), conv.Id)
	}
	conn.Flush()
}

/********************************************************************
		Comments
********************************************************************/

func redisGetCommentCount(id PostId) (count int, err error) {
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

func redisAddComment(id PostId, comment Comment) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("posts:%d:comments", id)
	baseKey := fmt.Sprintf("comments:%d", comment.Id)
	conn.Send("ZADD", key, comment.Time.Unix(), comment.Id)
	conn.Send("MSET", baseKey+":by", comment.By.Id, baseKey+":text", comment.Text, baseKey+":time", comment.Time.Format(time.RFC3339))
	conn.Flush()
}

func redisAddAllComments(postId PostId) {
	conf := GetConfig()
	comments, err := dbGetComments(postId, 0, conf.CommentCache)
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

func redisGetComments(postId PostId, start int64) (comments []Comment, err error) {
	conn := pool.Get()
	defer conn.Close()
	conf := GetConfig()
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
		comment, e := redisGetComment(CommentId(curr))
		if e != nil {
			return comments, e
		}
		comments = append(comments, comment)
	}
	return
}

func redisGetComment(commentId CommentId) (comment Comment, err error) {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("comments:%d", commentId)
	reply, err := redis.Values(conn.Do("MGET", key+":by", key+":text", key+":time"))
	if err != nil {
		return
	}
	var timeString string
	var by UserId
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

func redisGetUserNetwork(userId UserId) (networks []Network, err error) {
	/* Part 1 of the transition to one network per user (why did I ever allow more :| */
	//this returns a slice of 1 network to keep compatible with dbGetNetworks
	conn := pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("users:%d:network", userId)
	reply, err := redis.Values(conn.Do("MGET", baseKey+":id", baseKey+":name"))
	if err != nil {
		return networks, err
	}
	net := Network{}
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

func redisSetUserNetwork(userId UserId, network Network) {
	conn := pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("users:%d:network", userId)
	conn.Send("MSET", baseKey+":id", network.Id, baseKey+":name", network.Name)
	conn.Flush()
}

/********************************************************************
		Users
********************************************************************/

func redisSetUser(user User) {
	conn := pool.Get()
	defer conn.Close()
	BaseKey := fmt.Sprintf("users:%d", user.Id)
	conn.Send("MSET", BaseKey+":name", user.Name, BaseKey+":profile_image", user.Avatar)
	conn.Flush()
}

func redisGetUser(id UserId) (user User, err error) {
	conn := pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("users:%d", id)
	values, err := redis.Values(conn.Do("MGET", baseKey+":name", baseKey+":profile_image"))
	if len(values) < 3 {
		return user, redis.Error("That user isn't cached!")
	}
	if _, err := redis.Scan(values, &user.Name, &user.Avatar); err != nil {
		return user, err
	}
	user.Id = id
	return user, nil
}

/********************************************************************
		Tokens
********************************************************************/

func redisPutToken(token Token) {
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

func redisTokenExists(id UserId, token string) bool {
	conn := pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:token:%s", id, token)
	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false
	}
	return exists
}
