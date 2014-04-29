package db

import (
	"database/sql"
	"errors"
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

//CohortSignedUpBetween returns all the users who signed up between start and finish.
func (db *DB) CohortSignedUpBetween(start time.Time, finish time.Time) (users []gp.UserId, err error) {
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

//UsersVerifiedInCohort returns all the users who have verified their account in the cohort signed up between start and finish.
func (db *DB) UsersVerifiedInCohort(start time.Time, finish time.Time) (users []gp.UserId, err error) {
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

//UsersActivityInCohort returns all the users in the cohort (see CohortSignedUpBetween) who performed this activity, where activity is one of: liked, commented, posted, attended, initiated, messaged
func (db *DB) UsersActivityInCohort(activity string, start time.Time, finish time.Time) (users []gp.UserId, err error) {
	var s *sql.Stmt
	switch {
	case activity == "liked":
		s, err = db.prepare("SELECT DISTINCT user_id FROM post_likes WHERE user_id IN (SELECT id FROM users WHERE `timestamp` > ? AND `timestamp` < ?)")
	case activity == "commented":
		s, err = db.prepare("SELECT DISTINCT `by` FROM post_comments WHERE `by` IN (SELECT id FROM users WHERE `timestamp` > ? AND `timestamp` < ?)")
	case activity == "posted":
		s, err = db.prepare("SELECT DISTINCT `by` FROM wall_posts WHERE `by` IN (SELECT id FROM users WHERE `timestamp` > ? AND `timestamp` < ?)")
	case activity == "attended":
		s, err = db.prepare("SELECT DISTINCT `user_id` FROM event_attendees WHERE `user_id` IN (SELECT id FROM users WHERE `timestamp` > ? AND `timestamp` < ?)")
	case activity == "initiated":
		s, err = db.prepare("SELECT DISTINCT `initiator` FROM conversations WHERE `initiator` IN (SELECT id FROM users WHERE `timestamp` > ? AND `timestamp` < ?)")
	case activity == "messaged":
		s, err = db.prepare("SELECT DISTINCT `from` FROM chat_messages WHERE `from` IN (SELECT id FROM users WHERE `timestamp` > ? AND `timestamp` < ?)")
	default:
		err = errors.New("No such activity")
		return
	}
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
