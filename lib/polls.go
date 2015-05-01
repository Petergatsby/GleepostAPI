package lib

import (
	"fmt"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib/cache"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/go-sql-driver/mysql"
)

var (
	//EndingTooSoon means you tried to start a poll with a very short expiry.
	EndingTooSoon = gp.APIerror{Reason: "Poll ending too soon"}
	//EndingInPast means you tried to start a poll with an expiry in the past.
	EndingInPast = gp.APIerror{Reason: "Poll ending in the past"}
	//EndingTooLate means you tried to start a poll with a very long expiry.
	EndingTooLate = gp.APIerror{Reason: "Poll ending too late"}
	//MissingParameterPollExpiry means you didn't give an expiry when you should have.
	MissingParameterPollExpiry = gp.APIerror{Reason: "Missing parameter: poll-expiry"}
	//TooFewOptions means you specified less than 2 options in a poll.
	TooFewOptions = gp.APIerror{Reason: "Poll: too few options"}
	//TooManyOptions means you specified more than 4 options in a poll.
	TooManyOptions = gp.APIerror{Reason: "Poll: too many options"}
	//NotAPoll means you tried to vote on a post which wasn't a poll.
	NotAPoll = gp.APIerror{Reason: "Not a poll"}
	//InvalidOption means you tried to give an invalid poll option.
	InvalidOption = gp.APIerror{Reason: "Invalid option"}
	//PollExpired means you tried to vote in a poll that has already finished.
	PollExpired = gp.APIerror{Reason: "Poll has already ended"}
	//AlreadyVoted means you tried to vote in a poll that you already voted in.
	AlreadyVoted = gp.APIerror{Reason: "You already voted"}
)

func optionTooAdjective(adj string, n int) gp.APIerror {
	return gp.APIerror{Reason: fmt.Sprintf("Option too %s: %d", adj, n)}
}

func validatePollInput(expiry time.Time, pollOptions []string) (err error) {
	switch {
	case expiry.Before(time.Now()):
		err = EndingInPast
	case expiry.Before(time.Now().Add(15 * time.Minute)):
		err = EndingTooSoon
	case expiry.After(time.Now().AddDate(0, 1, 1)):
		err = EndingTooLate
	case len(pollOptions) < 2:
		err = TooFewOptions
	case len(pollOptions) > 4:
		err = TooManyOptions
	}
	for n, opt := range pollOptions {
		if len(opt) < 3 {
			err = optionTooAdjective("short", n)
		}
		if len(opt) > 50 {
			err = optionTooAdjective("long", n)
		}
	}
	return
}

func (api *API) getPoll(postID gp.PostID) (poll gp.Poll, err error) {
	poll.Expiry, err = api.getPollExpiry(postID)
	if err != nil {
		return
	}
	poll.Options, err = api.getPollOptions(postID)
	if err != nil {
		return
	}
	poll.Votes, err = api.getPollVotes(postID)
	return
}

func (api *API) userGetPoll(userID gp.UserID, postID gp.PostID) (poll gp.SubjectivePoll, err error) {
	poll.Poll, err = api.getPoll(postID)
	if err != nil {
		return
	}
	vote, err := api.getUserVote(userID, postID)
	if err == nil {
		poll.YourVote = vote
	}
	return poll, nil
}

//UserCastVote records this user's vote, or errors if this is not a poll
func (api *API) UserCastVote(userID gp.UserID, postID gp.PostID, option int) (err error) {
	canView, err := api.canViewPost(userID, postID)
	if err != nil || !canView {
		return ENOTALLOWED
	}
	poll, err := api.getPoll(postID)
	if err != nil {
		//Assuming error means not a poll, but could in fact be eg. db down
		return NotAPoll
	}
	if option < 0 || option > len(poll.Options)-1 {
		return InvalidOption
	}
	if time.Now().After(poll.Expiry) {
		return PollExpired
	}
	err = api.userCastVote(userID, postID, option)
	if err == nil {
		poll, err = api.getPoll(postID)
		if err == nil {
			go api.cache.PublishEvent("vote", "/posts/"+strconv.Itoa(int(postID)), poll, []string{cache.PostChannel(postID)})
		}
		api.notifObserver.Notify(voteEvent{userID: userID, postID: postID})
	}
	return
}

//SavePoll adds this poll to this post.
func (api *API) savePoll(postID gp.PostID, pollExpiry time.Time, pollOptions []string) (err error) {
	s, err := api.sc.Prepare("INSERT INTO post_polls (post_id, expiry_time) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(postID, pollExpiry.Format(mysqlTime))
	if err != nil {
		return
	}
	s, err = api.sc.Prepare("INSERT INTO poll_options (post_id, option_id, `option`) VALUES (?, ?, ?)")
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
func (api *API) getPollExpiry(postID gp.PostID) (expiry time.Time, err error) {
	s, err := api.sc.Prepare("SELECT expiry_time FROM post_polls WHERE post_id = ?")
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
func (api *API) getPollOptions(postID gp.PostID) (options []string, err error) {
	s, err := api.sc.Prepare("SELECT `option` FROM poll_options WHERE post_id = ? ORDER BY option_id ASC")
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
func (api *API) getPollVotes(postID gp.PostID) (votes map[string]int, err error) {
	votes = make(map[string]int)
	s, err := api.sc.Prepare("SELECT COUNT(*) as votes, `option` FROM poll_votes JOIN poll_options ON poll_votes.option_id = poll_options.option_id WHERE poll_votes.post_id = ? AND poll_votes.post_id = poll_options.post_id GROUP BY poll_votes.option_id")
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
func (api *API) getUserVote(userID gp.UserID, postID gp.PostID) (vote string, err error) {
	s, err := api.sc.Prepare("SELECT `option` FROM poll_votes JOIN poll_options ON poll_votes.option_id = poll_options.option_id WHERE poll_votes.post_id = ? AND poll_votes.post_id = poll_options.post_id AND poll_votes.user_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(postID, userID).Scan(&vote)
	return
}

//UserCastVote records this user's vote in this poll.
func (api *API) userCastVote(userID gp.UserID, postID gp.PostID, option int) (err error) {
	s, err := api.sc.Prepare("INSERT INTO poll_votes (post_id, option_id, user_id) VALUES (?, ?, ?)")
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
