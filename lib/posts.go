package lib

import (
	"log"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

var (
	//EBADTIME happens when you don't provide a well-formed time when looking for live posts.
	EBADTIME = gp.APIerror{Reason: "Could not parse as a time"}
	//CommentTooShort happens if you try to post an empty comment.
	CommentTooShort = gp.APIerror{Reason: "Comment too short"}
	//NoSuchUpload = You tried to attach a URL you didn't upload to tomething
	NoSuchUpload = gp.APIerror{Reason: "That upload doesn't exist"}
)

//GetPost returns a particular Post
func (api *API) GetPost(postID gp.PostID) (post gp.Post, err error) {
	return api.db.GetPost(postID)
}

//UserGetPost returns the post identified by postId, if the user is allowed to access it; otherwise, ENOTALLOWED.
func (api *API) UserGetPost(userID gp.UserID, postID gp.PostID) (post gp.PostFull, err error) {
	canView, err := api.canViewPost(userID, postID)
	switch {
	case err != nil:
		return post, err
	case !canView:
		return post, &ENOTALLOWED
	default:
		return api.getPostFull(userID, postID)
	}
}

func (api *API) canViewPost(userID gp.UserID, postID gp.PostID) (canView bool, err error) {
	p, err := api.getPostFull(userID, postID)
	if err != nil {
		return
	}
	in, err := api.db.UserInNetwork(userID, p.Network)
	return in, err
}

func (api *API) getPostFull(userID gp.UserID, postID gp.PostID) (post gp.PostFull, err error) {
	post.Post, err = api.db.UserGetPost(userID, postID)
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
	post.Attribs, err = api.getPostAttribs(postID)
	if err != nil {
		return
	}
	post.CommentCount = api.getCommentCount(postID)
	post.Comments, err = api.getComments(postID, 0, api.Config.CommentPageSize)
	if err != nil {
		return
	}
	post.LikeCount, post.Likes, err = api.likesAndCount(postID)
	if err != nil {
		return
	}
	if post.By.ID == userID {
		post.ReviewHistory, err = api.db.ReviewHistory(post.ID)
		if err != nil {
			return
		}
	}
	post.Views, err = api.db.PostViewCount(postID)
	if err != nil {
		log.Println(err)
		err = nil
	}
	post.Attending, err = api.db.IsAttending(userID, postID)
	if err != nil {
		log.Println(err)
		err = nil
	}
	return
}

//UserGetLive gets the live events (soonest first, starting from after) from the perspective of userId.
func (api *API) UserGetLive(userID gp.UserID, after string, count int) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	t, enotstringtime := time.Parse(after, time.RFC3339)
	if enotstringtime != nil {
		unix, enotunixtime := strconv.ParseInt(after, 10, 64)
		if enotunixtime != nil {
			err = EBADTIME
			return
		}
		t = time.Unix(unix, 0)
	}
	primary, err := api.db.GetUserUniversity(userID)
	if err != nil {
		return
	}
	return api.getLive(primary.ID, t, count, userID)
}

//getLive returns the first count events happening after after, within network netId.
func (api *API) getLive(netID gp.NetworkID, after time.Time, count int, userID gp.UserID) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	posts, err = api.db.GetLive(netID, after, count)
	if err != nil {
		return
	}
	for i := range posts {
		processed, err := api.postProcess(posts[i], userID)
		if err == nil {
			posts[i] = processed
		}
	}
	return
}

//GetUserPosts returns the count most recent posts by userId since post `after`.
func (api *API) GetUserPosts(userID gp.UserID, perspective gp.UserID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	posts, err = api.db.GetUserPosts(userID, perspective, mode, index, count, category)
	if err != nil {
		return
	}
	for i := range posts {
		processed, err := api.postProcess(posts[i], perspective)
		if err == nil {
			posts[i] = processed
		}
	}
	return
}

