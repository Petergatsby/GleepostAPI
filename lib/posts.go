package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"log"
	"strconv"
	"time"
)

var EBADTIME = gp.APIerror{"Could not parse as a time"}

func (api *API) GetPost(postId gp.PostId) (post gp.Post, err error) {
	return api.db.GetPost(postId)
}

//UserGetPost returns the post identified by postId, if the user is allowed to access it; otherwise, ENOTALLOWED.
func (api *API) UserGetPost(userId gp.UserId, postId gp.PostId) (post gp.PostFull, err error) {
	p, err := api.getPostFull(postId)
	if err != nil {
		return
	}
	in, err := api.UserInNetwork(userId, p.Network)
	switch {
	case err != nil:
		return post, err
	case !in:
		log.Printf("User %d not in %d\n", userId, p.Network)
		return post, &ENOTALLOWED
	default:
		return p, nil
	}
}

func (api *API) getPostFull(postId gp.PostId) (post gp.PostFull, err error) {
	post.Post, err = api.GetPost(postId)
	if err != nil {
		return
	}
	post.Categories, err = api.postCategories(postId)
	if err != nil {
		return
	}
	for _, c := range post.Categories {
		if c.Tag == "event" {
			//Squelch the error, since the best way to handle it is for Popularity to be 0 anyway...
			post.Popularity, _ = api.db.GetEventPopularity(postId)
			break
		}
	}
	post.Attribs, err = api.GetPostAttribs(postId)
	if err != nil {
		return
	}
	post.CommentCount = api.GetCommentCount(postId)
	post.Comments, err = api.GetComments(postId, 0, api.Config.CommentPageSize)
	if err != nil {
		return
	}
	post.LikeCount, post.Likes, err = api.LikesAndCount(postId)
	return
}

//UserGetLive gets the live events (soonest first, starting from after) from the perspective of userId.
func (api *API) UserGetLive(userId gp.UserId, after string, count int) (posts []gp.PostSmall, err error) {
	t, enotstringtime := time.Parse(after, time.RFC3339)
	if enotstringtime != nil {
		unix, enotunixtime := strconv.ParseInt(after, 10, 64)
		if enotunixtime != nil {
			err = EBADTIME
			return
		}
		t = time.Unix(unix, 0)

	}
	networks, err := api.GetUserNetworks(userId)
	if err != nil {
		return
	}
	return api.getLive(networks[0].Id, t, count)
}

//getLive returns the first count events happening after after, within network netId.
func (api *API) getLive(netId gp.NetworkId, after time.Time, count int) (posts []gp.PostSmall, err error) {
	posts, err = api.db.GetLive(netId, after, count)
	if err != nil {
		return
	}
	for i, p := range posts {
		p.Likes, err = api.GetLikes(p.Id)
		if err != nil {
			return
		}
		p.Attribs, err = api.GetPostAttribs(p.Id)
		if err != nil {
			return
		}
		p.Categories, err = api.postCategories(p.Id)
		if err != nil {
			return
		}
		for _, c := range p.Categories {
			if c.Tag == "event" {
				//Squelch the error, since the best way to handle it is for Popularity to be 0 anyway...
				p.Popularity, _ = api.db.GetEventPopularity(p.Id)
				break
			}
		}
		posts[i] = p
	}
	return
}

//GetUserPosts returns the count most recent posts by userId since post `after`.
func (api *API) GetUserPosts(userId gp.UserId, perspective gp.UserId, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts, err = api.db.GetUserPosts(userId, perspective, mode, index, count, category)
	for i, p := range posts {
		p.Likes, err = api.GetLikes(p.Id)
		if err != nil {
			return
		}
		p.Attribs, err = api.GetPostAttribs(p.Id)
		if err != nil {
			return
		}
		for _, c := range p.Categories {
			if c.Tag == "event" {
				//Squelch the error, since the best way to handle it is for Popularity to be 0 anyway...
				p.Popularity, _ = api.db.GetEventPopularity(p.Id)
				break
			}
		}
		posts[i] = p
	}
	return
}

