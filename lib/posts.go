package lib

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/psc"
)

const (
	//ByOffsetDescending - This resource will be paginated by starting at a given offset
	ByOffsetDescending = iota
	//ChronologicallyBeforeID - This resource will be paginated by giving the posts immediately chronologically older than this ID.
	ChronologicallyBeforeID
	//ChronologicallyAfterID - This resource will be paginated by giving the posts immediately chronoligically more recent than this ID. The order within the collection will remain the same, however.
	ChronologicallyAfterID
)

var (
	//EBADTIME happens when you don't provide a well-formed time when looking for live posts.
	EBADTIME = gp.APIerror{Reason: "Could not parse as a time"}
	//CommentTooShort happens if you try to post an empty comment.
	CommentTooShort = gp.APIerror{Reason: "Comment too short"}
	//CommentTooLong happens if you try to post a comment with over 1024 chars.
	CommentTooLong = gp.APIerror{Reason: "Comment too long"}
	//NoSuchUpload = You tried to attach a URL you didn't upload to tomething
	NoSuchUpload = gp.APIerror{Reason: "That upload doesn't exist"}
	//PostNoContent = You tried to create a post that does not contain any content
	PostNoContent = gp.APIerror{Reason: "Post contains no content"}
	//InvalidImage = You tried to post with an invalid image
	InvalidImage = gp.APIerror{Reason: "That is not a valid image"}
	//InvalidVideo = You tried to post with an invalid video
	InvalidVideo = gp.APIerror{Reason: "That is not a valid video"}
	//EventInPast = you tried to create an event in the past.
	EventInPast = gp.APIerror{Reason: "Events can not be created in the past"}
	//EventTooLate = you tried to create an event too far in the future.
	EventTooLate = gp.APIerror{Reason: "Events must be within 2 years"}
)

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
	in, err := api.userInNetwork(userID, p.Network)
	return in, err
}

func (api *API) getPostFull(userID gp.UserID, postID gp.PostID) (post gp.PostFull, err error) {
	post.Post, err = api.userGetPost(userID, postID)
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
			post.Popularity, post.Attendees, err = api.getEventPopularity(postID)
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
	post.Comments, err = api.comments.getComments(postID, 0, api.Config.CommentPageSize)
	if err != nil {
		return
	}
	post.LikeCount, post.Likes, err = api.likesAndCount(postID)
	if err != nil {
		return
	}
	if post.By.ID == userID {
		post.ReviewHistory, err = api.reviewHistory(post.ID)
		if err != nil {
			return
		}
	}
	post.Views, err = api.Viewer.postViewCount(postID)
	if err != nil {
		log.Println(err)
		err = nil
	}
	post.Attending, err = api.isAttending(userID, postID)
	if err != nil {
		log.Println(err)
		err = nil
	}
	poll, err := api.userGetPoll(userID, postID)
	if err == nil {
		post.Poll = &poll
	}
	err = nil
	return
}

func parseTime(tstring string) (t time.Time, err error) {
	t, enotstringtime := time.Parse(time.RFC3339, tstring)
	if enotstringtime != nil {
		unix, enotunixtime := strconv.ParseInt(tstring, 10, 64)
		if enotunixtime != nil {
			err = EBADTIME
			return
		}
		t = time.Unix(unix, 0)
	}
	return
}

//UserGetLiveSummary gives a summary of the upcoming events in this user's university.
func (api *API) UserGetLiveSummary(userID gp.UserID, after, until string) (summary gp.LiveSummary, err error) {
	afterTime, err := parseTime(after)
	if err != nil {
		return
	}
	untilTime, err := parseTime(until)
	if err != nil {
		return
	}
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}

	return api.getLiveSummary(primary.ID, afterTime, untilTime)
}

func (api *API) getLiveSummary(netID gp.NetworkID, after, until time.Time) (summary gp.LiveSummary, err error) {
	q := "SELECT COUNT(*) FROM wall_posts " +
		"JOIN post_attribs ON wall_posts.id = post_attribs.post_id " +
		"WHERE deleted = 0 AND pending = 0 AND network_id = ? AND attrib = 'event-time' AND value > ? AND value < ? "
	totalStmt, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = totalStmt.QueryRow(netID, after.Unix(), until.Unix()).Scan(&summary.Posts)
	if err != nil {
		return
	}
	q = "SELECT categories.tag, COUNT(*) FROM wall_posts " +
		"JOIN post_attribs ON wall_posts.id = post_attribs.post_id " + categoryClause +
		"WHERE deleted = 0 AND pending = 0 AND network_id = ? AND attrib = 'event-time' AND value > ? AND value < ? " +
		"GROUP BY categories.tag"
	catsStmt, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	rows, err := catsStmt.Query(netID, after.Unix(), until.Unix())
	if err != nil {
		return
	}
	defer rows.Close()
	summary.CatCounts = make(map[string]int)
	for rows.Next() {
		var cat string
		var count int
		err = rows.Scan(&cat, &count)
		if err != nil {
			return
		}
		summary.CatCounts[cat] = count
	}
	return
}

//UserGetLive gets the live events (soonest first, starting from after) from the perspective of userId.
func (api *API) UserGetLive(userID gp.UserID, after, until string, count int, category string) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	afterTime, err := parseTime(after)
	if err != nil {
		return
	}
	untilTime, err := parseTime(until)
	if err != nil {
		untilTime = time.Now().AddDate(10, 0, 0).UTC()
	}
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	return api.getLive(primary.ID, afterTime, untilTime, count, userID, category)
}

