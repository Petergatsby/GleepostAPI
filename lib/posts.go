package lib

import (
	"log"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

var EBADTIME = gp.APIerror{Reason: "Could not parse as a time"}

func (api *API) GetPost(postID gp.PostId) (post gp.Post, err error) {
	return api.db.GetPost(postID)
}

//UserGetPost returns the post identified by postId, if the user is allowed to access it; otherwise, ENOTALLOWED.
func (api *API) UserGetPost(userID gp.UserID, postID gp.PostId) (post gp.PostFull, err error) {
	p, err := api.getPostFull(postID)
	if err != nil {
		return
	}
	in, err := api.UserInNetwork(userID, p.Network)
	switch {
	case err != nil:
		return post, err
	case !in:
		log.Printf("User %d not in %d\n", userID, p.Network)
		return post, &ENOTALLOWED
	default:
		return p, nil
	}
}

func (api *API) getPostFull(postID gp.PostId) (post gp.PostFull, err error) {
	post.Post, err = api.GetPost(postID)
	if err != nil {
		return
	}
	post.Categories, err = api.postCategories(postID)
	if err != nil {
		return
	}
	for _, c := range post.Categories {
		if c.Tag == "event" {
			//Don't squelch the error. Those things are useful as it turns out.
			post.Popularity, post.Attendees, err = api.db.GetEventPopularity(postID)
			if err != nil {
				log.Println("Error getting popularity:", err)
			}
			break
		}
	}
	post.Attribs, err = api.GetPostAttribs(postID)
	if err != nil {
		return
	}
	post.CommentCount = api.GetCommentCount(postID)
	post.Comments, err = api.GetComments(postID, 0, api.Config.CommentPageSize)
	if err != nil {
		return
	}
	post.LikeCount, post.Likes, err = api.LikesAndCount(postID)
	return
}

//UserGetLive gets the live events (soonest first, starting from after) from the perspective of userId.
func (api *API) UserGetLive(userID gp.UserID, after string, count int) (posts []gp.PostSmall, err error) {
	t, enotstringtime := time.Parse(after, time.RFC3339)
	if enotstringtime != nil {
		unix, enotunixtime := strconv.ParseInt(after, 10, 64)
		if enotunixtime != nil {
			err = EBADTIME
			return
		}
		t = time.Unix(unix, 0)

	}
	networks, err := api.GetUserNetworks(userID)
	if err != nil {
		return
	}
	return api.getLive(networks[0].Id, t, count)
}

//getLive returns the first count events happening after after, within network netId.
func (api *API) getLive(netID gp.NetworkID, after time.Time, count int) (posts []gp.PostSmall, err error) {
	posts, err = api.db.GetLive(netID, after, count)
	if err != nil {
		return
	}
	for i, _ := range posts {
		processed, err := api.PostProcess(posts[i])
		if err == nil {
			posts[i] = processed
		}
	}
	return
}

//GetUserPosts returns the count most recent posts by userId since post `after`.
func (api *API) GetUserPosts(userID gp.UserID, perspective gp.UserID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts, err = api.db.GetUserPosts(userID, perspective, mode, index, count, category)
	if err != nil {
		return
	}
	for i, _ := range posts {
		processed, err := api.PostProcess(posts[i])
		if err == nil {
			posts[i] = processed
		}
	}
	return
}

//UserGetNetworkPosts returns the posts in netId if userId can access it, or ENOTALLOWED otherwise.
func (api *API) UserGetNetworkPosts(userID gp.UserID, netID gp.NetworkID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	in, err := api.UserInNetwork(userID, netID)
	switch {
	case err != nil:
		return posts, err
	case !in:
		return posts, &ENOTALLOWED
	default:
		return api.getPosts(netID, mode, index, count, category)
	}
}

func (api *API) getPosts(netID gp.NetworkID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts, err = api.db.GetPosts(netID, mode, index, count, category)
	if err != nil {
		return
	}
	for i, _ := range posts {
		processed, err := api.PostProcess(posts[i])
		if err == nil {
			posts[i] = processed
		} else {
			log.Println("Error getting extra details for post:", err)
		}
	}
	return
}

//UserGetGroupsPosts returns up to count posts from this user's user-groups (ie, networks which aren't universities). Acts exactly the same as GetPosts in other respects, except that it will also populate the post's Group attribute.
func (api *API) UserGetGroupsPosts(user gp.UserID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts, err = api.db.UserGetGroupsPosts(user, mode, index, count, category)
	if err != nil {
		return
	}
	for i, _ := range posts {
		processed, err := api.PostProcess(posts[i])
		if err == nil {
			posts[i] = processed
		} else {
			log.Println(err)
			err = nil
		}
	}
	return
}

func (api *API) PostProcess(post gp.PostSmall) (processed gp.PostSmall, err error) {
	//Ha! I am so funny...
	processed = post
	processed.Likes, err = api.GetLikes(processed.Id)
	if err != nil {
		return
	}
	processed.Attribs, err = api.GetPostAttribs(processed.Id)
	if err != nil {
		return
	}
	processed.Categories, err = api.postCategories(processed.Id)
	if err != nil {
		return
	}
	for _, c := range processed.Categories {
		if c.Tag == "event" {
			//Don't squelch the error, that shit's useful yo
			processed.Popularity, processed.Attendees, err = api.db.GetEventPopularity(processed.Id)
			if err != nil {
				log.Println(err)
			}
			break
		}
	}
	return processed, nil
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
			post.Popularity, post.Attendees, _ = api.db.GetEventPopularity(post.Id)
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

func (api *API) GetPostImages(postID gp.PostId) (images []string) {
	images, _ = api.db.GetPostImages(postID)
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

func (api *API) hasLiked(user gp.UserID, post gp.PostId) (liked bool, err error) {
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

func (api *API) CreateComment(postID gp.PostId, userID gp.UserID, text string) (commID gp.CommentId, err error) {
	post, err := api.GetPost(postID)
	if err != nil {
		return
	}
	commID, err = api.db.CreateComment(postID, userID, text)
	if err == nil {
		user, e := api.GetUser(userID)
		if e != nil {
			return commID, e
		}
		comment := gp.Comment{Id: commID, Post: postID, By: user, Time: time.Now().UTC(), Text: text}
		if userID != post.By.Id {
			go api.createNotification("commented", userID, post.By.Id, uint64(postID))
		}
		go api.cache.AddComment(postID, comment)
	}
	return commID, err
}

func (api *API) AddPostImage(postID gp.PostId, url string) (err error) {
	return api.db.AddPostImage(postID, url)
}

func (api *API) AddPost(userID gp.UserID, netID gp.NetworkID, text string, attribs map[string]string, tags ...string) (postID gp.PostId, err error) {
	in, err := api.UserInNetwork(userID, netID)
	switch {
	case err != nil:
		return
	case !in:
		return postID, &ENOTALLOWED
	default:
		postID, err = api.db.AddPost(userID, text, netID)
		if err == nil {
			if len(tags) > 0 {
				err = api.TagPost(postID, tags...)
				if err != nil {
					return
				}
			}
			if len(attribs) > 0 {
				err = api.SetPostAttribs(postID, attribs)
				if err != nil {
					return
				}
			}
			user, err := api.db.GetUser(userID)
			if err == nil {
				post := gp.Post{Id: postID, By: user, Text: text, Time: time.Now().UTC()}
				go api.cache.AddPost(post)
				go api.cache.AddPostToNetwork(post, netID)
			}
		}
		return
	}
}

func (api *API) AddPostWithImage(userID gp.UserID, netID gp.NetworkID, text string, attribs map[string]string, image string, tags ...string) (postID gp.PostId, err error) {
	postID, err = api.AddPost(userID, netID, text, attribs, tags...)
	if err != nil {
		return
	}
	exists, err := api.UserUploadExists(userID, image)
	if exists && err == nil {
		err = api.AddPostImage(postID, image)
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

func (api *API) AddLike(user gp.UserID, postID gp.PostId) (err error) {
	//TODO: add like to redis
	post, err := api.GetPost(postID)
	if err != nil {
		return
	}
	in, err := api.UserInNetwork(user, post.Network)
	switch {
	case err != nil:
		return
	case !in:
		return &ENOTALLOWED
	default:
		err = api.db.CreateLike(user, postID)
		if err != nil {
			return
		} else {
			if user != post.By.Id {
				api.createNotification("liked", user, post.By.Id, uint64(postID))
			}
		}
		return
	}
}

func (api *API) DelLike(user gp.UserID, post gp.PostId) (err error) {
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
func (api *API) UserAttend(event gp.PostId, user gp.UserID, attending bool) (err error) {
	post, err := api.GetPost(event)
	if err != nil {
		return
	}
	in, err := api.UserInNetwork(user, post.Network)
	switch {
	case err != nil || !in:
		err = &ENOTALLOWED
		return
	case attending:
		return api.db.Attend(event, user)
	default:
		return api.db.UnAttend(event, user)
	}
}

//UserAttends returns all the event IDs that a user is attending.
func (api *API) UserAttends(user gp.UserID) (events []gp.PostId, err error) {
	return api.db.UserAttends(user)
}

//UserDeletePost marks a post as deleted (it remains in the db but doesn't show up in feeds). You can only delete your own posts.
func (api *API) UserDeletePost(user gp.UserID, post gp.PostId) (err error) {
	p, err := api.getPostFull(post)
	switch {
	case err != nil:
		return
	case p.By.Id != user:
		return &ENOTALLOWED
	default:
		err = api.deletePost(post)
	}
	return
}

func (api *API) deletePost(post gp.PostId) (err error) {
	return api.db.DeletePost(post)
}

func (api *API) UserGetEventAttendees(user gp.UserID, postID gp.PostId) (attendees []gp.User, err error) {
	post, err := api.GetPost(postID)
	if err != nil {
		return
	}
	in, err := api.UserInNetwork(user, post.Network)
	switch {
	case err != nil || !in:
		return attendees, &ENOTALLOWED
	default:
		return api.db.EventAttendees(postID)
	}
}

func (api *API) UserGetEventPopularity(user gp.UserID, postID gp.PostId) (popularity int, attendees int, err error) {
	post, err := api.GetPost(postID)
	if err != nil {
		return
	}
	in, err := api.UserInNetwork(user, post.Network)
	switch {
	case err != nil || !in:
		err = &ENOTALLOWED
		return
	default:
		return api.db.GetEventPopularity(postID)
	}
}
