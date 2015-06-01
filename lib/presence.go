package lib

import (
	"fmt"
	"time"

	"github.com/draaglom/GleepostAPI/lib/cache"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

type Presences struct {
	cache *cache.Cache
}

type Presence struct {
}

type presenceEvent struct {
	UserID gp.UserID `json:"user"`
	Form   string    `json:"form"`
	At     time.Time `json:"at"`
}

//InvalidFormFactor occurs when a client attempts to register Presence with an unsupported form factor.
var InvalidFormFactor = gp.APIerror{Reason: "Form must be either 'desktop' or 'mobile'"}

func (p Presences) Broadcast(userID gp.UserID, FormFactor string) error {
	if FormFactor != "desktop" && FormFactor != "mobile" {
		return InvalidFormFactor
	}
	//TODO: Write to redis (formFactor, time.Now())
	chans := ConversationChannelKeys([]gp.User{{ID: userID}})
	event := presenceEvent{UserID: userID, Form: FormFactor, At: time.Now()}
	go p.cache.PublishEvent("presence", userURL(userID), event, chans)
	return nil
}

func userURL(userID gp.UserID) (url string) {
	return fmt.Sprintf("/user/%d", userID)
}

func (p Presences) getPresence(userID gp.UserID) (presence Presence, err error) {

	return
}
