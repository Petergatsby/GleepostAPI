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
	WNETWORK = iota
	WUSER
	WGROUPS
)

var EBADORDER = gp.APIerror{Reason: "Invalid order clause!"}
var EBADWHERE = gp.APIerror{Reason: "Bad WhereClause!"}

type WhereClause struct {
	Mode        int
	Network     gp.NetworkId
	User        gp.UserId
	Perspective gp.UserId
	Category    string
}

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
			orderClause = "ORDER BY time DESC LIMIT ?, ?"
		case orderMode == gp.OBEFORE:
			whereClause += "AND wall_posts.id < ? "
			orderClause = "ORDER BY time DESC LIMIT 0, ?"
		case orderMode == gp.OAFTER:
			whereClause += "AND wall_posts.id > ? "
			orderClause = "ORDER BY time DESC LIMIT 0, ?"
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
			orderClause = "ORDER BY time DESC LIMIT ?, ?"
		case orderMode == gp.OBEFORE:
			whereClause += "AND wall_posts.id < ? "
			orderClause = "ORDER BY time DESC LIMIT 0, ?"
		case orderMode == gp.OAFTER:
			whereClause += "AND wall_posts.id > ? "
			orderClause = "ORDER BY time DESC LIMIT 0, ?"
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
			orderClause = " ORDER BY time DESC LIMIT ?, ?"
		case orderMode == gp.OBEFORE:
			whereClause += "AND wall_posts.id < ? "
			orderClause = "ORDER BY time DESC LIMIT 0, ?"
		case orderMode == gp.OAFTER:
			whereClause += "AND wall_posts.id > ? "
			orderClause = "ORDER BY time DESC LIMIT 0, ?"
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
		var by gp.UserId
		err = rows.Scan(&post.Id, &by, &t, &post.Text, &post.Network)
		if err != nil {
			return posts, err
		}
		post.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return posts, err
		}
		post.By, err = db.GetUser(by)
		if err == nil {
			post.CommentCount = db.GetCommentCount(post.Id)
			post.Images, err = db.GetPostImages(post.Id)
			if err != nil {
				return
			}
			post.LikeCount, err = db.LikeCount(post.Id)
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
func (db *DB) GetUserPosts(userID gp.UserId, perspective gp.UserId, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	where := WhereClause{Mode: WUSER, User: userID, Perspective: perspective, Category: category}
	posts, err = db.NewGetPosts(where, mode, index, count)
	return
}

func (db *DB) AddPost(userID gp.UserId, text string, network gp.NetworkId) (postID gp.PostId, err error) {
	s := db.stmt["postInsert"]
	res, err := s.Exec(userID, text, network)
	if err != nil {
		return 0, err
	}
	_postId, err := res.LastInsertId()
	postID = gp.PostId(_postId)
	if err != nil {
		return 0, err
	}
	return postID, nil
}

//GetLive returns a list of events whose event time is after "after", ordered by time.
func (db *DB) GetLive(netID gp.NetworkId, after time.Time, count int) (posts []gp.PostSmall, err error) {
	s := db.stmt["liveSelect"]
	rows, err := s.Query(netID, after.Unix(), count)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var post gp.PostSmall
		var t string
		var by gp.UserId
		err = rows.Scan(&post.Id, &by, &t, &post.Text)
		if err != nil {
			return posts, err
		}
		post.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return posts, err
		}
		post.By, err = db.GetUser(by)
		if err == nil {
			post.CommentCount = db.GetCommentCount(post.Id)
			post.Images, err = db.GetPostImages(post.Id)
			if err != nil {
				return
			}
			post.LikeCount, err = db.LikeCount(post.Id)
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
func (db *DB) GetPosts(netID gp.NetworkId, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	where := WhereClause{Mode: WNETWORK, Network: netID, Category: category}
	posts, err = db.NewGetPosts(where, mode, index, count)
	return
}

func (db *DB) GetPostImages(postID gp.PostId) (images []string, err error) {
	s := db.stmt["imageSelect"]
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

func (db *DB) AddPostImage(postID gp.PostId, url string) (err error) {
	_, err = db.stmt["imageInsert"].Exec(postID, url)
	return
}

func (db *DB) CreateComment(postID gp.PostId, userID gp.UserId, text string) (commID gp.CommentId, err error) {
	s := db.stmt["commentInsert"]
	if res, err := s.Exec(postID, userID, text); err == nil {
		cId, err := res.LastInsertId()
		commID = gp.CommentId(cId)
		return commID, err
	} else {
		return 0, err
	}
}

func (db *DB) GetComments(postID gp.PostId, start int64, count int) (comments []gp.Comment, err error) {
	s := db.stmt["commentSelect"]
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
		var by gp.UserId
		err := rows.Scan(&comment.Id, &by, &comment.Text, &timeString)
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

func (db *DB) GetCommentCount(id gp.PostId) (count int) {
	s := db.stmt["commentCountSelect"]
	err := s.QueryRow(id).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

//GetPost returns the post postId or an error if it doesn't exist.
//TODO: This could return without an embedded user or images array
func (db *DB) GetPost(postID gp.PostId) (post gp.Post, err error) {
	s := db.stmt["postSelect"]
	post.Id = postID
	var by gp.UserId
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
	return
}

//SetPostAttribs associates all the attribute:value pairs in attrib with post.
//At the moment, it doesn't check if these attributes are at all reasonable;
//the onus is on the viewer of the attributes to look for just the ones which make sense,
//and on the caller of this function to ensure that the values conform to a particular format.
func (db *DB) SetPostAttribs(post gp.PostId, attribs map[string]string) (err error) {
	s := db.stmt["setPostAttribs"]
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
func (db *DB) GetPostAttribs(post gp.PostId) (attribs map[string]interface{}, err error) {
	s := db.stmt["getPostAttribs"]
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
func (db *DB) GetEventPopularity(post gp.PostId) (popularity int, attendees int, err error) {
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

//TODO: Verify shit doesn't break when a user has no user-groups
func (db *DB) UserGetGroupsPosts(user gp.UserId, mode int, index int64, count int, category string) (posts []gp.PostSmall, err error) {
	where := WhereClause{Mode: WGROUPS, User: user, Category: category}
	posts, err = db.NewGetPosts(where, mode, index, count)
	return
}

//DeletePost marks a post as deleted in the database.
func (db *DB) DeletePost(post gp.PostId) (err error) {
	q := "UPDATE wall_posts SET deleted = 1 WHERE id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(post)
	return
}

func (db *DB) EventAttendees(post gp.PostId) (attendees []gp.User, err error) {
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
		err = rows.Scan(&user.Id, &user.Name, &first, &avatar)
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
