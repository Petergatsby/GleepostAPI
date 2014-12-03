package lib

import (
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//ReportPost records that this user reported this post, with this (optional) reason.
func (api *API) ReportPost(user gp.UserID, post gp.PostID, reason string) error {
	p, err := api.getPostFull(user, post)
	if err != nil {
		return err
	}
	in, err := api.UserInNetwork(user, p.Network)
	switch {
	case err != nil:
		return err
	case !in:
		log.Printf("User %d not in %d\n", user, p.Network)
		return &ENOTALLOWED
	default:
		return api.db.ReportPost(user, post, reason)
	}
}
