package lib

import (
	"fmt"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/mattbaird/elastigo/lib"
)

//MessageResult contains one or more messages which matched a search query, plus their context (a few messages before and after the hit)
type MessageResult struct {
	Messages []MatchedMessage `json:"messages"`
}

//MatchedMessage is a message which (maybe) was a match in a query
type MatchedMessage struct {
	gp.Message
	Matched bool `json:"matched,omitempty"`
}

//SearchMessagesInConversation does exactly what it says on the tin.
func (api *API) SearchMessagesInConversation(userID gp.UserID, convID gp.ConversationID, query string, mode int, index int64) (hits []MessageResult, err error) {
	hits = []MessageResult{
		{
			Messages: []MatchedMessage{
				{Message: gp.Message{ID: 1, Text: "sup", By: gp.User{ID: 1, Name: "patrick"}, Time: time.Now().UTC().Add(-1 * time.Minute)}, Matched: true},
				{Message: gp.Message{ID: 2, Text: "nm u?", By: gp.User{ID: 2, Name: "tade"}, Time: time.Now().UTC()}},
			},
		},
	}
	return hits, nil
}

type esMessage struct {
	gp.Message
	convID gp.ConversationID
}

func (api *API) esIndexMessage(message gp.Message, conversation gp.ConversationID) {
	msg := esMessage{Message: message, convID: conversation}
	c := elastigo.NewConn()
	c.Domain = api.Config.ElasticSearch
	c.Index("gleepost", "messages", fmt.Sprintf("%d", msg.ID), nil, msg)
	return
}
