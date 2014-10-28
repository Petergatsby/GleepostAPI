package lib

import (
	"log"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//EBADTIME happens when you don't provide a well-formed time when looking for live posts.
var EBADTIME = gp.APIerror{Reason: "Could not parse as a time"}

//CommentTooShort happens if you try to post an empty comment.
var CommentTooShort = gp.APIerror{Reason: "Comment too short"}

//GetPost returns a particular Post
func (api *API) GetPost(postID gp.PostID) (post gp.Post, err error) {
	return api.db.GetPost(postID)
}

//UserGetPost returns the post identified by postId, if the user is allowed to access it; otherwise, ENOTALLOWED.
func (api *API) UserGetPost(userID gp.UserID, postID gp.PostID) (post gp.PostFull, err error) {
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

func (api *API) getPostFull(postID gp.PostID) (post gp.PostFull, err error) {
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
	post.Comments, err = api.getComments(postID, 0, api.Config.CommentPageSize)
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
	return api.getLive(networks[0].ID, t, count)
}

//getLive returns the first count events happening after after, within network netId.
func (api *API) getLive(netID gp.NetworkID, after time.Time, count int) (posts []gp.PostSmall, err error) {
	posts, err = api.db.GetLive(netID, after, count)
	if err != nil {
		return
	}
	for i := range posts {
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
	for i := range posts {
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
	for i := range posts {
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
func (api *API) UserGetGroupsPosts(user gp.UserID, mode int, index int64, count int, category string) (posts gp.PostSmallList, err error) {
	posts, err = api.db.UserGetGroupsPosts(user, mode, index, count, category)
	if err != nil {
		return
	}
	for i := range posts {
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

//PostProcess fetches all the parts of a post which the newGetPosts style methods don't provide
func (api *API) PostProcess(post gp.PostSmall) (processed gp.PostSmall, err error) {
	//Ha! I am so funny...
	processed = post
	processed.Likes, err = api.GetLikes(processed.ID)
	if err != nil {
		return
	}
	processed.Attribs, err = api.GetPostAttribs(processed.ID)
	if err != nil {
		return
	}
	processed.Categories, err = api.postCategories(processed.ID)
	if err != nil {
		return
	}
	for _, c := range processed.Categories {
		if c.Tag == "event" {
			//Don't squelch the error, that shit's useful yo
			processed.Popularity, processed.Attendees, err = api.db.GetEventPopularity(processed.ID)
			if err != nil {
				log.Println(err)
			}
			break
		}
	}
	return processed, nil
}

//PostSmall turns a PostCore (minimal detail Post) into a PostSmall (full detail but omitting comments).
func (api *API) PostSmall(p gp.PostCore) (post gp.PostSmall, err error) {
	post.ID = p.ID
	post.By = p.By
	post.Time = p.Time
	post.Text = p.Text
	post.Images = api.GetPostImages(p.ID)
	post.Videos = api.GetPostVideos(p.ID)
	post.CommentCount = api.GetCommentCount(p.ID)
	post.Categories, err = api.postCategories(p.ID)
	if err != nil {
		return
	}
	post.Attribs, err = api.GetPostAttribs(p.ID)
	if err != nil {
		return
	}
	post.LikeCount, post.Likes, err = api.LikesAndCount(p.ID)
	if err != nil {
		return
	}
	for _, c := range post.Categories {
		if c.Tag == "event" {
			//Squelch the error, since the best way to handle it is for Popularity to be 0 anyway...
			post.Popularity, post.Attendees, _ = api.db.GetEventPopularity(post.ID)
			break
		}
	}
	return
}

//getComments returns comments for this post, chronologically ordered starting from the start-th.
func (api *API) getComments(id gp.PostID, start int64, count int) (comments []gp.Comment, err error) {
	comments, err = api.cache.GetComments(id, start, count)
	if err != nil {
		comments, err = api.db.GetComments(id, start, count)
		go api.cache.AddAllCommentsFromDB(id, api.db)
	}
	return
}

//UserGetComments returns comments for this post, chronologically ordered starting from the start-th.
//If you are unable to view this post, it will return ENOTALLOWED
func (api *API) UserGetComments(user gp.UserID, id gp.PostID, start int64, count int) (comments []gp.Comment, err error) {
	p, err := api.getPostFull(id)
	if err != nil {
		return
	}
	in, err := api.UserInNetwork(user, p.Network)
	switch {
	case err != nil:
		return comments, err
	case !in:
		log.Printf("User %d not in %d\n", user, p.Network)
		return comments, &ENOTALLOWED
	default:
		return api.getComments(id, start, count)
	}
}

//GetCommentCount returns the total number of comments for this post, trying the cache first (so it could be inaccurate)
func (api *API) GetCommentCount(id gp.PostID) (count int) {
	count, err := api.cache.GetCommentCount(id)
	if err != nil {
		count = api.db.GetCommentCount(id)
	}
	return count
}

//GetPostImages returns all the images attached to postID.
func (api *API) GetPostImages(postID gp.PostID) (images []string) {
	images, _ = api.db.GetPostImages(postID)
	return
}

//GetPostVideos returns all the videos attached to postID.
func (api *API) GetPostVideos(postID gp.PostID) (videos []gp.Video) {
	videos, _ = api.db.GetPostVideos(postID)
	return
}

func (api *API) postCategories(post gp.PostID) (categories []gp.PostCategory, err error) {
	return api.db.PostCategories(post)
}

//GetLikes returns all the likes for a particular post.
func (api *API) GetLikes(post gp.PostID) (likes []gp.LikeFull, err error) {
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

func (api *API) hasLiked(user gp.UserID, post gp.PostID) (liked bool, err error) {
	return api.db.HasLiked(user, post)
}

func (api *API) likeCount(post gp.PostID) (count int, err error) {
	return api.db.LikeCount(post)
}

//LikesAndCount retrieves both the likes and the total count of likes for a post.
func (api *API) LikesAndCount(post gp.PostID) (count int, likes []gp.LikeFull, err error) {
	likes, err = api.GetLikes(post)
	if err != nil {
		return
	}
	count, err = api.likeCount(post)
	return
}

//CreateComment adds a comment to a post.
func (api *API) CreateComment(postID gp.PostID, userID gp.UserID, text string) (commID gp.CommentID, err error) {
	if len(text) == 0 {
		err = CommentTooShort
		return
	}
	post, err := api.GetPost(postID)
	if err != nil {
		return
	}
	in, err := api.UserInNetwork(userID, post.Network)
	switch {
	case err != nil:
		return
	case !in:
		err = &ENOTALLOWED
		return
	default:
		commID, err = api.db.CreateComment(postID, userID, text)
		if err == nil {
			user, e := api.GetUser(userID)
			if e != nil {
				return commID, e
			}
			comment := gp.Comment{ID: commID, Post: postID, By: user, Time: time.Now().UTC(), Text: text}
			if userID != post.By.ID {
				go api.createNotification("commented", userID, post.By.ID, uint64(postID))
			}
			go api.cache.AddComment(postID, comment)
		}
		return commID, err
	}
}

//UserAddPostImage adds an image (by url) to a post.
func (api *API) UserAddPostImage(userID gp.UserID, postID gp.PostID, url string) (err error) {
	post, err := api.GetPost(postID)
	if err != nil {
		return
	}
	in, err := api.UserInNetwork(userID, post.Network)
	switch {
	case err != nil:
		return
	case !in:
		err = ENOTALLOWED
		return
	default:
		return api.addPostImage(postID, url)
	}
}

func (api *API) addPostImage(postID gp.PostID, url string) (err error) {
	return api.db.AddPostImage(postID, url)
}

//AddPostVideo attaches a URL of a video file to a post.
func (api *API) addPostVideo(postID gp.PostID, videoID gp.VideoID) (err error) {
	return api.db.AddPostVideo(postID, videoID)
}

//UserAddPostVideo attaches a video to a post, or errors if the user isn't allowed.
func (api *API) UserAddPostVideo(userID gp.UserID, postID gp.PostID, videoID gp.VideoID) (err error) {
	p, err := api.GetPost(postID)
	if err != nil {
		return
	}
	in, err := api.UserInNetwork(userID, p.Network)
	switch {
	case err != nil:
		return
	case !in:
		return &ENOTALLOWED
	default:
		return api.addPostVideo(postID, videoID)
	}
}

//AddPost creates a post in the network netID, with the categories in []tags, or returns an ENOTALLOWED if userID is not a member of netID.
func (api *API) AddPost(userID gp.UserID, netID gp.NetworkID, text string, attribs map[string]string, tags ...string) (postID gp.PostID, err error) {
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
				post := gp.Post{ID: postID, By: user, Text: text, Time: time.Now().UTC()}
				go api.cache.AddPost(post)
				go api.cache.AddPostToNetwork(post, netID)
				creator, err := api.UserIsNetworkOwner(userID, netID)
				if err == nil && creator {
					go api.notifyGroupNewPost(userID, netID)
				}
			}
		}
		return
	}
}

func (api *API) notifyGroupNewPost(by gp.UserID, group gp.NetworkID) {
	users, err := api.db.GetNetworkUsers(group)
	if err != nil {
		log.Println(err)
		return
	}
	for _, u := range users {
		if u.ID != by {
			api.createNotification("group_post", by, u.ID, uint64(group))
		}
	}
	return
}

//AddPostWithImage creates a post and adds an image in a single step (if the image is one that has been uploaded to gleepost.)
func (api *API) AddPostWithImage(userID gp.UserID, netID gp.NetworkID, text string, attribs map[string]string, image string, tags ...string) (postID gp.PostID, err error) {
	postID, err = api.AddPost(userID, netID, text, attribs, tags...)
	if err != nil {
		return
	}
	exists, err := api.UserUploadExists(userID, image)
	if exists && err == nil {
		err = api.addPostImage(postID, image)
		if err != nil {
			return
		}
	}
	return
}

//AddPostWithVideo creates a post and attaches a video in a single step.
func (api *API) AddPostWithVideo(userID gp.UserID, netID gp.NetworkID, text string, attribs map[string]string, video gp.VideoID, tags ...string) (postID gp.PostID, err error) {
	postID, err = api.AddPost(userID, netID, text, attribs, tags...)
	if err != nil {
		return
	}
	if video > 0 {
		err = api.addPostVideo(postID, video)
	}
	return
}

//TagPost adds these tags/categories to the post if they're not already.
func (api *API) TagPost(post gp.PostID, tags ...string) (err error) {
	//TODO: Only allow the post owner to tag
	return api.tagPost(post, tags...)
}

func (api *API) tagPost(post gp.PostID, tags ...string) (err error) {
	//TODO: Stick this shit in cache
	return api.db.TagPost(post, tags...)
}

//AddLike likes a post!
func (api *API) AddLike(user gp.UserID, postID gp.PostID) (err error) {
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
		}
		if user != post.By.ID {
			api.createNotification("liked", user, post.By.ID, uint64(postID))
		}
		return
	}
}

//DelLike idempotently un-likes a post.
func (api *API) DelLike(user gp.UserID, post gp.PostID) (err error) {
	return api.db.RemoveLike(user, post)
}

//SetPostAttribs associates a set of key, value pairs with a particular post
func (api *API) SetPostAttribs(post gp.PostID, attribs map[string]string) (err error) {
	return api.db.SetPostAttribs(post, attribs)
}

//GetPostAttribs returns all the custom attributes of a post.
func (api *API) GetPostAttribs(post gp.PostID) (attribs map[string]interface{}, err error) {
	return api.db.GetPostAttribs(post)
}

//UserAttend adds the user to the "attending" list for this event. It's idempotent, and should only return an error if the database is down.
//The results are undefined for a post which isn't an event.
//(ie: it will work even though it shouldn't, until I can get round to enforcing it.)
func (api *API) UserAttend(event gp.PostID, user gp.UserID, attending bool) (err error) {
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

//UserEvents returns all the events that a user is attending.
func (api *API) UserEvents(perspective, user gp.UserID, category string, mode int, index int64, count int) (events []gp.PostSmall, err error) {
	events, err = api.db.UserAttending(perspective, user, category, mode, index, count)
	if err != nil {
		return
	}
	for i := range events {
		processed, err := api.PostProcess(events[i])
		if err == nil {
			events[i] = processed
		}
	}
	return
}

//UserAttends returns all event IDs that a user is attending.
func (api *API) UserAttends(user gp.UserID) (events []gp.PostID, err error) {
	return api.db.UserAttends(user)
}

//UserDeletePost marks a post as deleted (it remains in the db but doesn't show up in feeds). You can only delete your own posts.
func (api *API) UserDeletePost(user gp.UserID, post gp.PostID) (err error) {
	p, err := api.getPostFull(post)
	switch {
	case err != nil:
		return
	case p.By.ID != user:
		return &ENOTALLOWED
	default:
		err = api.deletePost(post)
	}
	return
}

func (api *API) deletePost(post gp.PostID) (err error) {
	return api.db.DeletePost(post)
}

//UserGetEventAttendees returns all the attendees of a given event, or ENOTALLOWED if user isn't in its network.
func (api *API) UserGetEventAttendees(user gp.UserID, postID gp.PostID) (attendees []gp.User, err error) {
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

//UserGetEventPopularity returns popularity (an arbitrary score between 0 and 100), and the number of attendees. If user isn't in the same network as the event, it will return ENOTALLOWED instead.
func (api *API) UserGetEventPopularity(user gp.UserID, postID gp.PostID) (popularity int, attendees int, err error) {
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
