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
	Group  NetworkID `json:"group,omitempty"`
}

//Read represents the most recent message a user has seen in a particular conversation (it doesn't make much sense without that context).
type Read struct {
	UserID   UserID     `json:"user"`
	LastRead MessageID  `json:"last_read"`
	At       *time.Time `json:"at,omitempty"`
}

//Conversation is a container for a bunch of messages.
type Conversation struct {
	ID           ConversationID `json:"id"`
	LastActivity time.Time      `json:"lastActivity,omitempty"`
	Participants []UserPresence `json:"participants"`   //Participants can send messages to and read from this conversation.
	Read         []Read         `json:"read,omitempty"` //Read represents the most recent message each user has seen.
	Unread       int            `json:"unread,omitempty"`
	Muted        bool           `json:"muted,omitempty'`
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

//UserPresence represents a user + their presence in a conversation.
type UserPresence struct {
	User
	Presence *Presence `json:"presence,omitempty"`
}

//Presence represents a user's presence (how recently they were online, and on which form factor) within the app.
type Presence struct {
	Form string    `json:"form"`
	At   time.Time `json:"at"`
}
