package gp

import "time"

//MessageID uniquely identifies a chat message.
type MessageID uint64

//ConversationID identifies a conversation.
type ConversationID uint64

//Message does not contain a conversation ID. If you need that, see RedisMessage.
//TODO: Combine them?
type Message struct {
	ID     MessageID `json:"id"`
	By     User      `json:"by"`
	Text   string    `json:"text"`
	Time   time.Time `json:"timestamp"`
	System bool      `json:"system,omitempty"`
}

//Read represents the most recent message a user has seen in a particular conversation (it doesn't make much sense without that context).
type Read struct {
	UserID   UserID    `json:"user"`
	LastRead MessageID `json:"last_read"`
}

//RedisMessage is a message with a ConversationID so that someone on the other end of a queue can place it in the correct context.
type RedisMessage struct {
	Message
	Conversation ConversationID `json:"conversation_id"`
}

//Conversation is a container for a bunch of messages.
type Conversation struct {
	ID           ConversationID `json:"id"`
	LastActivity time.Time      `json:"lastActivity"`
	Participants []User         `json:"participants"`   //Participants can send messages to and read from this conversation.
	Read         []Read         `json:"read,omitempty"` //Read represents the most recent message each user has seen.
	Unread       int            `json:"unread,omitempty"`
	Group        NetworkID      `json:"group,omitempty"`
}

//ConversationSmall only contains the last message in a conversation - for things like displaying an inbox view.
type ConversationSmall struct {
	Conversation
	LastMessage *Message `json:"mostRecentMessage,omitempty"`
}

//ConversationAndMessages contains the messages in this conversation.
type ConversationAndMessages struct {
	Conversation
	Messages []Message `json:"messages"`
}
