package gp

import "time"

//NotificationID identifies a gleepost notification, eg "John Smith commented on your post!"
type NotificationID uint64

//Notification is a gleepost notification which a user may receive based on other users' actions.
type Notification struct {
	ID      NotificationID `json:"id"`
	Type    string         `json:"type"`
	Time    time.Time      `json:"time"`
	By      User           `json:"user,omitempty"`
	Seen    bool           `json:"seen"`
	Post    PostID         `json:"post,omitempty"`
	Group   NetworkID      `json:"network,omitempty"`
	Preview string         `json:"preview,omitempty"`
}
