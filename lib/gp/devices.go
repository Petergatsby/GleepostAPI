package gp

//Device is a particular (iOS|Android) device owned by a particular user.
type Device struct {
	User UserID `json:"user"`
	Type string `json:"type"`
	ID   string `json:"id"`
}
