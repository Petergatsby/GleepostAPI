package gp

import "time"

//PostView represents a user who has viewed a particular post at a particular time.
type PostView struct {
	User UserID    `json:"user"`
	Post PostID    `json:"post"`
	Time time.Time `json:"time"`
}

type PostViewCount struct {
	Post  PostID `json:"post,omitempty"`
	Count int    `json:"views"`
}
