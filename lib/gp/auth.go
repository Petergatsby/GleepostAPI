package gp

import "time"

//Status represents a user's current signup state (You should only ever see "unverified" (you have an account pending email verification" or "registered" (this email is taken by someone else)
type Status struct {
	Status string `json:"status"`
	Email  string `json:"email"`
}

func NewStatus(status, email string) Status {
	return Status{Status: status, Email: email}
}

//Token is a gleepost access token.
//TODO: Add scopes?
//TODO: Deprecate in favour of OAuth?
type Token struct {
	UserID UserID    `json:"id"`
	Token  string    `json:"value"`
	Expiry time.Time `json:"expiry"`
}
