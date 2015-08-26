package lib

import "github.com/draaglom/GleepostAPI/lib/gp"

type MessageResult struct {
	Messages []MatchedMessage `json:"messages"`
}

type MatchedMessage struct {
	gp.Message
	Matched bool `json:"matched,omitempty"`
}

func (api *API) SearchMessagesInConversation(userID gp.UserID, convID gp.ConversationID, query string, mode int, index int64) (hits []MessageResult, err error) {
	return hits, nil
}
