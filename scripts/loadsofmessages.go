package main

import (
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func main() {
	conv := "https://dev.gleepost.com/api/v1/conversations/2896/messages?id=2783&token=30124bf77a40060e73641c18676d8a40c1620e270e565045f71b6daeea770e3f"
	i := 0
	client := http.Client{}
	for {
		client.PostForm(conv, url.Values{"text": {strconv.Itoa(i)}})
		time.Sleep(1 * time.Second)
		i++
	}
}
