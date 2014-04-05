package db

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"time"
)

func (db *DB) LikesForUserBetween(user gp.UserId, start time.Time, finish time.Time) (count int, err error) {
	s, err := db.prepare("SELECT COUNT(*) FROM post_likes WHERE post_id IN (SELECT id FROM wall_posts WHERE `by` = ?) WHERE `timestamp` > ? AND `timestamp` < ?")
	if err != nil {
		return
	}
	err = s.QueryRow(user, start, finish).Scan(&count)
	return
}