//UserGetPrimaryNetworkPosts returns the posts in the user's primary network (ie, their university)
func (api *API) UserGetPrimaryNetworkPosts(userID gp.UserID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	primary, err := api.db.GetUserUniversity(userID)
	if err != nil {
		return
	}
	return api.UserGetNetworkPosts(userID, primary.ID, mode, index, count, category)
}

//UserGetNetworkPosts returns the posts in netId if userId can access it, or ENOTALLOWED otherwise.
func (api *API) UserGetNetworkPosts(userID gp.UserID, netID gp.NetworkID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	in, err := api.db.UserInNetwork(userID, netID)
	switch {
	case err != nil:
		return posts, err
	case !in:
		return posts, &ENOTALLOWED
	default:
		return api.getPosts(netID, mode, index, count, category, userID)
	}
}

func (api *API) getPosts(netID gp.NetworkID, mode int, index int64, count int, category string, userID gp.UserID) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	posts, err = api.db.GetPosts(netID, mode, index, count, category)
	if err != nil {
		return
	}
	for i := range posts {
		processed, err := api.postProcess(posts[i], userID)
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
	posts = make([]gp.PostSmall, 0)
	posts, err = api.db.UserGetGroupsPosts(user, mode, index, count, category)
	if err != nil {
		return
	}
	for i := range posts {
		processed, err := api.postProcess(posts[i], user)
		if err == nil {
			posts[i] = processed
		} else {
			log.Println(err)
			err = nil
		}
	}
	return
}

//postProcess fetches all the parts of a post which the newGetPosts style methods don't provide
func (api *API) postProcess(post gp.PostSmall, userID gp.UserID) (processed gp.PostSmall, err error) {
	//Ha! I am so funny...
	processed = post
	processed.Likes, err = api.getLikes(processed.ID)
	if err != nil {
		return
	}
	processed.Attribs, err = api.getPostAttribs(processed.ID)
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
	processed.Views, err = api.db.PostViewCount(processed.ID)
	if err != nil {
		log.Println(err)
		err = nil
	}
	processed.Attending, err = api.db.IsAttending(userID, processed.ID)
	return processed, nil
}

//PostSmall turns a PostCore (minimal detail Post) into a PostSmall (full detail but omitting comments).
func (api *API) postSmall(p gp.PostCore) (post gp.PostSmall, err error) {
	post.ID = p.ID
	post.By = p.By
	post.Time = p.Time
	post.Text = p.Text
	post.Images = api.getPostImages(p.ID)
	post.Videos = api.getPostVideos(p.ID)
	post.CommentCount = api.getCommentCount(p.ID)
	post.Categories, err = api.postCategories(p.ID)
	if err != nil {
		return
	}
	post.Attribs, err = api.getPostAttribs(p.ID)
	if err != nil {
		return
	}
	post.LikeCount, post.Likes, err = api.likesAndCount(p.ID)
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
	comments = make([]gp.Comment, 0)
	comments, err = api.cache.GetComments(id, start, count)
	if err != nil {
		comments, err = api.db.GetComments(id, start, count)
		go api.cache.AddAllCommentsFromDB(id, api.db)
	}
	return
}

//UserGetComments returns comments for this post, chronologically ordered starting from the start-th.
//If you are unable to view this post, it will return ENOTALLOWED
func (api *API) UserGetComments(userID gp.UserID, postID gp.PostID, start int64, count int) (comments []gp.Comment, err error) {
	comments = make([]gp.Comment, 0)
	p, err := api.getPostFull(userID, postID)
	if err != nil {
		return
	}
	in, err := api.db.UserInNetwork(userID, p.Network)
	switch {
	case err != nil:
		return comments, err
	case !in:
		log.Printf("User %d not in %d\n", userID, p.Network)
		return comments, &ENOTALLOWED
	default:
		return api.getComments(postID, start, count)
	}
}

//GetCommentCount returns the total number of comments for this post, trying the cache first (so it could be inaccurate)
func (api *API) getCommentCount(id gp.PostID) (count int) {
	count, err := api.cache.GetCommentCount(id)
	if err != nil {
		count = api.db.GetCommentCount(id)
	}
	return count
}

