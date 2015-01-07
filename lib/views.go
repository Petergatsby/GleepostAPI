package lib

import (
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//RecordViews saves a bunch of post views.
func (api *API) RecordViews(views ...gp.PostView) {
	//Purge views the user couldn't have made
	views = api.verifyViews(views...)
	err := api.db.RecordViews(views...)
	if err != nil {
		log.Println(err)
	}
	//Publish
	go api.publishNewViewCounts(views...)
}

func (api *API) verifyViews(views ...gp.PostView) (verified []gp.PostView) {
	verified = make([]gp.PostView, 0)
	for _, v := range views {
		p, err := api.getPostFull(v.User, v.Post)
		if err != nil {
			log.Println(err)
		}
		in, err := api.UserInNetwork(v.User, p.Network)
		if in && err == nil {
			verified = append(verified, v)
		}
	}
	return verified
}

func (api *API) publishNewViewCounts(views ...gp.PostView) {
	done := make(map[gp.PostID]bool)
	counts := make([]gp.PostViewCount, 0)

	for _, v := range views {
		_, ok := done[v.Post]
		if !ok {
			count, err := api.db.PostViewCount(v.Post)
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
