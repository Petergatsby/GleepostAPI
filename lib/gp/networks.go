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

//ParentedGroup is a group which indicates its parent network (ie, its university)
type ParentedGroup struct {
	Group
	Parent NetworkID `json:"parent,omitempty"`
}

//GroupSubjective is a group plus (a) potential context (ie, the role of a user within that group) and (b) your own relation to that group (your role, unread, request status etc)
type GroupSubjective struct {
	Group
	UnreadCount    int        `json:"unread,omitempty"`
	YourRole       *Role      `json:"role,omitempty"`
	TheirRole      *Role      `json:"their_role,omitempty"`
	LastActivity   *time.Time `json:"last_activity,omitempty"`
	NewPosts       int        `json:"new_posts,omitempty"`
	PendingRequest bool       `json:"pending_request,omitempty"`
}

//Rule represents a condition that makes a user part of a particular Network. At the moment the only possible Rule.Type is "email";
//Rule.Value must then be a domain (eg "gleepost.com") - verified owners of emails in this domain will get added to this network.
type Rule struct {
	NetworkID NetworkID
	Type      string
	Value     string
}

type NetRequest struct {
	Requester User      `json:"requester"`
	ReqTime   time.Time `json:"requested-at"`
	Status    string    `json:"status"`
}
