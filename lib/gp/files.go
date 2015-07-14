package gp

//File is a particular file shared in a conversation.
type File struct {
	URL     string `json:"url"`
	Type    string `json:"type"`
	Message `json:"message"`
}
