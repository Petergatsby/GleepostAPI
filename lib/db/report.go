package db

import "github.com/draaglom/GleepostAPI/lib/gp"

//ReportPost records that this post has been flagged by user, because of reason.
func (db *DB) ReportPost(user gp.UserID, post gp.PostID, reason string) (err error) {
	s, err := db.prepare("REPLACE INTO user_reports (reporter_id, type, entity_id, reason) VALUES (?, 'post', ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(user, post, reason)
	return
}
