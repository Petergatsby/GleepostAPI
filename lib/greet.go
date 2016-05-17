package lib

import (
	"strings"

	"github.com/Petergatsby/GleepostAPI/lib/gp"
)

var messages = []string{"Welcome to {app}, {firstname}!"}

//ErrInvalidPreset is returned if you try to get a greeting which doesn't exist.
var ErrInvalidPreset = gp.APIerror{Reason: "No such preset message"}

//GreetMe sends a preset welcome message from CampusBot.
func (api *API) GreetMe(userID gp.UserID, n int) (err error) {
	greeterID, err := api.greeterID()
	if err != nil {
		return
	}
	conv, err := api.CreateConversationWith(greeterID, []gp.UserID{userID})
	if err != nil {
		return
	}
	if (n < 0) || (n > (len(messages) - 1)) {
		return ErrInvalidPreset
	}

	user, err := api.getProfile(userID, userID)
	if err != nil {
		return
	}
	names := strings.Split(user.Name, " ")
	first := names[0]
	msg := strings.Replace(messages[n], "{app}", "CampusPal", -1)
	msg = strings.Replace(msg, "{firstname}", first, -1)
	_, err = api.AddMessage(conv.ID, greeterID, msg)
	return
}

func (api *API) greeterID() (greeterID gp.UserID, err error) {
	s, err := api.sc.Prepare("SELECT id FROM users WHERE greeter = 1")
	if err != nil {
		return
	}
	err = s.QueryRow().Scan(&greeterID)
	if err != nil {
		return
	}
	return
}
