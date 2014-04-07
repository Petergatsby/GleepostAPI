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

func (db *DB) UsersSignedUpBetween(start time.Time, finish time.Time) (users []gp.UserId, err error) {
	s, err := db.prepare("SELECT id FROM users WHERE `timestamp` > ? AND `timestamp` < ?")
	if err != nil {
		return
	}
	rows, err := s.Query(start.UTC().Format(mysqlTime), finish.UTC().Format(mysqlTime))
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var u gp.UserId
		err = rows.Scan(&u)
		if err != nil {
			return
		}
		users = append(users, u)
	}
	return
}

func (db *DB) UsersVerifiedBetween(start time.Time, finish time.Time) (users []gp.UserId, err error) {
	s, err := db.prepare("SELECT id FROM users WHERE `verified` = 1 AND `timestamp` > ? AND `timestamp` < ?")
	rows, err := s.Query(start.UTC().Format(mysqlTime), finish.UTC().Format(mysqlTime))
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var u gp.UserId
		err = rows.Scan(&u)
		if err != nil {
			return
		}
		users = append(users, u)
	}
	return
}

func (db *DB) UsersLikedBetween(start time.Time, finish time.Time) (users []gp.UserId, err error) {
	s, err := db.prepare("SELECT DISTINCT user_id FROM post_likes WHERE user_id IN (SELECT id FROM users WHERE `timestamp` > ? AND `timestamp` < ?)")
	if err != nil {
		return
	}
	rows, err := s.Query(start.UTC().Format(mysqlTime), finish.UTC().Format(mysqlTime))
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var u gp.UserId
		err = rows.Scan(&u)
		if err != nil {
			return
		}
		users = append(users, u)
	}
	return
}
