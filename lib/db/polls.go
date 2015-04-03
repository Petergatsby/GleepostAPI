package db

import (
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//SavePoll adds this poll to this post.
func (db *DB) SavePoll(postID gp.PostID, pollExpiry time.Time, pollOptions []string) (err error) {
	s, err := db.prepare("INSERT INTO post_polls (post_id, expiry_time) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(postID, pollExpiry.Format(mysqlTime))
	if err != nil {
		return
	}
	s, err = db.prepare("INSERT INTO poll_options (post_id, option_id, `option`) VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	for i, opt := range pollOptions {
		_, err = s.Exec(postID, i, opt)
		if err != nil {
			return
		}
	}
	return
}
