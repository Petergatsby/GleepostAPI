package lib

import (
	"fmt"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//View represents a series of statistics about something over time.
type View struct {
	Start        time.Time         `json:"start"`
	Finish       time.Time         `json:"finish"`
	BucketLength time.Duration     `json:"period"`
	Series       map[Stat][]Bucket `json:"data"`
}

func newView() *View {
	view := new(View)
	Series := make(map[Stat][]Bucket)
	view.Series = Series
	return view
}

//Bucket represents an event count in the period beginning at Start. The length of the period will be in the View this bucket is a member of.
type Bucket struct {
	Start time.Time `json:"start"`
	Count int       `json:"count"`
}

//Stat is a type of event.
type Stat string

//LIKES - nubmer of likes a given entity has received.
const LIKES Stat = "likes"

//COMMENTS - nubmer of comments a given entity has received.
const COMMENTS Stat = "comments"

//POSTS - nubmer of posts a given entity has created.
const POSTS Stat = "posts"

//VIEWS - number of views a given entity has received:w
const VIEWS Stat = "views"

//RSVPS - Number of people who have attended events.
const RSVPS Stat = "rsvps"

//INTERACTIONS - Sum(LIKES, COMMENTS, RSVPS)
const INTERACTIONS Stat = "interactions"

//OVERVIEW - all the available stats together.
const OVERVIEW Stat = "overview"

//Used for OVERVIEW
var Stats = []Stat{LIKES, COMMENTS, VIEWS, RSVPS, POSTS}

func blankF(user gp.UserID, start time.Time, finish time.Time) (count int, err error) {
	return 0, nil
}

func blankPF(post gp.PostID, start time.Time, finish time.Time) (count int, err error) {
	return 0, nil
}

//AggregateStatsForUser aggregates the given Stat in the period between start and finish, grouped into buckets of length bucket.
//If no stats are given, it will return all.
func (api *API) AggregateStatsForUser(user gp.UserID, start time.Time, finish time.Time, bucket time.Duration, stats ...Stat) (view *View, err error) {
	view = newView()
	view.Start = start.Round(time.Duration(time.Second))
	view.Finish = finish.Round(time.Duration(time.Second))
	view.BucketLength = bucket / time.Second
	if len(stats) == 0 {
		stats = Stats
	}
	for _, stat := range stats {
		start = view.Start

		var statF func(gp.UserID, time.Time, time.Time) (int, error)
		switch {
		case stat == LIKES:
			statF = api.db.LikesForUserBetween
		case stat == COMMENTS:
			statF = api.db.CommentsForUserBetween
		case stat == POSTS:
			statF = api.db.PostsForUserBetween
		case stat == VIEWS:
			statF = blankF
		case stat == RSVPS:
			statF = api.db.RsvpsForUserBetween
		case stat == INTERACTIONS:
			statF = api.InteractionsForUserBetween
		default:
			err = gp.APIerror{Reason: "I don't know what that stat is."}
			return
		}
		var data []Bucket
		for start.Before(finish) {
			end := start.Add(bucket)
			var count int
			count, err = statF(user, start, end)
			if err == nil {
				if count > 0 {
					result := Bucket{Start: start.Round(time.Duration(time.Second)), Count: count}
					data = append(data, result)
				}
			} else {
				log.Println(err)
			}
			start = end
		}
		view.Series[stat] = data
	}
	return
}

//AggregateStatsForPost - Same as AggregateStatsForUser, but for an individual post (therefore POSTS is no longer a valid stat).
func (api *API) AggregateStatsForPost(post gp.PostID, start time.Time, finish time.Time, bucket time.Duration, stats ...Stat) (view *View, err error) {
	view = newView()
	view.Start = start.Round(time.Duration(time.Second))
	view.Finish = finish.Round(time.Duration(time.Second))
	view.BucketLength = bucket / time.Second
	if len(stats) == 0 {
		stats = Stats
	}
	for _, stat := range stats {
		start = view.Start

		var statF func(gp.PostID, time.Time, time.Time) (int, error)
		switch {
		case stat == LIKES:
			statF = api.db.LikesForPostBetween
		case stat == COMMENTS:
			statF = api.db.CommentsForPostBetween
		case stat == VIEWS:
			continue
		case stat == POSTS:
			continue
		case stat == RSVPS:
			statF = api.db.RsvpsForPostBetween
		case stat == INTERACTIONS:
			statF = api.InteractionsForPostBetween
		default:
			err = gp.APIerror{Reason: "I don't know what that stat is."}
			return
		}
		var data []Bucket
		for start.Before(finish) {
			end := start.Add(bucket)
			var count int
			count, err = statF(post, start, end)
			if err == nil {
				if count > 0 {
					result := Bucket{Start: start.Round(time.Duration(time.Second)), Count: count}
					data = append(data, result)
				}
			} else {
				log.Println(err)
			}
			start = end
		}
		view.Series[stat] = data
	}
	return
}

//InteractionsForUserBetween returns the number of interactions this user has received in the period between start and finish.
func (api *API) InteractionsForUserBetween(user gp.UserID, start time.Time, finish time.Time) (count int, err error) {
	likes, err := api.db.LikesForUserBetween(user, start, finish)
	if err != nil {
		return
	}
	comments, err := api.db.CommentsForUserBetween(user, start, finish)
	if err != nil {
		return
	}
	rsvps, err := api.db.RsvpsForUserBetween(user, start, finish)
	if err != nil {
		return
	}
	count = likes + comments + rsvps
	return
}

//InteractionsForPostBetween - the number of interactions this post has received in the period between start and finish.
func (api *API) InteractionsForPostBetween(post gp.PostID, start time.Time, finish time.Time) (count int, err error) {
	likes, err := api.db.LikesForPostBetween(post, start, finish)
	if err != nil {
		return
	}
	comments, err := api.db.CommentsForPostBetween(post, start, finish)
	if err != nil {
		return
	}
	rsvps, err := api.db.RsvpsForPostBetween(post, start, finish)
	if err != nil {
		return
	}
	count = likes + comments + rsvps
	return

}

//ActivatedUsersInCohort finds, among the cohort of users signed up between start and finish, all the users who have performed each activity
//"liked", "commented", "posted", "attended", "initiated", "messaged".
func (api *API) ActivatedUsersInCohort(start time.Time, finish time.Time) (ActiveUsers map[string][]gp.UserID, err error) {
	ActiveUsers = make(map[string][]gp.UserID)
	activities := []string{"liked", "commented", "posted", "attended", "initiated", "messaged"}
	for _, activity := range activities {
		users, err := api.db.UsersActivityInCohort(activity, start, finish)
		if err != nil {
			log.Println("Error getting active cohort:", err)
		} else {
			ActiveUsers[activity] = users
		}
	}
	return
}

func deduplicate(userLists ...[]gp.UserID) (deduplicated []gp.UserID) {
	deduped := make(map[gp.UserID]bool)
	for _, list := range userLists {
		for _, u := range list {
			deduped[u] = true
		}
	}
	for k := range deduped {
		deduplicated = append(deduplicated, k)
	}
	return
}

//SummarizePeriod returns an overview of all the users who have signed up, verified, performed specific actions and performed any action, in a given period.
func (api *API) SummarizePeriod(start time.Time, finish time.Time) (stats map[string]int) {
	statFs := make(map[string]func(time.Time, time.Time) ([]gp.UserID, error))
	stats = make(map[string]int)
	statFs["signups"] = api.db.CohortSignedUpBetween
	statFs["verified"] = api.db.UsersVerifiedInCohort
	for k, f := range statFs {
		users, err := f(start, finish)
		if err != nil {
			log.Printf("Error getting %s: %s\n", k, err)
		} else {
			stats[k] = len(users)
		}
	}
	UsersByActivity, err := api.ActivatedUsersInCohort(start, finish)
	if err != nil {
		return
	}
	usersLists := make([][]gp.UserID, len(UsersByActivity))
	for k, v := range UsersByActivity {
		stats[k] = len(v)
		usersLists = append(usersLists, v)
	}
	stats["activated"] = len(deduplicate(usersLists...))
	return (stats)
}

//SummaryEmail sends out an email to everyone in the Admin group, summarizing what the users have done in this period.
func (api *API) summaryEmail(start time.Time, finish time.Time) {
	stats := api.SummarizePeriod(start, finish)
	title := fmt.Sprintf("Report card for %s - %s\n", start.UTC().Round(time.Hour), finish.UTC().Round(time.Hour))
	var text string
	if stats["signups"] > 0 {
		text = fmt.Sprintf("Signups in this period: %d\n", stats["signups"])
		if stats["verified"] > 0 {
			text += fmt.Sprintf("Of these, %d (%2.1f%%) verified their account\n", stats["verified"], 100*float64(stats["verified"])/float64(stats["signups"]))
			text += fmt.Sprintf("Of these, %d (%2.1f%%) activated their account (performed one of the following actions)\n", stats["activated"], 100*float64(stats["activated"])/float64(stats["verified"]))
			activities := []string{"liked", "commented", "posted", "attended", "initiated", "messaged"}
			for _, activity := range activities {
				text += fmt.Sprintf("%s: %d (%2.1f%%)\n", activity, stats[activity], 100*float64(stats[activity])/float64(stats["verified"]))
			}
		} else {
			text += "Nobody verified their account.\n"
		}
	} else {
		text = "There were no signups in this period :(\n"
	}
	users, err := api.db.GetNetworkUsers(gp.NetworkID(api.Config.Admins))
	if err != nil {
		log.Println(err)
		return
	}
	for _, u := range users {
		email, err := api.getEmail(u.ID)
		if err != nil {
			log.Println(err)
		} else {
			err = api.mail.Send(email, title, text)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

//PeriodicSummary is intended to send out a summary email for each time interval starting from start.
//What it actually does, however, is send an email summarizing the previous day every interval.
func (api *API) PeriodicSummary(start time.Time, interval time.Duration) {
	f := func() {
		api.summaryEmail(time.Now().AddDate(0, 0, -1), time.Now())
		tick := time.Tick(interval)
		for {
			select {
			case <-tick:
				api.summaryEmail(time.Now().AddDate(0, 0, -1), time.Now())
			}
		}
	}

	for {
		if start.After(time.Now()) {
			wait := start.Sub(time.Now())
			time.AfterFunc(wait, f)
			return
		}
		start = start.Add(interval)
	}
}
