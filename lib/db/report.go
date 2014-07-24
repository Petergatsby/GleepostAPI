package db

import "github.com/draaglom/GleepostAPI/lib/gp"

func (db *DB) ReportPost(user gp.UserID, post gp.PostID, reason string) (err error) {
	s, err := db.prepare("INSERT INTO user_reports (reporter_id, type, entity_id, reason) VALUES (?, 'post', ?, ?)")
	if err != nil {
		return
	}
	_, err := s.Exec(user, post, reason)
	return
}
