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
	in, err := api.userInNetwork(user, p.Network)
	switch {
	case err != nil:
		return err
	case !in:
		log.Printf("User %d not in %d\n", user, p.Network)
		return &ENOTALLOWED
	default:
		return api.reportPost(user, post, reason)
	}
}

//ReportPost records that this post has been flagged by user, because of reason.
func (api *API) reportPost(user gp.UserID, post gp.PostID, reason string) (err error) {
	s, err := api.sc.Prepare("REPLACE INTO user_reports (reporter_id, type, entity_id, reason) VALUES (?, 'post', ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(user, post, reason)
	return
}
