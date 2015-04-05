package lib

import (
	"fmt"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
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
)

func optionTooAdjective(adj string, n int) gp.APIerror {
	return gp.APIerror{Reason: fmt.Sprintf("Option too %s: %d", adj, n)}
}

func validatePollInput(tags []string, pollExpiry string, pollOptions []string) (poll bool, expiry time.Time, err error) {
	poll = false
	for _, t := range tags {
		if t == "poll" {
			poll = true
			break
		}
	}
	if !poll {
		return
	}
	expiry, err = time.Parse(time.RFC3339, pollExpiry)
	switch {
	case err != nil:
		err = MissingParameterPollExpiry
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
	poll.Expiry, err = api.db.GetPollExpiry(postID)
	if err != nil {
		return
	}
	poll.Options, err = api.db.GetPollOptions(postID)
	if err != nil {
		return
	}
	poll.Votes, err = api.db.GetPollVotes(postID)
	return
}

func (api *API) userGetPoll(userID gp.UserID, postID gp.PostID) (poll gp.SubjectivePoll, err error) {
	poll.Poll, err = api.getPoll(postID)
	if err != nil {
		return
	}
	vote, err := api.db.GetUserVote(userID, postID)
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
	if option < 0 || option > len(poll.Options) {
		return InvalidOption
	}
	return api.db.UserCastVote(userID, postID, option)
}