//getLive returns the first count events happening after after, within network netId.
func (api *API) getLive(netID gp.NetworkID, after, until time.Time, count int, userID gp.UserID, category string) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	posts, err = api._getLive(netID, after, until, count, category)
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
	posts, err = api.getUserPosts(userID, perspective, mode, index, count, category)
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
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	return api.UserGetNetworkPosts(userID, primary.ID, mode, index, count, category)
}

//UserGetNetworkPosts returns the posts in netId if userId can access it, or ENOTALLOWED otherwise.
func (api *API) UserGetNetworkPosts(userID gp.UserID, netID gp.NetworkID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	in, err := api.userInNetwork(userID, netID)
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
	posts, err = api._getPosts(netID, mode, index, count, category)
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
	posts, err = api.userGetGroupsPosts(user, mode, index, count, category)
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
			processed.Popularity, processed.Attendees, err = api.getEventPopularity(processed.ID)
			if err != nil {
				log.Println(err)
			}
			break
		}
	}
	processed.Views, err = api.Viewer.postViewCount(processed.ID)
	if err != nil {
		log.Println(err)
		err = nil
	}
	processed.Attending, err = api.isAttending(userID, processed.ID)
	poll, err := api.userGetPoll(userID, processed.ID)
	if err == nil {
		processed.Poll = &poll
	}
	return processed, nil
}

//UserGetComments returns comments for this post, chronologically ordered starting from the start-th.
//If you are unable to view this post, it will return ENOTALLOWED
func (api *API) UserGetComments(userID gp.UserID, postID gp.PostID, start int64, count int) (comments []gp.Comment, err error) {
	comments = make([]gp.Comment, 0)
	p, err := api.getPostFull(userID, postID)
	if err != nil {
		return
	}
	in, err := api.userInNetwork(userID, p.Network)
	switch {
	case err != nil:
		return comments, err
	case !in:
		log.Printf("User %d not in %d\n", userID, p.Network)
		return comments, &ENOTALLOWED
	default:
		return api.comments.getComments(postID, start, count)
	}
}

//GetCommentCount returns the total number of comments for this post
func (api *API) getCommentCount(id gp.PostID) (count int) {
	s, err := api.sc.Prepare("SELECT COUNT(*) FROM post_comments WHERE post_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

//GetPostImages returns all the images attached to postID.
func (api *API) getPostImages(postID gp.PostID) (images []string) {
	defer api.Statsd.Time(time.Now(), "gleepost.postImages.byPostID.db")
	s, err := api.sc.Prepare("SELECT url FROM post_images WHERE post_id = ?")
	if err != nil {
		log.Println(err)
		return
	}
	rows, err := s.Query(postID)
	if err != nil {
		log.Println(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var image string
		err = rows.Scan(&image)
		if err != nil {
			log.Println(err)
			return
		}
		images = append(images, image)
	}
	return
}

//GetPostVideos returns all the videos attached to postID.
func (api *API) getPostVideos(postID gp.PostID) (videos []gp.Video) {
	defer api.Statsd.Time(time.Now(), "gleepost.postVideos.byPostID.db")
	s, err := api.sc.Prepare("SELECT url, mp4_url, webm_url FROM uploads JOIN post_videos ON upload_id = video_id WHERE post_id = ? AND status = 'ready'")
	if err != nil {
		log.Println(err)
		return
	}
	rows, err := s.Query(postID)
	if err != nil {
		log.Println(err)
		return
	}
	defer rows.Close()
	var thumb, mp4, webm sql.NullString
	for rows.Next() {
		err = rows.Scan(&thumb, &mp4, &webm)
		if err != nil {
			log.Println(err)
			return
		}
		video := gp.Video{}
		if mp4.Valid {
			video.MP4 = mp4.String
		}
		if webm.Valid {
			video.WebM = webm.String
		}
		if thumb.Valid {
			video.Thumbs = append(video.Thumbs, thumb.String)
		}
		videos = append(videos, video)
	}
	return
}

//PostCategories returns all the categories which post belongs to.
func (api *API) postCategories(post gp.PostID) (categories []gp.PostCategory, err error) {
	s, err := api.sc.Prepare("SELECT categories.id, categories.tag, categories.name FROM post_categories JOIN categories ON post_categories.category_id = categories.id WHERE post_categories.post_id = ?")
	if err != nil {
		return
	}
	rows, err := s.Query(post)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		c := gp.PostCategory{}
		err = rows.Scan(&c.ID, &c.Tag, &c.Name)
		if err != nil {
			return
		}
		categories = append(categories, c)
	}
	return
}

//GetLikes returns all the likes for a particular post.
func (api *API) getLikes(post gp.PostID) (likes []gp.LikeFull, err error) {
	defer api.Statsd.Time(time.Now(), "gleepost.likes.byPostID.db")
	s, err := api.sc.Prepare("SELECT user_id, timestamp FROM post_likes WHERE post_id = ?")
	if err != nil {
		return
	}
	rows, err := s.Query(post)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		var like gp.LikeFull
		var userID gp.UserID
		err = rows.Scan(&userID, &t)
		if err != nil {
			return
		}
		like.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return
		}
		like.User, err = api.users.byID(userID)
		if err != nil {
			log.Println("Bad like: no such user:", userID)
			continue
		}
		likes = append(likes, like)
	}
	return
}

