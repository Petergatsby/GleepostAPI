package db

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"time"
)

func (db *DB) LikesForUserBetween(user gp.UserId, start time.Time, finish time.Time) (count int, err error) {
	s, err := db.prepare("SELECT COUNT(*) FROM post_likes WHERE post_id IN (SELECT id FROM wall_posts WHERE `by` = ?) AND `timestamp` > ? AND `timestamp` < ?")
	if err != nil {
		return
	}
	err = s.QueryRow(user, start.UTC().Format(mysqlTime), finish.UTC().Format(mysqlTime)).Scan(&count)
	return
}

func (db *DB) CommentsForUserBetween(user gp.UserId, start time.Time, finish time.Time) (count int, err error) {
	s, err := db.prepare("SELECT COUNT(*) FROM post_comments WHERE post_id IN (SELECT id FROM wall_posts WHERE `by` = ?) AND `timestamp` > ? AND `timestamp` < ?")
	if err != nil {
		return
	}
	err = s.QueryRow(user, start.UTC().Format(mysqlTime), finish.UTC().Format(mysqlTime)).Scan(&count)
	return
}

func (db *DB) PostsForUserBetween(user gp.UserId, start time.Time, finish time.Time) (count int, err error) {
	s, err := db.prepare("SELECT COUNT(*) FROM wall_posts WHERE `by` = ? AND `time` > ? AND `time` < ?")
	if err != nil {
		return
	}
	err = s.QueryRow(user, start.UTC().Format(mysqlTime), finish.UTC().Format(mysqlTime)).Scan(&count)
	return
}

func (db *DB) RsvpsForUserBetween(user gp.UserId, start time.Time, finish time.Time) (count int, err error) {
	s, err := db.prepare("SELECT COUNT(*) FROM event_attendees WHERE post_id IN (SELECT id FROM wall_posts WHERE `by` = ?) AND `time` > ? AND `time` < ?")
	if err != nil {
		return
	}
	err = s.QueryRow(user, start.UTC().Format(mysqlTime), finish.UTC().Format(mysqlTime)).Scan(&count)
	return
}
