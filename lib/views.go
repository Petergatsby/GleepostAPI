package lib

import (
	"log"

	"github.com/draaglom/GleepostAPI/lib/cache"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/psc"
)

//Viewer handles Views submitted by clients.
type Viewer interface {
	RecordViews(views []gp.PostView)
}

type viewer struct {
	c  *cache.Cache
	sc *psc.StatementCache
}

//RecordViews saves a bunch of post views, after purging views that the user couldn't have done. It also triggers a views-change event on all the posts involved.
func (api *API) RecordViews(views ...gp.PostView) {
	//Purge views the user couldn't have made
	views = api.verifyViews(views...)
	err := api.recordViews(views...)
	if err != nil {
		log.Println(err)
	}
	//Publish
	go api.publishNewViewCounts(views...)
}

func (api *API) verifyViews(views ...gp.PostView) (verified []gp.PostView) {
	verified = make([]gp.PostView, 0)
	for _, v := range views {
		canView, err := api.canViewPost(v.User, v.Post)
		if err != nil {
			log.Println(err)
		}
		if canView {
			verified = append(verified, v)
		}
	}
	return verified
}

func (api *API) publishNewViewCounts(views ...gp.PostView) {
	done := make(map[gp.PostID]bool)
	var counts []gp.PostViewCount

	for _, v := range views {
		_, ok := done[v.Post]
		if !ok {
			count, err := api.postViewCount(v.Post)
			if err != nil {
				log.Println(err)
				continue
			}
			counts = append(counts, gp.PostViewCount{Post: v.Post, Count: count})
			done[v.Post] = true
		}
	}
	go api.cache.PublishViewCounts(counts...)
}

//CanSubscribePosts takes a list of posts that a user wishes to subscribe to and returns the ones they can actually see.
func (api *API) CanSubscribePosts(user gp.UserID, posts []gp.PostID) (subscribable []gp.PostID, err error) {
	subscribable = make([]gp.PostID, 0)
	for _, p := range posts {
		viewable, err := api.canViewPost(user, p)
		if err == nil && viewable {
			subscribable = append(subscribable, p)
		}
	}
	return subscribable, err
}

//RecordViews saves a bunch of post views. You probably want api.RecordViews() instead.
func (api *API) recordViews(views ...gp.PostView) error {
	q := "INSERT INTO post_views (user_id, post_id, ts) VALUES (?, ?, ?)"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return err
	}
	for _, v := range views {
		_, err = s.Exec(v.User, v.Post, v.Time.UTC())
		if err != nil {
			return err
		}
	}
	return nil
}

//PostViewCount returns the number of total views this post has had.
func (api *API) postViewCount(post gp.PostID) (count int, err error) {
	q := "SELECT COUNT(*) FROM post_views WHERE post_id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(post).Scan(&count)
	return
}