//UserGetNetworkPosts returns the posts in netId if userId can access it, or ENOTALLOWED otherwise.
func (api *API) UserGetNetworkPosts(userId gp.UserId, netId gp.NetworkId, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	in, err := api.UserInNetwork(userId, netId)
	switch {
	case err != nil:
		return posts, err
	case !in:
		return posts, &ENOTALLOWED
	default:
		return api.getPosts(netId, mode, index, count, category)
	}
}

func (api *API) getPosts(netId gp.NetworkId, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts, err = api.db.GetPosts(netId, mode, index, count, category)
	for i, _ := range posts {
		posts[i].Likes, err = api.GetLikes(posts[i].Id)
		if err != nil {
			return
		}
		posts[i].Attribs, err = api.GetPostAttribs(posts[i].Id)
		if err != nil {
			return
		}
		for _, c := range posts[i].Categories {
			if c.Tag == "event" {
				//Squelch the error, since the best way to handle it is for Popularity to be 0 anyway...
				posts[i].Popularity, _ = api.db.GetEventPopularity(posts[i].Id)
				break
			}
		}
	}
	return
}

func (api *API) PostSmall(p gp.PostCore) (post gp.PostSmall, err error) {
	post.Id = p.Id
	post.By = p.By
	post.Time = p.Time
	post.Text = p.Text
	post.Images = api.GetPostImages(p.Id)
	post.CommentCount = api.GetCommentCount(p.Id)
	post.Categories, err = api.postCategories(p.Id)
	if err != nil {
		return
	}
	post.Attribs, err = api.GetPostAttribs(p.Id)
	if err != nil {
		return
	}
	post.LikeCount, post.Likes, err = api.LikesAndCount(p.Id)
	if err != nil {
		return
	}
	for _, c := range post.Categories {
		if c.Tag == "event" {
			//Squelch the error, since the best way to handle it is for Popularity to be 0 anyway...
			post.Popularity, _ = api.db.GetEventPopularity(post.Id)
			break
		}
	}
	return
}

func (api *API) GetComments(id gp.PostId, start int64, count int) (comments []gp.Comment, err error) {
	comments, err = api.cache.GetComments(id, start, count)
	if err != nil {
		comments, err = api.db.GetComments(id, start, count)
		go api.cache.AddAllCommentsFromDB(id, api.db)
	}
	return
}

func (api *API) GetCommentCount(id gp.PostId) (count int) {
	count, err := api.cache.GetCommentCount(id)
	if err != nil {
		count = api.db.GetCommentCount(id)
	}
	return count
}

func (api *API) GetPostImages(postId gp.PostId) (images []string) {
	images, _ = api.db.GetPostImages(postId)
	return
}

func (api *API) postCategories(post gp.PostId) (categories []gp.PostCategory, err error) {
	return api.db.PostCategories(post)
}

func (api *API) GetLikes(post gp.PostId) (likes []gp.LikeFull, err error) {
	log.Println("GetLikes", post)
	l, err := api.db.GetLikes(post)
	if err != nil {
		return
	}
	for _, like := range l {
		lf := gp.LikeFull{}
		lf.User, err = api.GetUser(like.UserID)
		if err == nil {
			lf.Time = like.Time
			likes = append(likes, lf)
		} else {
			log.Println("No such user:", like.UserID)
		}
	}
	return
}

func (api *API) hasLiked(user gp.UserId, post gp.PostId) (liked bool, err error) {
	return api.db.HasLiked(user, post)
}

func (api *API) likeCount(post gp.PostId) (count int, err error) {
	return api.db.LikeCount(post)
}

func (api *API) LikesAndCount(post gp.PostId) (count int, likes []gp.LikeFull, err error) {
	likes, err = api.GetLikes(post)
	if err != nil {
		return
	}
	count, err = api.likeCount(post)
	return
}

func (api *API) CreateComment(postId gp.PostId, userId gp.UserId, text string) (commId gp.CommentId, err error) {
	post, err := api.GetPost(postId)
	if err != nil {
		return
	}
	commId, err = api.db.CreateComment(postId, userId, text)
	if err == nil {
		user, e := api.GetUser(userId)
		if e != nil {
			return commId, e
		}
		comment := gp.Comment{Id: commId, Post: postId, By: user, Time: time.Now().UTC(), Text: text}
		if userId != post.By.Id {
			go api.createNotification("commented", userId, post.By.Id, uint64(postId))
		}
		go api.cache.AddComment(postId, comment)
	}
	return commId, err
}

