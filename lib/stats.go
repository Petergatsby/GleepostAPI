package lib

import (
	"time"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"log"
)

type Aggregate struct {
	Type Stat
	Start time.Time
	Finish time.Time
	BucketLength time.Duration
	Counts []Bucket
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
var Stats = []Stat{LIKES, COMMENTS, POSTS, VIEWS, RSVPS}

func (api *API) AggregateStatForUser(stat Stat, user gp.UserId, start time.Time, finish time.Time, bucket time.Duration) (stats *Aggregate, err error) {
	stats = new(Aggregate)
	stats.Type = stat
	stats.Start = start
	stats.Finish = finish
	stats.BucketLength = bucket
	for start.Before(finish) {
		end := start.Add(bucket)
		var count int
		switch {
		case stat == LIKES:
			count, err = api.db.LikesForUserBetween(user, start, end)
		case stat == COMMENTS:
		case stat == POSTS:
		case stat == VIEWS:
		case stat == RSVPS:
		default:
			err = gp.APIerror{Reason:"I don't know what that stat is."}
			return
		}
		if err == nil {
			result := Bucket{Start:start, Count: count}
			stats.Counts = append(stats.Counts, result)
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
