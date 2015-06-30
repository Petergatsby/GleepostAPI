package gp

type File struct {
	URL     string `json:"url"`
	Type    string `json:"type"`
	Message `json:"message"`
}
