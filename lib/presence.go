package lib

import (
	"fmt"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/events"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/psc"
)

//Presences handles users' presence.
type Presences struct {
	broker *events.Broker
	sc     *psc.StatementCache
	Statsd PrefixStatter
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
	people, err := p.everyConversationParticipants(userID)
	if err != nil {
		log.Println(err)
		return err
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

func (p Presences) getPresence(userID gp.UserID) (presence gp.Presence, err error) {

	return
}

func (p Presences) everyConversationParticipants(user gp.UserID) (participants []gp.UserID, err error) {
	defer p.Statsd.Time(time.Now(), "gleepost.conversations.everyConversationParticipants.db")
	s, err := p.sc.Prepare("SELECT DISTINCT(participant_id) FROM conversation_participants WHERE conversation_id IN (SELECT conversation_id from conversation_participants WHERE participant_id = ? AND deleted = 0)")
	if err != nil {
		return
	}
	rows, err := s.Query(user)
	if err != nil {
		return
	}
	defer rows.Close()
	var u gp.UserID
	for rows.Next() {
		err = rows.Scan(&u)
		if err != nil {
			return
		}
		participants = append(participants, u)
	}
	return
}
