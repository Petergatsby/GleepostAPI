package gp

import "time"

//NetworkID is the id of a network (which Groups are a subset of).
type NetworkID uint64

//Network is any grouping of users / posts - ie, a university or a user-created group.
type Network struct {
	ID   NetworkID `json:"id"`
	Name string    `json:"name"`
}

//Role is a particular permissions level / name pair within a network.
type Role struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

//Group is a user-group. It's a network with a cover image, a description and maybe a creator.
type Group struct {
	Network
	Image        string         `json:"image,omitempty"`
	Desc         string         `json:"description,omitempty"`
	Creator      *User          `json:"creator,omitempty"`
	Privacy      string         `json:"privacy,omitempty"`
	MemberCount  int            `json:"size,omitempty"`
	Conversation ConversationID `json:"conversation,omitempty"`
}

//GroupMembership is a group and a user's membership status in that group.
type GroupMembership struct {
	Group
	UnreadCount  int `json:"unread,omitempty"`
	Role         `json:"role"`
	LastActivity time.Time `json:"last_activity,omitempty"`
}

//Rule represents a condition that makes a user part of a particular Network. At the moment the only possible Rule.Type is "email";
//Rule.Value must then be a domain (eg "gleepost.com") - verified owners of emails in this domain will get added to this network.
type Rule struct {
	NetworkID NetworkID
	Type      string
	Value     string
}
