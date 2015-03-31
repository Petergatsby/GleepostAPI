package db

import (
	"database/sql"
	"log"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

const (
	//WNETWORK is posts in this network.
	WNETWORK = iota
	//WUSER is posts by this user.
	WUSER
	//WGROUPS is posts in all groups this user belongs to.
	WGROUPS
	//WATTENDS is events this user has attended
	WATTENDS
)

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
	orderLinear        = "ORDER BY time DESC, id DESC LIMIT ?, ?"
	orderChronological = "ORDER BY time DESC, id DESC LIMIT 0, ?"

	orderLinearAttend        = "ORDER BY event_attendees.time DESC, id DESC LIMIT ?, ?"
	orderChronologicalAttend = "ORDER BY event_attendees.time DESC, id DESC LIMIT 0, ?"
)

//EBADORDER means you tried to order a post query in an unexpected way.
var EBADORDER = gp.APIerror{Reason: "Invalid order clause!"}

func (db *DB) scanPostRows(rows *sql.Rows, expandNetworks bool) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	for rows.Next() {
		log.Println("Post!")
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
		post.By, err = db.GetUser(by)
		if err == nil {
			post.CommentCount = db.GetCommentCount(post.ID)
			post.Images, err = db.GetPostImages(post.ID)
			if err != nil {
				return
			}
			post.Videos, err = db.GetPostVideos(post.ID)
			if err != nil {
				return
			}
			post.LikeCount, err = db.LikeCount(post.ID)
			if err != nil {
				return
			}
			if expandNetworks {
				net, err := db.GetNetwork(post.Network)
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
func (db *DB) GetUserPosts(userID, perspective gp.UserID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	var q string
	if len(category) > 0 {
		q = baseQuery + categoryClause + notDeleted + notPending + byPoster + category
	} else {
		q = baseQuery + notDeleted + notPending + byPoster
	}
	switch {
	case mode == gp.OSTART:
		q += orderLinear
	case mode == gp.OAFTER:
		q += whereAfter + orderChronological
	case mode == gp.OBEFORE:
		q += whereBefore + orderChronological
	}
	s, err := db.prepare(q)
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
	return db.scanPostRows(rows, true)
}

//AddPost creates a post, returning the created ID. It only handles the core of the post; other attributes, images and so on must be created separately.
func (db *DB) AddPost(userID gp.UserID, text string, network gp.NetworkID, pending bool) (postID gp.PostID, err error) {
	s, err := db.prepare("INSERT INTO wall_posts(`by`, `text`, network_id, pending) VALUES (?,?,?,?)")
	if err != nil {
		return
	}
	res, err := s.Exec(userID, text, network, pending)
	if err != nil {
		return 0, err
	}
	_postID, err := res.LastInsertId()
	postID = gp.PostID(_postID)
	if err != nil {
		return 0, err
	}
	return postID, nil
}

//GetLive returns a list of events whose event time is after "after", ordered by time.
func (db *DB) GetLive(netID gp.NetworkID, after time.Time, count int) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	q := "SELECT wall_posts.id, `by`, time, text, network_id " +
		"FROM wall_posts " +
		"JOIN post_attribs ON wall_posts.id = post_attribs.post_id " +
		"WHERE deleted = 0 AND pending = 0 AND network_id = ? AND attrib = 'event-time' AND value > ? " +
		"ORDER BY value ASC LIMIT 0, ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(netID, after.Unix(), count)
	if err != nil {
		return
	}
	defer rows.Close()
	//The second argument is meaningless and should be removed.
	return db.scanPostRows(rows, false)
}

//GetPosts finds posts in the network netId.
func (db *DB) GetPosts(netID gp.NetworkID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	var q string
	if len(category) > 0 {
		q = baseQuery + categoryClause + notDeleted + notPending + byNetwork + whereCategory
	} else {
		q = baseQuery + notDeleted + notPending + byNetwork
	}
	switch {
	case mode == gp.OSTART:
		q += orderLinear
	case mode == gp.OAFTER:
		q += whereAfter + orderChronological
	case mode == gp.OBEFORE:
		q += whereBefore + orderChronological
	}
	s, err := db.prepare(q)
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
	return db.scanPostRows(rows, false)
}

//GetPostImages returns all the images associated with postID.
func (db *DB) GetPostImages(postID gp.PostID) (images []string, err error) {
	s, err := db.prepare("SELECT url FROM post_images WHERE post_id = ?")
	if err != nil {
		return
	}
	rows, err := s.Query(postID)
	defer rows.Close()
	log.Println("DB hit: getImages postId(image)")
	if err != nil {
		return
	}
	for rows.Next() {
		var image string
		err = rows.Scan(&image)
		if err != nil {
			return
		}
		images = append(images, image)
	}
	return
}

//AddPostImage adds an image (url) to postID.
func (db *DB) AddPostImage(postID gp.PostID, url string) (err error) {
	s, err := db.prepare("INSERT INTO post_images (post_id, url) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(postID, url)
	return
}

//ClearPostImages deletes all images from this post.
func (db *DB) ClearPostImages(postID gp.PostID) (err error) {
	s, err := db.prepare("DELETE FROM post_images WHERE post_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(postID)
	return
}

//ClearPostVideos deletes all videos from this post.
func (db *DB) ClearPostVideos(postID gp.PostID) (err error) {
	s, err := db.prepare("DELETE FROM post_videos WHERE post_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(postID)
	return
}

//AddPostVideo adds this video URL to a post.
func (db *DB) AddPostVideo(postID gp.PostID, videoID gp.VideoID) (err error) {
	s, err := db.prepare("INSERT INTO post_videos (post_id, video_id) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(postID, videoID)
	return
}

//GetPostVideos returns all the videos associated with postID
func (db *DB) GetPostVideos(postID gp.PostID) (videos []gp.Video, err error) {
	s, err := db.prepare("SELECT url, mp4_url, webm_url FROM uploads JOIN post_videos ON upload_id = video_id WHERE post_id = ? AND status = 'ready'")
	if err != nil {
		return
	}
	rows, err := s.Query(postID)
	defer rows.Close()
	log.Println("DB hit: getVideos postId(image)")
	if err != nil {
		return
	}
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

//CreateComment adds a comment on this post.
func (db *DB) CreateComment(postID gp.PostID, userID gp.UserID, text string) (commID gp.CommentID, err error) {
	s, err := db.prepare("INSERT INTO post_comments (post_id, `by`, text) VALUES (?, ?, ?)")
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

//GetComments returns up to count comments for this post.
func (db *DB) GetComments(postID gp.PostID, start int64, count int) (comments []gp.Comment, err error) {
	comments = make([]gp.Comment, 0)
	q := "SELECT id, `by`, text, `timestamp` " +
		"FROM post_comments " +
		"WHERE post_id = ? " +
		"ORDER BY `timestamp` DESC LIMIT ?, ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(postID, start, count)
	log.Println("DB hit: getComments postid, start(comment.id, comment.by, comment.text, comment.time)")
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
		comment.By, err = db.GetUser(by)
		if err != nil {
			log.Printf("error getting user %d %v", by, err)
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

//GetCommentCount returns the total number of comments for this post.
func (db *DB) GetCommentCount(id gp.PostID) (count int) {
	s, err := db.prepare("SELECT COUNT(*) FROM post_comments WHERE post_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

//UserGetPost returns the post postId or an error if it doesn't exist.
//TODO: This could return without an embedded user or images array
func (db *DB) UserGetPost(userID gp.UserID, postID gp.PostID) (post gp.Post, err error) {
	s, err := db.prepare("SELECT `network_id`, `by`, `time`, text FROM wall_posts WHERE deleted = 0 AND id = ? AND (pending = 0 OR `by` = ?)")
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
	post.By, err = db.GetUser(by)
	if err != nil {
		return
	}
	post.Time, err = time.Parse(mysqlTime, t)
	if err != nil {
		return
	}
	post.Images, err = db.GetPostImages(postID)
	if err != nil {
		return
	}
	post.Videos, err = db.GetPostVideos(postID)
	return
}

//GetPost returns the post postId or an error if it doesn't exist.
//TODO: This could return without an embedded user or images array
func (db *DB) GetPost(postID gp.PostID) (post gp.Post, err error) {
	s, err := db.prepare("SELECT `network_id`, `by`, `time`, text FROM wall_posts WHERE deleted = 0 AND id = ?")
	if err != nil {
		return
	}
	post.ID = postID
	var by gp.UserID
	var t string
	err = s.QueryRow(postID).Scan(&post.Network, &by, &t, &post.Text)
	if err != nil {
		return
	}
	post.By, err = db.GetUser(by)
	if err != nil {
		return
	}
	post.Time, err = time.Parse(mysqlTime, t)
	if err != nil {
		return
	}
	post.Images, err = db.GetPostImages(postID)
	if err != nil {
		return
	}
	post.Videos, err = db.GetPostVideos(postID)
	return
}

//SetPostAttribs associates all the attribute:value pairs in attrib with post.
//At the moment, it doesn't check if these attributes are at all reasonable;
//the onus is on the viewer of the attributes to look for just the ones which make sense,
//and on the caller of this function to ensure that the values conform to a particular format.
func (db *DB) SetPostAttribs(post gp.PostID, attribs map[string]string) (err error) {
	s, err := db.prepare("REPLACE INTO post_attribs (post_id, attrib, value) VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	for attrib, value := range attribs {
		//How could I be so foolish to store time strings rather than unix timestamps...
		if attrib == "event-time" {
			t, e := time.Parse(value, time.RFC3339)
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
	return
}

//GetPostAttribs returns a map of all attributes associated with post.
func (db *DB) GetPostAttribs(post gp.PostID) (attribs map[string]interface{}, err error) {
	s, err := db.prepare("SELECT attrib, value FROM post_attribs WHERE post_id=?")
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
			log.Println("event-time")
			var unix int64
			unix, err = strconv.ParseInt(val, 10, 64)
			if err == nil {
				log.Println("no error")
				attribs[attrib] = time.Unix(unix, 0)
			}
		default:
			attribs[attrib] = val
		}
	}
	return
}

//GetEventPopularity returns the popularity score (0 - 99) and the actual attendees count
func (db *DB) GetEventPopularity(post gp.PostID) (popularity int, attendees int, err error) {
	query := "SELECT COUNT(*) FROM event_attendees WHERE post_id = ?"
	s, err := db.prepare(query)
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
func (db *DB) UserGetGroupsPosts(user gp.UserID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	posts = make([]gp.PostSmall, 0)
	var q string
	if len(category) > 0 {
		q = baseQuery + categoryClause + notDeleted + notPending + byUserGroups + whereCategory
	} else {
		q = baseQuery + notDeleted + notPending + byUserGroups
	}
	switch {
	case mode == gp.OSTART:
		q += orderLinear
	case mode == gp.OAFTER:
		q += whereAfter + orderChronological
	case mode == gp.OBEFORE:
		q += whereBefore + orderChronological
	}
	s, err := db.prepare(q)
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
	return db.scanPostRows(rows, true)
}

//DeletePost marks a post as deleted in the database.
func (db *DB) DeletePost(post gp.PostID) (err error) {
	q := "UPDATE wall_posts SET deleted = 1 WHERE id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(post)
	return
}

//EventAttendees returns all users who are attending this event.
func (db *DB) EventAttendees(post gp.PostID) (attendees []gp.User, err error) {
	q := "SELECT id, firstname, avatar, official FROM users JOIN event_attendees ON user_id = id WHERE post_id = ?"
	s, err := db.prepare(q)
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
func (db *DB) UserPostCount(perspective, user gp.UserID) (count int, err error) {
	q := "SELECT COUNT(*) FROM wall_posts "
	q += "WHERE `by` = ? "
	q += "AND deleted = 0 AND pending = 0 "
	q += "AND network_id IN (SELECT network_id FROM user_network WHERE user_id = ?)"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(user, perspective).Scan(&count)
	return
}

//UserAttending returns all the events this user is attending.
func (db *DB) UserAttending(perspective, user gp.UserID, category string, mode int, index int64, count int) (events []gp.PostSmall, err error) {
	events = make([]gp.PostSmall, 0)
	q := baseQuery + attendClause
	if len(category) > 0 {
		q += categoryClause + notDeleted + notPending + byVisibleAttendance + category
	} else {
		q += notDeleted + notPending + byVisibleAttendance
	}
	switch {
	case mode == gp.OSTART:
		q += orderLinearAttend
	case mode == gp.OAFTER:
		q += whereAfterAtt + orderChronologicalAttend
	case mode == gp.OBEFORE:
		q += whereBeforeAtt + orderChronologicalAttend
	}
	s, err := db.prepare(q)
	if err != nil {
		log.Println("Error preparing statement:", err, "Statement:", q)
		return
	}
	var rows *sql.Rows
	switch {
	case len(category) > 0 && mode != gp.OSTART:
		rows, err = s.Query(perspective, user, category, index, user, count)
	case len(category) > 0 && mode == gp.OSTART:
		rows, err = s.Query(perspective, user, category, index, count)
	case mode != gp.OSTART:
		rows, err = s.Query(perspective, user, index, user, count)
	default:
		rows, err = s.Query(perspective, user, index, count)
	}
	if err != nil {
		log.Println("Error querying:", err, q)
		return
	}
	return db.scanPostRows(rows, false)
}

//IsAttending returns true iff this user is attending/has attended this post.
func (db *DB) IsAttending(userID gp.UserID, postID gp.PostID) (attending bool, err error) {
	q := "SELECT COUNT(*) FROM event_attendees WHERE user_id = ? AND post_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(userID, postID).Scan(&attending)
	return
}

//ChangePostText sets this post's text.
func (db *DB) ChangePostText(postID gp.PostID, text string) (err error) {
	q := "UPDATE wall_posts SET text = ? WHERE id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(text, postID)
	return
}

//AddCategory marks the post id as a member of category.
func (db *DB) AddCategory(id gp.PostID, category gp.CategoryID) (err error) {
	s, err := db.prepare("INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(id, category)
	return
}

//CategoryList returns all existing categories.
func (db *DB) CategoryList() (categories []gp.PostCategory, err error) {
	s, err := db.prepare("SELECT id, tag, name FROM categories WHERE 1")
	if err != nil {
		return
	}
	rows, err := s.Query()
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

//TagPost accepts a post id and any number of string tags. Any of the tags that exist will be added to the post.
func (db *DB) TagPost(post gp.PostID, tags ...string) (err error) {
	s, err := db.prepare("INSERT INTO post_categories( post_id, category_id ) SELECT ? , categories.id FROM categories WHERE categories.tag = ?")
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

//PostCategories returns all the categories which post belongs to.
func (db *DB) PostCategories(post gp.PostID) (categories []gp.PostCategory, err error) {
	s, err := db.prepare("SELECT categories.id, categories.tag, categories.name FROM post_categories JOIN categories ON post_categories.category_id = categories.id WHERE post_categories.post_id = ?")
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

//ClearCategories removes all this post's categories.
func (db *DB) ClearCategories(post gp.PostID) (err error) {
	s, err := db.prepare("DELETE FROM categories WHERE post_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(post)
	return
}

//CreateLike records that this user has liked this post. Acts idempotently.
func (db *DB) CreateLike(user gp.UserID, post gp.PostID) (err error) {
	s, err := db.prepare("REPLACE INTO post_likes (post_id, user_id) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(post, user)
	return
}

//RemoveLike un-likes a post.
func (db *DB) RemoveLike(user gp.UserID, post gp.PostID) (err error) {
	s, err := db.prepare("DELETE FROM post_likes WHERE post_id = ? AND user_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(post, user)
	return
}

//GetLikes returns all this post's likes
func (db *DB) GetLikes(post gp.PostID) (likes []gp.Like, err error) {
	s, err := db.prepare("SELECT user_id, timestamp FROM post_likes WHERE post_id = ?")
	if err != nil {
		return
	}
	rows, err := s.Query(post)
	defer rows.Close()
	if err != nil {
		return
	}
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

//HasLiked retuns true if this user has already liked this post.
func (db *DB) HasLiked(user gp.UserID, post gp.PostID) (liked bool, err error) {
	s, err := db.prepare("SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND user_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(post, user).Scan(&liked)
	return
}

//LikeCount returns the number of likes this post has.
func (db *DB) LikeCount(post gp.PostID) (count int, err error) {
	s, err := db.prepare("SELECT COUNT(*) FROM post_likes WHERE post_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(post).Scan(&count)
	return
}

//Attend adds the user to the "attending" list for this event. It's idempotent, and should only return an error if the database is down.
//The results are undefined for a post which isn't an event.
//(ie: it will work even though it shouldn't, until I can get round to enforcing it.)
func (db *DB) Attend(event gp.PostID, user gp.UserID) (err error) {
	query := "REPLACE INTO event_attendees (post_id, user_id) VALUES (?, ?)"
	s, err := db.prepare(query)
	if err != nil {
		return
	}
	_, err = s.Exec(event, user)
	return
}

//UnAttend removes a user's attendance to an event. Idempotent, returns an error if the DB is down.
func (db *DB) UnAttend(event gp.PostID, user gp.UserID) (err error) {
	query := "DELETE FROM event_attendees WHERE post_id = ? AND user_id = ?"
	s, err := db.prepare(query)
	if err != nil {
		return
	}
	_, err = s.Exec(event, user)
	return
}

//UserAttends returns all the event IDs that a user is attending.
func (db *DB) UserAttends(user gp.UserID) (events []gp.PostID, err error) {
	events = make([]gp.PostID, 0)
	query := "SELECT post_id FROM event_attendees WHERE user_id = ?"
	s, err := db.prepare(query)
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

//SubjectiveRSVPCount shows the number of events otherID has attended, from the perspective of the `perspective` user (ie, not counting those events perspective can't see...)
func (db *DB) SubjectiveRSVPCount(perspective gp.UserID, otherID gp.UserID) (count int, err error) {
	q := "SELECT COUNT(*) FROM event_attendees JOIN wall_posts ON event_attendees.post_id = wall_posts.id "
	q += "WHERE wall_posts.network_id IN ( SELECT network_id FROM user_network WHERE user_network.user_id = ? ) "
	q += "AND wall_posts.deleted = 0 AND wall_posts.pending = 0 "
	q += "AND event_attendees.user_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(perspective, otherID).Scan(&count)
	return
}

//KeepPostsInFuture returns all the posts which should be kept in the future
func (db *DB) KeepPostsInFuture() (err error) {
	s, err := db.prepare("SELECT post_id, value FROM post_attribs WHERE attrib = 'meta-future'")
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
		err = db.SetPostAttribs(post, attribs)
		if err != nil {
			return err
		}
	}
	return nil
}
