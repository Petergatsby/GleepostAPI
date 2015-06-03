package lib

import (
	"fmt"
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

type typingEvent struct {
	UserID gp.UserID `json:"user"`
	Typing bool      `json:"typing"`
}

//UserIsTyping broadcasts this user's typing status to everyone else in this conversation.
func (api *API) UserIsTyping(userID gp.UserID, conversationID gp.ConversationID, typing bool) {
	if !api.userCanViewConversation(userID, conversationID) {
		log.Printf("user %d attempted to send a typing indicator to a disallowed conversation %d\n", userID, conversationID)
		return
	}
	participants, err := api.getParticipants(conversationID, false)
	if err != nil {
		log.Println("Error getting conversation participants:", err)
		return
	}
	event := typingEvent{UserID: userID, Typing: typing}
	var chans []string
	for _, p := range participants {
		if p.ID != userID {
			chans = append(chans, fmt.Sprintf("c:%d", p.ID))
		}
	}
	api.broker.PublishEvent("typing", conversationURI(conversationID), event, chans)
}