func (api *API) AddPostImage(postId gp.PostId, url string) (err error) {
	return api.db.AddPostImage(postId, url)
}

func (api *API) AddPost(userId gp.UserId, netId gp.NetworkId, text string, attribs map[string]string, tags ...string) (postId gp.PostId, err error) {
	in, err := api.UserInNetwork(userId, netId)
	switch {
	case err != nil:
		return
	case !in:
		return postId, &ENOTALLOWED
	default:
		postId, err = api.db.AddPost(userId, text, netId)
		if err == nil {
			if len(tags) > 0 {
				err = api.TagPost(postId, tags...)
				if err != nil {
					return
				}
			}
			if len(attribs) > 0 {
				err = api.SetPostAttribs(postId, attribs)
				if err != nil {
					return
				}
			}
			user, err := api.db.GetUser(userId)
			if err == nil {
				post := gp.Post{Id: postId, By: user, Text: text, Time: time.Now().UTC()}
				go api.cache.AddPost(post)
				go api.cache.AddPostToNetwork(post, netId)
			}
		}
		return
	}
}

func (api *API) AddPostWithImage(userId gp.UserId, netId gp.NetworkId, text string, attribs map[string]string, image string, tags ...string) (postId gp.PostId, err error) {
	postId, err = api.AddPost(userId, netId, text, attribs, tags...)
	if err != nil {
		return
	}
	exists, err := api.UserUploadExists(userId, image)
	if exists && err == nil {
		err = api.AddPostImage(postId, image)
		return
	}
	return
}

//
func (api *API) TagPost(post gp.PostId, tags ...string) (err error) {
	//TODO: Only allow the post owner to tag
	return api.tagPost(post, tags...)
}

func (api *API) tagPost(post gp.PostId, tags ...string) (err error) {
	//TODO: Stick this shit in cache
	return api.db.TagPost(post, tags...)
}

func (api *API) AddLike(user gp.UserId, postId gp.PostId) (err error) {
	//TODO: add like to redis
	post, err := api.GetPost(postId)
	if err != nil {
		return
	} else {
		err = api.db.CreateLike(user, postId)
		if err != nil {
			return
		} else {
			if user != post.By.Id {
				api.createNotification("liked", user, post.By.Id, uint64(postId))
			}
		}
	}
	return
}

func (api *API) DelLike(user gp.UserId, post gp.PostId) (err error) {
	return api.db.RemoveLike(user, post)
}

//SetPostAttribs associates a set of key, value pairs with a particular post
func (api *API) SetPostAttribs(post gp.PostId, attribs map[string]string) (err error) {
	return api.db.SetPostAttribs(post, attribs)
}

func (api *API) GetPostAttribs(post gp.PostId) (attribs map[string]interface{}, err error) {
	return api.db.GetPostAttribs(post)
}

//Attend adds the user to the "attending" list for this event. It's idempotent, and should only return an error if the database is down.
//The results are undefined for a post which isn't an event.
//(ie: it will work even though it shouldn't, until I can get round to enforcing it.)
func (api *API) Attend(event gp.PostId, user gp.UserId) (err error) {
	//TODO: Can this user actually attend this event? Does this event even exist?
	return api.db.Attend(event, user)
}

//UnAttend removes a user's attendance to an event. Idempotent, returns an error if the DB is down.
func (api *API) UnAttend(event gp.PostId, user gp.UserId) (err error) {
	//TODO: Merge into Attend
	return api.db.UnAttend(event, user)
}

//UserAttends returns all the event IDs that a user is attending.
func (api *API) UserAttends(user gp.UserId) (events []gp.PostId, err error) {
	return api.db.UserAttends(user)
}

//UserGetGroupsPosts returns up to count posts from this user's user-groups (ie, networks which aren't universities). Acts exactly the same as GetPosts in other respects, except that it will also populate the post's Group attribute.
func (api *API) UserGetGroupsPosts(user gp.UserId, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	return api.db.UserGetGroupsPosts(user, mode, index, count, category)
}