//GetPostImages returns all the images attached to postID.
func (api *API) getPostImages(postID gp.PostID) (images []string) {
	images, _ = api.db.GetPostImages(postID)
	return
}

//GetPostVideos returns all the videos attached to postID.
func (api *API) getPostVideos(postID gp.PostID) (videos []gp.Video) {
	videos, _ = api.db.GetPostVideos(postID)
	return
}

func (api *API) postCategories(post gp.PostID) (categories []gp.PostCategory, err error) {
	return api.db.PostCategories(post)
}

//GetLikes returns all the likes for a particular post.
func (api *API) getLikes(post gp.PostID) (likes []gp.LikeFull, err error) {
	log.Println("GetLikes", post)
	l, err := api.db.GetLikes(post)
	if err != nil {
		return
	}
	for _, like := range l {
		lf := gp.LikeFull{}
		lf.User, err = api.getUser(like.UserID)
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
func (api *API) likesAndCount(post gp.PostID) (count int, likes []gp.LikeFull, err error) {
	likes, err = api.getLikes(post)
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
	in, err := api.db.UserInNetwork(userID, post.Network)
	switch {
	case err != nil:
		return
	case !in:
		err = &ENOTALLOWED
		return
	default:
		commID, err = api.db.CreateComment(postID, userID, text)
		if err == nil {
			user, e := api.getUser(userID)
			if e != nil {
				return commID, e
			}
			comment := gp.Comment{ID: commID, Post: postID, By: user, Time: time.Now().UTC(), Text: text}
			if userID != post.By.ID {
				go api.createNotification("commented", userID, post.By.ID, postID, 0, text)
			}
			go api.cache.AddComment(postID, comment)
		}
		return commID, err
	}
}

//UserAddPostImage adds an image (by url) to a post.
func (api *API) UserAddPostImage(userID gp.UserID, postID gp.PostID, url string) (images []string, err error) {
	post, err := api.GetPost(postID)
	if err != nil {
		return
	}
	exists, err := api.userUploadExists(userID, url)
	if !exists || err != nil {
		return nil, NoSuchUpload
	}
	in, err := api.db.UserInNetwork(userID, post.Network)
	switch {
	case err != nil:
		return
	case !in:
		err = ENOTALLOWED
		return
	case post.By.ID != userID:
		err = ENOTALLOWED
		return
	default:
		err = api.addPostImage(postID, url)
		if err == nil {
			return api.getPostImages(postID), nil
		}
		return
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
func (api *API) UserAddPostVideo(userID gp.UserID, postID gp.PostID, videoID gp.VideoID) (videos []gp.Video, err error) {
	p, err := api.GetPost(postID)
	if err != nil {
		return
	}
	in, err := api.db.UserInNetwork(userID, p.Network)
	switch {
	case err != nil:
		return
	case !in:
		return nil, &ENOTALLOWED
	default:
		err = api.addPostVideo(postID, videoID)
		if err == nil {
			return api.getPostVideos(postID), nil
		}
		return
	}
}

func (api *API) clearPostVideos(postID gp.PostID) (err error) {
	return api.db.ClearPostVideos(postID)
}

func (api *API) needsReview(netID gp.NetworkID, categories ...string) (needsReview bool, err error) {
	level, e := api.db.ApproveLevel(netID)
	switch {
	case e != nil:
		return false, e
	case level.Level == 0:
		return false, nil
	case level.Level == 3: //3 is "all"
		return true, nil
	default:
		for _, tag := range categories {
			for _, filter := range level.Categories {
				if tag == filter {
					return true, nil
				}
			}
		}
		return false, nil
	}

}

//UserAddPostToPrimary creates a post in the user's university.
func (api *API) UserAddPostToPrimary(userID gp.UserID, text string, attribs map[string]string, video gp.VideoID, allowUnowned bool, imageURL string, tags ...string) (postID gp.PostID, pending bool, err error) {
	primary, err := api.db.GetUserUniversity(userID)
	if err != nil {
		return
	}
	return api.UserAddPost(userID, primary.ID, text, attribs, video, allowUnowned, imageURL, tags...)
}

//UserAddPost creates a post in the network netID, with the categories in []tags, or returns an ENOTALLOWED if userID is not a member of netID. If imageURL is set, the post will be created with this image. If allowUnowned, it will allow the post to be created without checking if the user "owns" this image. If video > 0, the post will be created with this video.
func (api *API) UserAddPost(userID gp.UserID, netID gp.NetworkID, text string, attribs map[string]string, video gp.VideoID, allowUnowned bool, imageURL string, tags ...string) (postID gp.PostID, pending bool, err error) {
	in, err := api.db.UserInNetwork(userID, netID)
	switch {
	case err != nil:
		return
	case !in:
		return postID, false, &ENOTALLOWED
	default:
		//If the post matches one of the filters for this network, we want to hide it for now
		pending, err = api.needsReview(netID, tags...)
		postID, err = api.db.AddPost(userID, text, netID, pending)
		if err == nil {
			if len(tags) > 0 {
				err = api.tagPost(postID, tags...)
				if err != nil {
					return
				}
			}
			if len(attribs) > 0 {
				err = api.setPostAttribs(postID, attribs)
				if err != nil {
					return
				}
			}
			if len(imageURL) > 0 {
				var exists bool
				exists, err = api.userUploadExists(userID, imageURL)
				if allowUnowned || (exists && err == nil) {
					err = api.addPostImage(postID, imageURL)
					if err != nil {
						return
					}
				}
			}
			if video > 0 {
				err = api.addPostVideo(postID, video)
				if err != nil {
					return
				}
			}
			_, err := api.db.GetUser(userID)
			if err == nil {
				creator, err := api.userIsNetworkOwner(userID, netID)
				if err == nil && creator && !pending {
					go api.notifyGroupNewPost(userID, netID, postID)
				}
			}
			if pending {
				api.postsToApproveNotification(userID, netID)
			}
		}
		return
	}
}

func (api *API) notifyGroupNewPost(by gp.UserID, group gp.NetworkID, post gp.PostID) {
	users, err := api.db.GetNetworkUsers(group)
	if err != nil {
		log.Println(err)
		return
	}
	for _, u := range users {
		if u.ID != by {
			api.createNotification("group_post", by, u.ID, post, group, "")
		}
	}
	return
}

//TagPost adds these tags/categories to the post if they're not already.
func (api *API) tagPost(post gp.PostID, tags ...string) (err error) {
	//TODO: Only allow the post owner to tag
	return api._tagPost(post, tags...)
}

func (api *API) _tagPost(post gp.PostID, tags ...string) (err error) {
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
	in, err := api.db.UserInNetwork(user, post.Network)
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
			api.createNotification("liked", user, post.By.ID, postID, 0, "")
		}
		return
	}
}

//DelLike idempotently un-likes a post.
func (api *API) DelLike(user gp.UserID, post gp.PostID) (err error) {
	return api.db.RemoveLike(user, post)
}

//setPostAttribs associates a set of key, value pairs with a particular post
func (api *API) setPostAttribs(post gp.PostID, attribs map[string]string) (err error) {
	return api.db.SetPostAttribs(post, attribs)
}

//getPostAttribs returns all the custom attributes of a post.
func (api *API) getPostAttribs(post gp.PostID) (attribs map[string]interface{}, err error) {
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
	in, err := api.db.UserInNetwork(user, post.Network)
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
	events = make([]gp.PostSmall, 0)
	events, err = api.db.UserAttending(perspective, user, category, mode, index, count)
	if err != nil {
		log.Println("Error getting events:", err)
		return
	}
	for i := range events {
		processed, err := api.postProcess(events[i], perspective)
		if err == nil {
			events[i] = processed
		} else {
			log.Println("Error processing events:", err)
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
	p, err := api.getPostFull(user, post)
	switch {
	case err != nil && err == gp.NoSuchPost:
		return nil //You're allowed to "delete" any post which doesn't exist from your perspective.
		//(ie: your posts you've already deleted, for example.)
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

//UserEditPost updates this post with entirely new information. Any fields which aren't set are unchanged.
func (api *API) UserEditPost(userID gp.UserID, postID gp.PostID, text string, attribs map[string]string, url string, videoID gp.VideoID, reason string, tags ...string) (post gp.PostFull, err error) {
	editable, err := api.canEdit(userID, postID)
	switch {
	case err != nil:
		return
	case !editable:
		return post, &ENOTALLOWED
	default:
		if len(text) > 0 {
			err = api.changePostText(postID, text)
			if err != nil {
				return
			}
		}
		//Set attribs
		if len(attribs) > 0 {
			err = api.setPostAttribs(postID, attribs)
			if err != nil {
				return
			}
		}
		if len(url) > 0 {
			err = api.clearPostImages(postID)
			if err != nil {
				return
			}
			_, err = api.UserAddPostImage(userID, postID, url)
			if err != nil {
				return
			}
		}
		if videoID > 0 {
			err = api.clearPostVideos(postID)
			if err != nil {
				return
			}
			//Set new video
			_, err = api.UserAddPostVideo(userID, postID, videoID)
			if err != nil {
				return
			}
		}
		if len(tags) > 0 {
			//Delete and re-set the categories
			err = api.db.ClearCategories(postID)
			if err != nil {
				return
			}
			err = api.tagPost(postID, tags...)
			if err != nil {
				return
			}
		}
	}
	post, err = api.getPostFull(userID, postID)
	if err != nil {
		return
	}
	err = api.maybeResubmitPost(userID, postID, post.Network, reason)
	if err != nil {
		return
	}
	return post, nil
}

func (api *API) canEdit(userID gp.UserID, postID gp.PostID) (editable bool, err error) {
	post, err := api.GetPost(postID)
	if err != nil {
		return
	}
	if post.By.ID == userID {
		return true, nil
	}
	return false, nil
}

//UserGetEventAttendees returns all the attendees of a given event, or ENOTALLOWED if user isn't in its network.
func (api *API) UserGetEventAttendees(user gp.UserID, postID gp.PostID) (attendeeSummary gp.AttendeeSummary, err error) {
	post, err := api.GetPost(postID)
	if err != nil {
		return
	}
	in, err := api.db.UserInNetwork(user, post.Network)
	switch {
	case err != nil || !in:
		return attendeeSummary, ENOTALLOWED
	default:
		attendeeSummary.Attendees, err = api.db.EventAttendees(postID)
		if err != nil {
			return
		}
		attendeeSummary.Popularity, attendeeSummary.AttendeeCount, err = api.userGetEventPopularity(user, postID)
		return
	}
}

//UserGetEventPopularity returns popularity (an arbitrary score between 0 and 100), and the number of attendees. If user isn't in the same network as the event, it will return ENOTALLOWED instead.
func (api *API) userGetEventPopularity(user gp.UserID, postID gp.PostID) (popularity int, attendees int, err error) {
	post, err := api.GetPost(postID)
	if err != nil {
		return
	}
	in, err := api.db.UserInNetwork(user, post.Network)
	switch {
	case err != nil || !in:
		err = ENOTALLOWED
		return
	default:
		return api.db.GetEventPopularity(postID)
	}
}

func (api *API) changePostText(postID gp.PostID, text string) (err error) {
	return api.db.ChangePostText(postID, text)
}

func (api *API) clearPostImages(postID gp.PostID) (err error) {
	return api.db.ClearPostImages(postID)
}

//SubjectiveRSVPCount shows the number of events otherID has attended, from the perspective of the `perspective` user (ie, not counting those events perspective can't see...)
func (api *API) subjectiveRSVPCount(perspective gp.UserID, otherID gp.UserID) (count int, err error) {
	return api.db.SubjectiveRSVPCount(perspective, otherID)
}
