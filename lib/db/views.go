package db

import "github.com/draaglom/GleepostAPI/lib/gp"

func (db *DB) RecordViews(views ...gp.PostView) error {
	q := "INSERT INTO post_views (user_id, post_id, ts) VALUES (?, ?, ?)"
	s, err := db.prepare(q)
	if err != nil {
		return err
	}
	for _, v := range views {
		_, err = s.Exec(v.User, v.Post, v.Time.UTC())
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) PostViewCount(post gp.PostID) (count int, err error) {
	q := "SELECT COUNT(*) FROM post_views WHERE post_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(post).Scan(&count)
	return
}
