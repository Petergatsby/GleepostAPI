package db

import (
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/go-sql-driver/mysql"
)

var (
	//AlreadyVoted means you tried to vote in a poll that you already voted in.
	AlreadyVoted = gp.APIerror{Reason: "You already voted"}
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

//GetPollExpiry returns this poll's expiry time -- or err if this is not a poll.
func (db *DB) GetPollExpiry(postID gp.PostID) (expiry time.Time, err error) {
	s, err := db.prepare("SELECT expiry_time FROM post_polls WHERE post_id = ?")
	if err != nil {
		return
	}
	var t string
	err = s.QueryRow(postID).Scan(&t)
	if err != nil {
		return
	}
	expiry, err = time.Parse(mysqlTime, t)
	return
}

//GetPollOptions returns this poll's options -- or err if this is not a poll.
func (db *DB) GetPollOptions(postID gp.PostID) (options []string, err error) {
	s, err := db.prepare("SELECT `option` FROM poll_options WHERE post_id = ? ORDER BY option_id ASC")
	if err != nil {
		return
	}
	rows, err := s.Query(postID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var opt string
		err = rows.Scan(&opt)
		if err != nil {
			return
		}
		options = append(options, opt)
	}
	return
}

//GetPollVotes returns a map of option-name:vote count for this poll, or err if this is not a poll.
func (db *DB) GetPollVotes(postID gp.PostID) (votes map[string]int, err error) {
	votes = make(map[string]int)
	s, err := db.prepare("SELECT COUNT(*) as votes, `option` FROM poll_votes JOIN poll_options ON poll_votes.option_id = poll_options.option_id WHERE poll_votes.post_id = ? AND poll_votes.post_id = poll_options.post_id GROUP BY poll_votes.option_id")
	if err != nil {
		return
	}
	rows, err := s.Query(postID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var option string
		var count int
		err = rows.Scan(&count, &option)
		if err != nil {
			return
		}
		votes[option] = count
	}
	return
}

//GetUserVote returns the way this user voted in this poll.
func (db *DB) GetUserVote(userID gp.UserID, postID gp.PostID) (vote string, err error) {
	s, err := db.prepare("SELECT `option` FROM poll_votes JOIN poll_options ON poll_votes.option_id = poll_options.option_id WHERE poll_votes.post_id = ? AND poll_votes.post_id = poll_options.post_id AND poll_votes.user_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(postID, userID).Scan(&vote)
	return
}

//UserCastVote records this user's vote in this poll.
func (db *DB) UserCastVote(userID gp.UserID, postID gp.PostID, option int) (err error) {
	s, err := db.prepare("INSERT INTO poll_votes (post_id, option_id, user_id) VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(postID, option, userID)
	if err != nil {
		if err, ok := err.(*mysql.MySQLError); ok {
			if err.Number == 1062 {
				return AlreadyVoted
			}
		}
	}
	return
}
