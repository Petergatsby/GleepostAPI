package lib

import (
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

type MessageResult struct {
	Messages []MatchedMessage `json:"messages"`
}

type MatchedMessage struct {
	gp.Message
	Matched bool `json:"matched,omitempty"`
}

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
