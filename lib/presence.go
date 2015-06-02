package lib

import (
	"fmt"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/events"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

//Presences handles users' presence.
type Presences struct {
	broker *events.Broker
}

//Presence represents a user's presence (how recently they were online, and on which form factor) within the app.
type Presence struct {
}

type presenceEvent struct {
	UserID gp.UserID `json:"user"`
	Form   string    `json:"form"`
	At     time.Time `json:"at"`
}

//InvalidFormFactor occurs when a client attempts to register Presence with an unsupported form factor.
var InvalidFormFactor = gp.APIerror{Reason: "Form must be either 'desktop' or 'mobile'"}

//Broadcast sends this user's presence to all conversations they participate in.
func (p Presences) Broadcast(userID gp.UserID, FormFactor string) error {
	if FormFactor != "desktop" && FormFactor != "mobile" {
		return InvalidFormFactor
	}
	//TODO: Write to redis (formFactor, time.Now())
	people, err := api.everyConversationParticipants(userID)
	if err != nil {
		log.Println(err)
		return
	}
	var chans []string
	for _, u := range people {
		chans = append(chans, fmt.Sprintf("c:%d", u))
	}
	event := presenceEvent{UserID: userID, Form: FormFactor, At: time.Now()}
	go p.broker.PublishEvent("presence", userURL(userID), event, chans)
	return nil
}

func userURL(userID gp.UserID) (url string) {
	return fmt.Sprintf("/user/%d", userID)
}

func (p Presences) getPresence(userID gp.UserID) (presence Presence, err error) {

	return
}
