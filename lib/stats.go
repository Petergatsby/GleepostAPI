package lib

import (
	"time"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"log"
)

type Aggregate struct {
	Type Stat `json:"type"`
	Start time.Time `json:"start"`
	Finish time.Time `json:"finish"`
	BucketLength time.Duration `json:"period"`
	Counts []Bucket `json:"data"`
}

type Bucket struct {
	Start time.Time
	Count int
}

type Stat string
const LIKES Stat = "likes"
const COMMENTS Stat = "comments"
const POSTS Stat = "posts"
const VIEWS Stat = "views"
const RSVPS Stat = "rsvps"
const INTERACTIONS Stat = "interactions"
var Stats = []Stat{LIKES, COMMENTS, POSTS, VIEWS, RSVPS}

func (api *API) AggregateStatForUser(stat Stat, user gp.UserId, start time.Time, finish time.Time, bucket time.Duration) (stats *Aggregate, err error) {
	stats = new(Aggregate)
	stats.Type = stat
	stats.Start = start.Round(time.Duration(time.Second))
	stats.Finish = finish.Round(time.Duration(time.Second))
	stats.BucketLength = bucket / time.Second
	var statF func(gp.UserId, time.Time, time.Time) (int, error)
	switch {
	case stat == LIKES:
		statF = api.db.LikesForUserBetween
	case stat == COMMENTS:
		statF = api.db.CommentsForUserBetween
	case stat == POSTS:
		statF = api.db.PostsForUserBetween
	case stat == VIEWS:
	case stat == RSVPS:
		statF = api.db.RsvpsForUserBetween
	case stat == INTERACTIONS:
		statF = api.InteractionsForUserBetween
	default:
		err = gp.APIerror{Reason:"I don't know what that stat is."}
		return
	}
	for start.Before(finish) {
		end := start.Add(bucket)
		var count int
		count, err = statF(user, start, end)
		if err == nil {
			if count > 0 {
				result := Bucket{Start:start.Round(time.Duration(time.Second)), Count: count}
				stats.Counts = append(stats.Counts, result)
			}
		} else {
			log.Println(err)
		}
		start = end
	}
	return
}

func aggregateStatForPost(stat Stat, post gp.PostId, start time.Time, finish time.Time, bucket time.Duration) (stats *Aggregate, err error) {
	return
}

func (api *API) InteractionsForUserBetween(user gp.UserId, start time.Time, finish time.Time) (count int, err error) {
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