//HasLiked retuns true if this user has already liked this post.
func (api *API) hasLiked(user gp.UserID, post gp.PostID) (liked bool, err error) {
	s, err := api.sc.Prepare("SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND user_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(post, user).Scan(&liked)
	return
}

//LikeCount returns the number of likes this post has.
func (api *API) likeCount(post gp.PostID) (count int, err error) {
	s, err := api.sc.Prepare("SELECT COUNT(*) FROM post_likes WHERE post_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(post).Scan(&count)
	return
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
	} else if len(text) > 1024 {
		err = CommentTooLong
		return
	}
	post, err := api.getPost(postID)
	if err != nil {
		return
	}
	in, err := api.userInNetwork(userID, post.Network)
	switch {
	case err != nil:
		return
	case !in:
		err = &ENOTALLOWED
		return
	default:
		commID, err = api.createComment(postID, userID, text)
		if err == nil {
			api.notifObserver.Notify(commentEvent{userID: userID, recipientID: post.By.ID, postID: postID, text: text})
			comment := gp.Comment{ID: commID, Post: postID, Time: time.Now().UTC(), Text: text}
			comment.By, err = api.users.byID(userID)
			if err != nil {
				log.Println(err)
				return
			}
			go api.broker.PublishEvent("comment", "/posts/"+strconv.Itoa(int(postID)), comment, []string{PostChannel(postID)})
		}
		return commID, err
	}
}

//UserAddPostImage adds an image (by url) to a post.
func (api *API) UserAddPostImage(userID gp.UserID, postID gp.PostID, url string) (images []string, err error) {
	post, err := api.getPost(postID)
	if err != nil {
		return
	}
	exists, err := api.userUploadExists(userID, url)
	if !exists || err != nil {
		return nil, NoSuchUpload
	}
	in, err := api.userInNetwork(userID, post.Network)
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

//AddPostImage adds an image (url) to postID.
func (api *API) addPostImage(postID gp.PostID, url string) (err error) {
	s, err := api.sc.Prepare("INSERT INTO post_images (post_id, url) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(postID, url)
	return
}

//AddPostVideo attaches a URL of a video file to a post.
func (api *API) addPostVideo(userID gp.UserID, postID gp.PostID, videoID gp.VideoID) (err error) {
	s, err := api.sc.Prepare("INSERT INTO post_videos (post_id, video_id) SELECT ?, upload_id FROM uploads WHERE upload_id = ? AND user_id = ?")
	if err != nil {
		return
	}
	result, err := s.Exec(postID, videoID, userID)
	if err != nil {
		return
	}
	rowsAffected, err := result.RowsAffected()
	if rowsAffected <= 0 {
		err = InvalidVideo
	}
	return
}

//UserAddPostVideo attaches a video to a post, or errors if the user isn't allowed.
func (api *API) UserAddPostVideo(userID gp.UserID, postID gp.PostID, videoID gp.VideoID) (videos []gp.Video, err error) {
	p, err := api.getPost(postID)
	if err != nil {
		return
	}
	in, err := api.userInNetwork(userID, p.Network)
	switch {
	case err != nil:
		return
	case !in:
		return nil, &ENOTALLOWED
	default:
		err = api.addPostVideo(userID, postID, videoID)
		if err == nil {
			return api.getPostVideos(postID), nil
		}
		return
	}
}

//ClearPostVideos deletes all videos from this post.
func (api *API) clearPostVideos(postID gp.PostID) (err error) {
	s, err := api.sc.Prepare("DELETE FROM post_videos WHERE post_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(postID)
	return
}

func (api *API) needsReview(netID gp.NetworkID, categories ...string) (needsReview bool, err error) {
	level, e := api.approveLevel(netID)
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

func hasTag(lookingFor string, tags []string) bool {
	for _, tag := range tags {
		if tag == lookingFor {
			return true
		}
	}
	return false
}

func validatePost(text string, attribs map[string]string, video gp.VideoID, imageURL, pollExpiry string, pollOptions, tags []string) (expiry time.Time, errs []error) {
	if len(text) == 0 && len(attribs["title"]) == 0 && video == 0 && len(imageURL) == 0 {
		errs = append(errs, PostNoContent)
	}
	poll := hasTag("poll", tags)
	if poll {
		var err error
		expiry, err = time.Parse(time.RFC3339, pollExpiry)
		if err != nil {
			_unix, err := strconv.ParseInt(pollExpiry, 10, 64)
			if err != nil {
				errs = append(errs, MissingParameterPollExpiry)
			}
			expiry = time.Unix(_unix, 0)
		}
		err = validatePollInput(expiry, pollOptions)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for attrib, value := range attribs {
		if attrib == "event-time" {
			t, e := time.Parse(time.RFC3339, value)
			if e != nil {
				unixt, e := strconv.ParseInt(value, 10, 64)
				if e != nil {
					errs = append(errs, e)
				}
				t = time.Unix(unixt, 0)
				if t.After(time.Now().AddDate(2, 0, 0)) {
					errs = append(errs, EventTooLate)
				}
				if time.Now().After(t) {
					errs = append(errs, EventInPast)
				}
			}
		}
	}
	return
}

//UserAddPostToPrimary creates a post in the user's university.
func (api *API) UserAddPostToPrimary(userID gp.UserID, text string, attribs map[string]string, video gp.VideoID, allowUnowned bool, imageURL string, pollExpiry string, pollOptions []string, tags ...string) (postID gp.PostID, pending bool, err error) {
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	return api.UserAddPost(userID, primary.ID, text, attribs, video, allowUnowned, imageURL, pollExpiry, pollOptions, tags...)
}

//UserAddPost creates a post in the network netID, with the categories in []tags, or returns an ENOTALLOWED if userID is not a member of netID. If imageURL is set, the post will be created with this image. If allowUnowned, it will allow the post to be created without checking if the user "owns" this image. If video > 0, the post will be created with this video.
func (api *API) UserAddPost(userID gp.UserID, netID gp.NetworkID, text string, attribs map[string]string, video gp.VideoID, allowUnowned bool, imageURL string, pollExpiry string, pollOptions []string, tags ...string) (postID gp.PostID, pending bool, err error) {
	in, err := api.userInNetwork(userID, netID)
	switch {
	case err != nil:
		return
	case !in:
		return postID, false, ENOTALLOWED
	default:
		expiry, errs := validatePost(text, attribs, video, imageURL, pollExpiry, pollOptions, tags)
		if len(errs) > 0 {
			return postID, false, errs[0]
		}
		//If the post matches one of the filters for this network, we want to hide it for now
		pending, err = api.needsReview(netID, tags...)
		if err != nil {
			return
		}
		postID, err = api.addPost(userID, text, netID, pending, tags, attribs)
		if err != nil {
			return
		}
		if len(imageURL) > 0 {
			var exists bool
			exists, err = api.userUploadExists(userID, imageURL)
			if allowUnowned || (exists && err == nil) {
				err = api.addPostImage(postID, imageURL)
				if err != nil {
					return
				}
			} else {
				err = InvalidImage
				return
			}
		}
		if video > 0 {
			err = api.addPostVideo(userID, postID, video)
			if err != nil {
				return
			}
		}
		if hasTag("poll", tags) {
			err = api.savePoll(postID, expiry, pollOptions)
			if err != nil {
				return
			}
		}
		api.notifObserver.Notify(postEvent{userID: userID, netID: netID, postID: postID, pending: pending})
		if pending {
			api.postsToApproveNotification(userID, netID)
		} else {
			post, err := api.getPost(postID)
			if err == nil {
				go api.broker.PublishEvent("post", fmt.Sprintf("/networks/%d/posts", netID), post, []string{NetworkChannel(netID)})
			}
		}
		return
	}
}

//TagPost adds these tags/categories to the post if they're not already.
func (api *API) tagPost(post gp.PostID, tags ...string) (err error) {
	if len(tags) == 0 {
		return
	}
	s, err := api.sc.Prepare("INSERT INTO post_categories( post_id, category_id ) SELECT ? , categories.id FROM categories WHERE categories.tag = ?")
	if err != nil {
		return
	}
	for _, tag := range tags {
		_, err = s.Exec(post, tag)
		if err != nil {
			return
		}
	}
	return
}

//UserSetLike marks a post as "liked" or "unliked" by this user.
func (api *API) UserSetLike(user gp.UserID, postID gp.PostID, liked bool) (err error) {
	post, err := api.getPost(postID)
	if err != nil {
		return
	}
	in, err := api.userInNetwork(user, post.Network)
	switch {
	case err != nil:
		return
	case !in:
		return ENOTALLOWED
	case !liked:
		var s *sql.Stmt
		s, err = api.sc.Prepare("DELETE FROM post_likes WHERE post_id = ? AND user_id = ?")
		if err != nil {
			return
		}
		_, err = s.Exec(postID, user)
		return
	default:
		err = api.createLike(user, postID)
		if err != nil {
			return
		}
		api.notifObserver.Notify(likeEvent{userID: user, recipientID: post.By.ID, postID: postID})
		return
	}
}

//setPostAttribs associates a set of key, value pairs with a particular post
//At the moment, it doesn't check if these attributes are at all reasonable;
//the onus is on the viewer of the attributes to look for just the ones which make sense,
//and on the caller of this function to ensure that the values conform to a particular format.
func (api *API) setPostAttribs(post gp.PostID, attribs map[string]string) (err error) {
	if len(attribs) == 0 {
		return
	}
	s, err := api.sc.Prepare("REPLACE INTO post_attribs (post_id, attrib, value) VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	for attrib, value := range attribs {
		//How could I be so foolish to store time strings rather than unix timestamps...
		if attrib == "event-time" {
			t, e := time.Parse(time.RFC3339, value)
			if e != nil {
				unixt, e := strconv.ParseInt(value, 10, 64)
				if e != nil {
					return e
				}
				t = time.Unix(unixt, 0)
			}
			unix := t.Unix()
			value = strconv.FormatInt(unix, 10)
		}
		_, err = s.Exec(post, attrib, value)
		if err != nil {
			return
		}
	}
	return nil
}

//UserAttend adds the user to the "attending" list for this event. It's idempotent, and should only return an error if the database is down.
//The results are undefined for a post which isn't an event.
//(ie: it will work even though it shouldn't, until I can get round to enforcing it.)
func (api *API) UserAttend(event gp.PostID, user gp.UserID, attending bool) (err error) {
	post, err := api.getPost(event)
	if err != nil {
		return
	}
	in, err := api.userInNetwork(user, post.Network)
	switch {
	case err != nil || !in:
		err = &ENOTALLOWED
		return
	case attending:
		var changed bool
		changed, err = api.attend(event, user)
		if err == nil && changed {
			api.notifObserver.Notify(attendEvent{userID: user, recipientID: post.By.ID, postID: event})
		}
		return
	default:
		return api.unAttend(event, user)
	}
}

//UserEvents returns all the events that a user is attending.
func (api *API) UserEvents(perspective, user gp.UserID, category string, mode int, index int64, count int) (events []gp.PostSmall, err error) {
	events = make([]gp.PostSmall, 0)
	events, err = api.userAttending(perspective, user, category, mode, index, count)
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
	events = make([]gp.PostID, 0)
	query := "SELECT post_id FROM event_attendees WHERE user_id = ?"
	s, err := api.sc.Prepare(query)
	if err != nil {
		return
	}
	rows, err := s.Query(user)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var post gp.PostID
		err = rows.Scan(&post)
		if err != nil {
			return
		}
		events = append(events, post)
	}
	return
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

//DeletePost marks a post as deleted in the database.
func (api *API) deletePost(post gp.PostID) (err error) {
	q := "UPDATE wall_posts SET deleted = 1 WHERE id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(post)
	return
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
			err = api.clearCategories(postID)
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
	post, err := api.getPost(postID)
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
	post, err := api.getPost(postID)
	if err != nil {
		return
	}
	in, err := api.userInNetwork(user, post.Network)
	switch {
	case err != nil || !in:
		return attendeeSummary, ENOTALLOWED
	default:
		attendeeSummary.Attendees, err = api.eventAttendees(postID)
		if err != nil {
			return
		}
		attendeeSummary.Popularity, attendeeSummary.AttendeeCount, err = api.userGetEventPopularity(user, postID)
		return
	}
}

//UserGetEventPopularity returns popularity (an arbitrary score between 0 and 100), and the number of attendees. If user isn't in the same network as the event, it will return ENOTALLOWED instead.
func (api *API) userGetEventPopularity(user gp.UserID, postID gp.PostID) (popularity int, attendees int, err error) {
	post, err := api.getPost(postID)
	if err != nil {
		return
	}
	in, err := api.userInNetwork(user, post.Network)
	switch {
	case err != nil || !in:
		err = ENOTALLOWED
		return
	default:
		return api.getEventPopularity(postID)
	}
}

const (
	//Base
	baseQuery = "SELECT wall_posts.id, `by`, wall_posts.time, text, network_id FROM wall_posts "
	//Joins
	categoryClause = "JOIN post_categories ON wall_posts.id = post_categories.post_id " +
		"JOIN categories ON post_categories.category_id = categories.id "

	attendClause = "JOIN event_attendees ON wall_posts.id = event_attendees.post_id "
	//Wheres
	notDeleted    = "WHERE deleted = 0 "
	notPending    = "AND pending = 0 "
	whereCategory = "AND categories.tag = ? "

	whereBefore = "AND wall_posts.id < ? "
	whereAfter  = "AND wall_posts.id > ? "

	whereBeforeAtt = "AND event_attendees.time < (SELECT time FROM event_attendees WHERE post_id = ? AND user_id = ?) "
	whereAfterAtt  = "AND event_attendees.time < (SELECT time FROM event_attendees WHERE post_id = ? AND user_id = ?) "

	byNetwork = "AND network_id = ? "
	byPoster  = "AND `by` = ? AND network_id IN ( " +
		"SELECT network_id FROM user_network WHERE user_id = ? ) "
	byUserGroups = "AND network_id IN ( " +
		"SELECT network_id FROM user_network " +
		"JOIN network ON user_network.network_id = network.id " +
		"WHERE user_id = ? AND network.user_group = 1 ) "
	byVisibleAttendance = "AND network_id IN ( " +
		"SELECT network_id FROM user_network WHERE user_id = ? ) " +
		"AND event_attendees.user_id = ? "

	//Orders
	orderLinear               = "ORDER BY time DESC, id DESC LIMIT ?, ?"
	orderChronological        = "ORDER BY time DESC, id DESC LIMIT 0, ?"
	orderReverseChronological = "ORDER BY time ASC, id ASC limit 0, ?"

	reverse = "SELECT `id`, `by`, `time`, `text`, `network_id` FROM ( %s ) AS `wp` ORDER BY `time` DESC, `id` DESC"

	orderLinearAttend        = "ORDER BY event_attendees.time DESC, id DESC LIMIT ?, ?"
	orderChronologicalAttend = "ORDER BY event_attendees.time DESC, id DESC LIMIT 0, ?"
)

var (
	//EBADORDER means you tried to order a post query in an unexpected way.
	EBADORDER = gp.APIerror{Reason: "Invalid order clause!"}
)

func (api *API) scanPostRows(rows *sql.Rows, expandNetworks bool) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	for rows.Next() {
		var post gp.PostSmall
		var t string
		var by gp.UserID
		err = rows.Scan(&post.ID, &by, &t, &post.Text, &post.Network)
		if err != nil {
			return posts, err
		}
		post.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return posts, err
		}
		post.By, err = api.users.byID(by)
		if err == nil {
			post.CommentCount = api.getCommentCount(post.ID)
			post.Images = api.getPostImages(post.ID)
			post.Videos = api.getPostVideos(post.ID)
			post.LikeCount, err = api.likeCount(post.ID)
			if err != nil {
				return
			}
			if expandNetworks {
				net, err := api.getNetwork(post.Network)
				if err == nil {
					post.Group = &net
				} else {
					log.Println("Error getting network:", err)
				}
			}
			posts = append(posts, post)
		} else {
			log.Println("Bad post: ", post)
		}
	}
	return posts, nil
}

//GetUserPosts returns the most recent count posts by userId after the post with id after.
func (api *API) getUserPosts(userID, perspective gp.UserID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	var q string
	if len(category) > 0 {
		q = baseQuery + categoryClause + notDeleted + notPending + byPoster + category
	} else {
		q = baseQuery + notDeleted + notPending + byPoster
	}
	switch {
	case mode == ByOffsetDescending:
		q += orderLinear
	case mode == ChronologicallyAfterID:
		q += whereAfter + orderChronological
	case mode == ChronologicallyBeforeID:
		q += whereBefore + orderChronological
	}
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	var rows *sql.Rows
	if len(category) > 0 {
		rows, err = s.Query(userID, perspective, category, index, count)
	} else {
		rows, err = s.Query(userID, perspective, index, count)
	}
	if err != nil {
		return
	}
	defer rows.Close()
	return api.scanPostRows(rows, true)
}

//AddPost creates a post, returning the created ID. It only handles the core of the post; other attributes, images and so on must be created separately.
func (api *API) addPost(userID gp.UserID, text string, network gp.NetworkID, pending bool, tags []string, attribs map[string]string) (postID gp.PostID, err error) {
	s, err := api.sc.Prepare("INSERT INTO wall_posts(`by`, `text`, network_id, pending) VALUES (?,?,?,?)")
	if err != nil {
		return
	}
	res, err := s.Exec(userID, text, network, pending)
	if err != nil {
		return 0, err
	}
	_postID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	postID = gp.PostID(_postID)
	err = api.tagPost(postID, tags...)
	if err != nil {
		return 0, err
	}
	err = api.setPostAttribs(postID, attribs)
	if err != nil {
		return 0, err
	}
	return postID, nil
}

//GetLive returns a list of events whose event time is after "after", ordered by time.
func (api *API) _getLive(netID gp.NetworkID, after time.Time, until time.Time, count int, category string) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	q := "SELECT wall_posts.id, `by`, time, text, network_id " +
		"FROM wall_posts " +
		"JOIN post_attribs ON wall_posts.id = post_attribs.post_id "
	if len(category) > 0 {
		q += categoryClause
	}
	q += "WHERE deleted = 0 AND pending = 0 AND network_id = ? AND attrib = 'event-time' AND value > ? AND value < ? "
	if len(category) > 0 {
		q += whereCategory
	}
	q += "ORDER BY value ASC LIMIT 0, ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	var rows *sql.Rows
	if len(category) > 0 {
		rows, err = s.Query(netID, after.Unix(), until.Unix(), category, count)
	} else {
		rows, err = s.Query(netID, after.Unix(), until.Unix(), count)
	}
	if err != nil {
		return
	}
	defer rows.Close() //The second argument is meaningless and should be removed.
	return api.scanPostRows(rows, false)
}

//GetPosts finds posts in the network netId.
func (api *API) _getPosts(netID gp.NetworkID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	var q string
	if len(category) > 0 {
		q = baseQuery + categoryClause + notDeleted + notPending + byNetwork + whereCategory
	} else {
		q = baseQuery + notDeleted + notPending + byNetwork
	}
	switch {
	case mode == ByOffsetDescending:
		q += orderLinear
	case mode == ChronologicallyAfterID:
		q += whereAfter + orderReverseChronological
		q = fmt.Sprintf(reverse, q)
	case mode == ChronologicallyBeforeID:
		q += whereBefore + orderChronological
	}
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	var rows *sql.Rows
	if len(category) > 0 {
		rows, err = s.Query(netID, category, index, count)
	} else {
		rows, err = s.Query(netID, index, count)
	}
	if err != nil {
		return
	}
	defer rows.Close()
	return api.scanPostRows(rows, false)
}

//ClearPostImages deletes all images from this post.
func (api *API) clearPostImages(postID gp.PostID) (err error) {
	s, err := api.sc.Prepare("DELETE FROM post_images WHERE post_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(postID)
	return
}

//CreateComment adds a comment on this post.
func (api *API) createComment(postID gp.PostID, userID gp.UserID, text string) (commID gp.CommentID, err error) {
	s, err := api.sc.Prepare("INSERT INTO post_comments (post_id, `by`, text) VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	if res, err := s.Exec(postID, userID, text); err == nil {
		cID, err := res.LastInsertId()
		commID = gp.CommentID(cID)
		return commID, err
	}
	return 0, err
}

type comments struct {
	sc    *psc.StatementCache
	stats PrefixStatter
	users *Users
}

//GetComments returns up to count comments for this post.
func (comm comments) getComments(postID gp.PostID, start int64, count int) (comments []gp.Comment, err error) {
	defer comm.stats.Time(time.Now(), "gleepost.comments.byPostID.db")
	comments = make([]gp.Comment, 0)
	q := "SELECT id, `by`, text, `timestamp` " +
		"FROM post_comments " +
		"WHERE post_id = ? " +
		"ORDER BY `timestamp` DESC LIMIT ?, ?"
	s, err := comm.sc.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(postID, start, count)
	if err != nil {
		return comments, err
	}
	defer rows.Close()
	for rows.Next() {
		var comment gp.Comment
		comment.Post = postID
		var timeString string
		var by gp.UserID
		err := rows.Scan(&comment.ID, &by, &comment.Text, &timeString)
		if err != nil {
			return comments, err
		}
		comment.Time, _ = time.Parse(mysqlTime, timeString)
		comment.By, err = comm.users.byID(by)
		if err != nil {
			log.Printf("error getting user %d: %v\n", by, err)
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

//UserGetPost returns the post postId or an error if it doesn't exist.
//TODO: This could return without an embedded user or images array
func (api *API) userGetPost(userID gp.UserID, postID gp.PostID) (post gp.Post, err error) {
	s, err := api.sc.Prepare("SELECT `network_id`, `by`, `time`, text FROM wall_posts WHERE deleted = 0 AND id = ? AND (pending = 0 OR `by` = ?)")
	if err != nil {
		if err == sql.ErrNoRows {
			err = gp.NoSuchPost
		}
		return
	}
	post.ID = postID
	var by gp.UserID
	var t string
	err = s.QueryRow(postID, userID).Scan(&post.Network, &by, &t, &post.Text)
	if err != nil {
		return
	}
	post.By, err = api.users.byID(by)
	if err != nil {
		return
	}
	post.Time, err = time.Parse(mysqlTime, t)
	if err != nil {
		return
	}
	post.Images = api.getPostImages(postID)
	post.Videos = api.getPostVideos(postID)
	return
}

//GetPost returns the post postId or an error if it doesn't exist.
//TODO: This could return without an embedded user or images array
func (api *API) getPost(postID gp.PostID) (post gp.Post, err error) {
	s, err := api.sc.Prepare("SELECT `network_id`, `by`, `time`, text FROM wall_posts WHERE deleted = 0 AND id = ?")
	if err != nil {
		return
	}
	post.ID = postID
	var by gp.UserID
	var t string
	err = s.QueryRow(postID).Scan(&post.Network, &by, &t, &post.Text)
	if err != nil {
		if err == sql.ErrNoRows {
			err = gp.NoSuchPost
		}
		return
	}
	post.By, err = api.users.byID(by)
	if err != nil {
		return
	}
	post.Time, err = time.Parse(mysqlTime, t)
	if err != nil {
		return
	}
	post.Images = api.getPostImages(postID)
	post.Videos = api.getPostVideos(postID)
	return
}

//GetPostAttribs returns a map of all attributes associated with post.
func (api *API) getPostAttribs(post gp.PostID) (attribs map[string]interface{}, err error) {
	s, err := api.sc.Prepare("SELECT attrib, value FROM post_attribs WHERE post_id=?")
	if err != nil {
		return
	}
	rows, err := s.Query(post)
	if err != nil {
		return
	}
	defer rows.Close()
	attribs = make(map[string]interface{})
	for rows.Next() {
		var attrib, val string
		err = rows.Scan(&attrib, &val)
		if err != nil {
			return
		}
		switch {
		case attrib == "event-time":
			var unix int64
			unix, err = strconv.ParseInt(val, 10, 64)
			if err == nil {
				attribs[attrib] = time.Unix(unix, 0)
			}
		default:
			attribs[attrib] = val
		}
	}
	return
}

//GetEventPopularity returns the popularity score (0 - 99) and the actual attendees count
func (api *API) getEventPopularity(post gp.PostID) (popularity int, attendees int, err error) {
	query := "SELECT COUNT(*) FROM event_attendees WHERE post_id = ?"
	s, err := api.sc.Prepare(query)
	if err != nil {
		return
	}
	err = s.QueryRow(post).Scan(&attendees)
	if err != nil {
		return
	}
	switch {
	case attendees > 3:
		popularity = 100
	case attendees > 2:
		popularity = 75
	case attendees > 1:
		popularity = 50
	case attendees > 0:
		popularity = 25
	default:
		popularity = 0
	}
	return
}

//UserGetGroupsPosts retrieves posts from this user's groups (non-university networks)
//TODO: Verify shit doesn't break when a user has no user-groups
func (api *API) userGetGroupsPosts(user gp.UserID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	var q string
	if len(category) > 0 {
		q = baseQuery + categoryClause + notDeleted + notPending + byUserGroups + whereCategory
	} else {
		q = baseQuery + notDeleted + notPending + byUserGroups
	}
	switch {
	case mode == ByOffsetDescending:
		q += orderLinear
	case mode == ChronologicallyAfterID:
		q += whereAfter + orderChronological
	case mode == ChronologicallyBeforeID:
		q += whereBefore + orderChronological
	}
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	var rows *sql.Rows
	if len(category) > 0 {
		rows, err = s.Query(user, category, index, count)
	} else {
		rows, err = s.Query(user, index, count)
	}
	if err != nil {
		return
	}
	defer rows.Close()
	return api.scanPostRows(rows, true)
}

//EventAttendees returns all users who are attending this event.
func (api *API) eventAttendees(post gp.PostID) (attendees []gp.User, err error) {
	q := "SELECT id, firstname, avatar, official FROM users JOIN event_attendees ON user_id = id WHERE post_id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(post)
	if err != nil {
		return
	}
	var avatar sql.NullString
	for rows.Next() {
		var user gp.User
		err = rows.Scan(&user.ID, &user.Name, &avatar, &user.Official)
		if avatar.Valid {
			user.Avatar = avatar.String
		}
		attendees = append(attendees, user)
	}
	return
}

//UserPostCount returns this user's number of posts, from the other user's perspective (ie, only the posts in groups they share).
func (api *API) userPostCount(perspective, user gp.UserID) (count int, err error) {
	q := "SELECT COUNT(*) FROM wall_posts "
	q += "WHERE `by` = ? "
	q += "AND deleted = 0 AND pending = 0 "
	q += "AND network_id IN (SELECT network_id FROM user_network WHERE user_id = ?)"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(user, perspective).Scan(&count)
	return
}

//UserAttending returns all the events this user is attending.
func (api *API) userAttending(perspective, user gp.UserID, category string, mode int, index int64, count int) (events []gp.PostSmall, err error) {
	events = make([]gp.PostSmall, 0)
	q := baseQuery + attendClause
	if len(category) > 0 {
		q += categoryClause + notDeleted + notPending + byVisibleAttendance + category
	} else {
		q += notDeleted + notPending + byVisibleAttendance
	}
	switch {
	case mode == ByOffsetDescending:
		q += orderLinearAttend
	case mode == ChronologicallyAfterID:
		q += whereAfterAtt + orderChronologicalAttend
	case mode == ChronologicallyBeforeID:
		q += whereBeforeAtt + orderChronologicalAttend
	}
	s, err := api.sc.Prepare(q)
	if err != nil {
		log.Println("Error preparing statement:", err, "Statement:", q)
		return
	}
	var rows *sql.Rows
	switch {
	case len(category) > 0 && mode != ByOffsetDescending:
		rows, err = s.Query(perspective, user, category, index, user, count)
	case len(category) > 0 && mode == ByOffsetDescending:
		rows, err = s.Query(perspective, user, category, index, count)
	case mode != ByOffsetDescending:
		rows, err = s.Query(perspective, user, index, user, count)
	default:
		rows, err = s.Query(perspective, user, index, count)
	}
	if err != nil {
		log.Println("Error querying:", err, q)
		return
	}
	return api.scanPostRows(rows, false)
}

//IsAttending returns true iff this user is attending/has attended this post.
func (api *API) isAttending(userID gp.UserID, postID gp.PostID) (attending bool, err error) {
	q := "SELECT COUNT(*) FROM event_attendees WHERE user_id = ? AND post_id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(userID, postID).Scan(&attending)
	return
}

//ChangePostText sets this post's text.
func (api *API) changePostText(postID gp.PostID, text string) (err error) {
	q := "UPDATE wall_posts SET text = ? WHERE id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(text, postID)
	return
}

//AddCategory marks the post id as a member of category.
func (api *API) addCategory(id gp.PostID, category gp.CategoryID) (err error) {
	s, err := api.sc.Prepare("INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(id, category)
	return
}

//CategoryList returns all existing categories.
func (api *API) CategoryList() (categories []gp.PostCategory, err error) {
	s, err := api.sc.Prepare("SELECT id, tag, name FROM categories WHERE 1")
	if err != nil {
		return
	}
	rows, err := s.Query()
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		c := gp.PostCategory{}
		err = rows.Scan(&c.ID, &c.Tag, &c.Name)
		if err != nil {
			return
		}
		categories = append(categories, c)
	}
	return
}

//ClearCategories removes all this post's categories.
func (api *API) clearCategories(post gp.PostID) (err error) {
	s, err := api.sc.Prepare("DELETE FROM categories WHERE post_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(post)
	return
}

//CreateLike records that this user has liked this post. Acts idempotently.
func (api *API) createLike(user gp.UserID, post gp.PostID) (err error) {
	s, err := api.sc.Prepare("REPLACE INTO post_likes (post_id, user_id) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(post, user)
	return
}

//GetLikes returns all this post's likes
func (api *API) _getLikes(post gp.PostID) (likes []gp.Like, err error) {
	s, err := api.sc.Prepare("SELECT user_id, timestamp FROM post_likes WHERE post_id = ?")
	if err != nil {
		return
	}
	rows, err := s.Query(post)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		var like gp.Like
		err = rows.Scan(&like.UserID, &t)
		if err != nil {
			return
		}
		like.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return
		}
		likes = append(likes, like)
	}
	return
}

//Attend adds the user to the "attending" list for this event. It's idempotent, and should only return an error if the database is down.
//The results are undefined for a post which isn't an event.
//(ie: it will work even though it shouldn't, until I can get round to enforcing it.)
func (api *API) attend(event gp.PostID, user gp.UserID) (changed bool, err error) {
	query := "REPLACE INTO event_attendees (post_id, user_id) VALUES (?, ?)"
	s, err := api.sc.Prepare(query)
	if err != nil {
		return
	}
	res, err := s.Exec(event, user)
	if err != nil {
		return
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return
	}
	if affected > 0 {
		changed = true
	}
	return
}

//UnAttend removes a user's attendance to an event. Idempotent, returns an error if the DB is down.
func (api *API) unAttend(event gp.PostID, user gp.UserID) (err error) {
	query := "DELETE FROM event_attendees WHERE post_id = ? AND user_id = ?"
	s, err := api.sc.Prepare(query)
	if err != nil {
		return
	}
	_, err = s.Exec(event, user)
	return
}

//SubjectiveRSVPCount shows the number of events otherID has attended, from the perspective of the `perspective` user (ie, not counting those events perspective can't see...)
func (api *API) subjectiveRSVPCount(perspective gp.UserID, otherID gp.UserID) (count int, err error) {
	q := "SELECT COUNT(*) FROM event_attendees JOIN wall_posts ON event_attendees.post_id = wall_posts.id "
	q += "WHERE wall_posts.network_id IN ( SELECT network_id FROM user_network WHERE user_network.user_id = ? ) "
	q += "AND wall_posts.deleted = 0 AND wall_posts.pending = 0 "
	q += "AND event_attendees.user_id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(perspective, otherID).Scan(&count)
	return
}

//KeepPostsInFuture returns all the posts which should be kept in the future
func (api *API) keepPostsInFuture() (err error) {
	s, err := api.sc.Prepare("SELECT post_id, value FROM post_attribs WHERE attrib = 'meta-future'")
	if err != nil {
		return
	}
	rows, err := s.Query()
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var post gp.PostID
		var tstring string
		err := rows.Scan(&post, &tstring)
		d, err := time.ParseDuration(tstring)
		if err != nil {
			return err
		}
		attribs := make(map[string]string)
		attribs["event-time"] = strconv.FormatInt(time.Now().UTC().Add(d).Unix(), 10)
		err = api.setPostAttribs(post, attribs)
		if err != nil {
			return err
		}
	}
	return nil
}

func postOwner(sc *psc.StatementCache, post gp.PostID) (by gp.UserID, err error) {
	s, err := sc.Prepare("SELECT `by` FROM wall_posts WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(post).Scan(&by)
	return
}

func (api *API) MarkPostsSeen(userID gp.UserID, netID gp.NetworkID, upTo gp.PostID) (err error) {
	s, err := api.sc.Prepare("UPDATE user_network SET seen_upto = (SELECT MAX(id) FROM wall_posts WHERE id < ?) WHERE user_id = ? AND network_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(upTo, userID, netID)
	return
}

//NetworkChannel gives the event channel for this network
func NetworkChannel(netID gp.NetworkID) string {
	return fmt.Sprintf("networks.%d", netID)
}
