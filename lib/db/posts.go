package db

import (
	"database/sql"
	"log"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

/********************************************************************
		Post
********************************************************************/
const (
	//WNETWORK is posts in this network.
	WNETWORK = iota
	//WUSER is posts by this user.
	WUSER
	//WGROUPS is posts in all groups this user belongs to.
	WGROUPS
)

//EBADORDER means you tried to order a post query in an unexpected way.
var EBADORDER = gp.APIerror{Reason: "Invalid order clause!"}

//EBADWHERE means you tried to filter posts in an unexpected way.
var EBADWHERE = gp.APIerror{Reason: "Bad WhereClause!"}

//WhereClause specifies how we're looking for posts.
//Mode may be one of WNETWORk, WUSER, WGROUPS.
//Network specifies which network we're looking at if using WNETWORK
//User specifies the user we're looking at if using WUSER
//Perspective is the user whose perspective we're looking from when using WUSER or WGROUPS
//Category optionally restricts posts just to this one category.
type WhereClause struct {
	Mode        int
	Network     gp.NetworkID
	User        gp.UserID
	Perspective gp.UserID
	Category    string
}

//WhereRows basically acts as a shitty ORM and returns a sql.Rows which can then be blindly iterated over by NewGetPosts.
//This is an abomination which I'm too scared and have no time to touch, but it works. For now.
//Let's just hope that you, dear reader, don't ever have to extend it.
func (db *DB) WhereRows(w WhereClause, orderMode int, index int64, count int) (rows *sql.Rows, err error) {
	//Oh shit. I accidentally an ORM?
	baseQuery := "SELECT wall_posts.id, `by`, time, text, network_id FROM wall_posts "
	var orderClause string
	var categoryClause = "JOIN post_categories ON wall_posts.id = post_categories.post_id " +
		"JOIN categories ON post_categories.category_id = categories.id "
	var stmt *sql.Stmt
	switch {
	case w.Mode == WNETWORK:
		whereClause := "WHERE deleted = 0 AND network_id = ? "
		switch {
		case orderMode == gp.OSTART:
			orderClause = "ORDER BY time DESC, id DESC LIMIT ?, ?"
		case orderMode == gp.OBEFORE:
			whereClause += "AND wall_posts.id < ? "
			orderClause = "ORDER BY time DESC, id DESC LIMIT 0, ?"
		case orderMode == gp.OAFTER:
			whereClause += "AND wall_posts.id > ? "
			orderClause = "ORDER BY time DESC, id DESC LIMIT 0, ?"
		default:
			err = &EBADORDER
			return
		}
		if len(w.Category) > 0 {
			whereClause = categoryClause + whereClause + "AND categories.tag = ? "
		}
		stmt, err = db.prepare(baseQuery + whereClause + orderClause)
		if err != nil {
			return
		}
		if len(w.Category) > 0 {
			rows, err = stmt.Query(w.Network, w.Category, index, count)
		} else {
			rows, err = stmt.Query(w.Network, index, count)
		}
	case w.Mode == WUSER:
		whereClause := "WHERE deleted = 0 AND `by` = ? " +
			"AND network_id IN ( " +
			"SELECT network_id FROM user_network WHERE user_id = ? " +
			") "
		switch {
		case orderMode == gp.OSTART:
			orderClause = "ORDER BY time DESC, id DESC LIMIT ?, ?"
		case orderMode == gp.OBEFORE:
			whereClause += "AND wall_posts.id < ? "
			orderClause = "ORDER BY time DESC, id DESC LIMIT 0, ?"
		case orderMode == gp.OAFTER:
			whereClause += "AND wall_posts.id > ? "
			orderClause = "ORDER BY time DESC, id DESC LIMIT 0, ?"
		default:
			err = &EBADORDER
			return
		}
		if len(w.Category) > 0 {
			whereClause = categoryClause + whereClause + "AND categories.tag = ? "
		}
		log.Println("User networks query:", baseQuery+whereClause+orderClause)
		stmt, err = db.prepare(baseQuery + whereClause + orderClause)
		if err != nil {
			return
		}
		if len(w.Category) > 0 {
			rows, err = stmt.Query(w.User, w.Perspective, w.Category, index, count)
			log.Println("User networks query arguments:", w.User, w.Perspective, w.Category, index, count)
		} else {
			rows, err = stmt.Query(w.User, w.Perspective, index, count)
			log.Println("User networks query arguments:", w.User, w.Perspective, index, count)
		}
	case w.Mode == WGROUPS:
		whereClause := "WHERE deleted = 0 AND network_id IN ( " +
			"SELECT network_id " +
			"FROM user_network " +
			"JOIN network ON user_network.network_id = network.id " +
			"WHERE user_id = ? " +
			"AND network.user_group = 1 " +
			" ) "
		switch {
		case orderMode == gp.OSTART:
			orderClause = " ORDER BY time DESC, id DESC LIMIT ?, ?"
		case orderMode == gp.OBEFORE:
			whereClause += "AND wall_posts.id < ? "
			orderClause = "ORDER BY time DESC, id DESC LIMIT 0, ?"
		case orderMode == gp.OAFTER:
			whereClause += "AND wall_posts.id > ? "
			orderClause = "ORDER BY time DESC, id DESC LIMIT 0, ?"
		default:
			err = &EBADORDER
			return
		}
		if len(w.Category) > 0 {
			whereClause = categoryClause + whereClause + "AND categories.tag = ? "
		}
		stmt, err = db.prepare(baseQuery + whereClause + orderClause)
		if err != nil {
			return
		}
		if len(w.Category) > 0 {
			rows, err = stmt.Query(w.User, w.Category, index, count)
		} else {
			rows, err = stmt.Query(w.User, index, count)
		}
	default:
		err = &EBADWHERE
		return
	}
	return rows, err
}

//NewGetPosts returns posts matching the WhereClause.
func (db *DB) NewGetPosts(where WhereClause, orderMode int, index int64, count int) (posts []gp.PostSmall, err error) {
	rows, err := db.WhereRows(where, orderMode, index, count)
	if err != nil {
		return
	}
	defer rows.Close()
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
			if where.Mode == WGROUPS || where.Mode == WUSER {
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
func (db *DB) GetUserPosts(userID gp.UserID, perspective gp.UserID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	where := WhereClause{Mode: WUSER, User: userID, Perspective: perspective, Category: category}
	posts, err = db.NewGetPosts(where, mode, index, count)
	return
}

//AddPost creates a post, returning the created ID. It only handles the core of the post; other attributes, images and so on must be created separately.
func (db *DB) AddPost(userID gp.UserID, text string, network gp.NetworkID) (postID gp.PostID, err error) {
	s, err := db.prepare("INSERT INTO wall_posts(`by`, `text`, network_id) VALUES (?,?,?)")
	if err != nil {
		return
	}
	res, err := s.Exec(userID, text, network)
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
	q := "SELECT wall_posts.id, `by`, time, text " +
		"FROM wall_posts " +
		"JOIN post_attribs ON wall_posts.id = post_attribs.post_id " +
		"WHERE deleted = 0 AND network_id = ? AND attrib = 'event-time' AND value > ? " +
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
	for rows.Next() {
		var post gp.PostSmall
		var t string
		var by gp.UserID
		err = rows.Scan(&post.ID, &by, &t, &post.Text)
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
			posts = append(posts, post)
		} else {
			log.Println("Bad post: ", post)
		}
	}
	return
}

//GetPosts finds posts in the network netId.
func (db *DB) GetPosts(netID gp.NetworkID, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	where := WhereClause{Mode: WNETWORK, Network: netID, Category: category}
	posts, err = db.NewGetPosts(where, mode, index, count)
	return
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
	where := WhereClause{Mode: WGROUPS, User: user, Category: category}
	posts, err = db.NewGetPosts(where, mode, index, count)
	return
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
	q := "SELECT id, name, firstname, avatar FROM users JOIN event_attendees ON user_id = id WHERE post_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(post)
	if err != nil {
		return
	}
	var first, avatar sql.NullString
	for rows.Next() {
		var user gp.User
		err = rows.Scan(&user.ID, &user.Name, &first, &avatar)
		if first.Valid {
			user.Name = first.String
		}
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
	q += "AND deleted = 0 "
	q += "AND network_id IN (SELECT network_id FROM user_network WHERE user_id = ?)"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(user, perspective).Scan(&count)
	return
}
