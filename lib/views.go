package lib

import (
	"database/sql"
	"log"

	"github.com/draaglom/GleepostAPI/lib/cache"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/psc"
)

//Viewer handles Views submitted by clients.
type Viewer interface {
	RecordViews(views []gp.PostView)
	postViewCount(post gp.PostID) (views int, err error)
}

type viewer struct {
	cache *cache.Cache
	sc    *psc.StatementCache
}

//RecordViews saves a bunch of post views, after purging views that the user couldn't have done. It also triggers a views-change event on all the posts involved.
func (v *viewer) RecordViews(views []gp.PostView) {
	views = v.verifyViews(views)
	err := v.recordViews(views)
	if err != nil {
		log.Println("Error recording views:", err)
		return
	}
	go v.publishNewViewCounts(views)
}

func (v *viewer) verifyViews(views []gp.PostView) (verified []gp.PostView) {
	verified = make([]gp.PostView, 0)
	s, err := v.sc.Prepare("SELECT 1 FROM user_network JOIN wall_posts ON user_network.network_id = wall_posts.network_id WHERE user_id = ? AND wall_posts.id = ? AND wall_posts.deleted = 0")
	if err != nil {
		log.Println(err)
		return
	}
	for _, view := range views {
		var visible bool
		err := s.QueryRow(view.User, view.Post).Scan(&visible)
		switch {
		case visible:
			verified = append(verified, view)
		case err == sql.ErrNoRows:
			//The user couldn't see this post
		default:
			log.Println("Error verifying view:", err)
		}
	}
	return
}

func (v *viewer) recordViews(views []gp.PostView) (err error) {
	s, err := v.sc.Prepare("INSERT INTO post_views (user_id, post_id, ts) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	for _, view := range views {
		_, err = s.Exec(view.User, view.Post, view.Time.UTC())
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *viewer) publishNewViewCounts(views []gp.PostView) {
	done := make(map[gp.PostID]bool)
	var counts []gp.PostViewCount

	for _, view := range views {
		_, ok := done[view.Post]
		if !ok {
			count, err := v.postViewCount(view.Post)
			if err != nil {
				log.Println(err)
				continue
			}
			counts = append(counts, gp.PostViewCount{Post: view.Post, Count: count})
			done[view.Post] = true
		}
	}
	go v.cache.PublishViewCounts(counts...)
}

//PostViewCount returns the number of total views this post has had.
func (v *viewer) postViewCount(post gp.PostID) (count int, err error) {
	q := "SELECT COUNT(*) FROM post_views WHERE post_id = ?"
	s, err := v.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(post).Scan(&count)
	return
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
