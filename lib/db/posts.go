package db

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"strconv"
	"database/sql"
	"time"
	"log"
)

/********************************************************************
		Post
********************************************************************/

//GetUserPosts returns the most recent count posts by userId after the post with id after.
func (db *DB) GetUserPosts(userId gp.UserId, index int64, count int, sel string) (posts []gp.PostSmall, err error) {
	var q string
	switch {
	case sel == "start":
		q = "SELECT wall_posts.id, `by`, time, text " +
			"FROM wall_posts " +
			"WHERE `by` = ? " +
			"ORDER BY time DESC LIMIT ?, ?"
	case sel == "before":
		q = "SELECT wall_posts.id, `by`, time, text " +
			"FROM wall_posts " +
			"WHERE `by` = ? AND id < ? " +
			"ORDER BY time DESC LIMIT 0, ?"
	case sel == "after":
		q = "SELECT wall_posts.id, `by`, time, text " +
			"FROM wall_posts " +
			"WHERE `by` = ? AND id > ? " +
			"ORDER BY time DESC LIMIT 0, ?"
	default:
		return posts, gp.APIerror{"Invalid selector"}
	}

	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(userId, index, count)
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

func (db *DB) AddPost(userId gp.UserId, text string, network gp.NetworkId) (postId gp.PostId, err error) {
	s := db.stmt["postInsert"]
	res, err := s.Exec(userId, text, network)
	if err != nil {
		return 0, err
	}
	_postId, err := res.LastInsertId()
	postId = gp.PostId(_postId)
	if err != nil {
		return 0, err
	}
	return postId, nil
}

//GetLive returns a list of events whose event time is after "after", ordered by time.
func (db *DB) GetLive(netId gp.NetworkId, after time.Time, count int) (posts []gp.PostSmall, err error) {
	s := db.stmt["liveSelect"]
	rows, err := s.Query(netId, after.Unix(), count)
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
func (db *DB) GetPosts(netId gp.NetworkId, index int64, count int, sel string) (posts []gp.PostSmall, err error) {
	var s *sql.Stmt
	switch {
	case sel == "start":
		s = db.stmt["wallSelect"]
	case sel == "before":
		s = db.stmt["wallSelectBefore"]
	case sel == "after":
		s = db.stmt["wallSelectAfter"]
	default:
		return posts, gp.APIerror{"Invalid selector"}
	}
	rows, err := s.Query(netId, index, count)
	log.Println(rows, err, netId, index, count)
	log.Println("DB hit: getPosts netId(post.id, post.by, post.time, post.texts)")
	if err != nil {
		log.Println("Error yo! ", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		log.Println("Post!")
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
func (db *DB) GetPostsByCategory(netId gp.NetworkId, index int64, count int, sel string, categoryTag string) (posts []gp.PostSmall, err error) {
	var s *sql.Stmt
	switch {
	case sel == "start":
		s = db.stmt["wallSelectCategory"]
	case sel == "before":
		s = db.stmt["wallSelectCategoryBefore"]
	case sel == "after":
		s = db.stmt["wallSelectCategoryAfter"]
	default:
		return posts, gp.APIerror{"Invalid selector"}
	}
	rows, err := s.Query(netId, categoryTag, index, count)
	defer rows.Close()
	log.Printf("DB hit: getPostsByCategory network: %s category: %s index: %d count: %d", netId, categoryTag, index, count)
	if err != nil {
		log.Println(err)
		return
	}
	for rows.Next() {
		log.Println("Got a post")
		var post gp.PostSmall
		var t string
		var by gp.UserId
		err = rows.Scan(&post.Id, &by, &t, &post.Text)
		log.Println("Scanned a post")
		if err != nil {
			log.Println("Error scanning post: ", err)
			return posts, err
		}
		post.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			log.Println("Error parsing time: ", err)
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
			log.Println("Added a post")
		} else {
			log.Println("Bad post: ", post)
		}
	}
	return
}

func (db *DB) GetPostImages(postId gp.PostId) (images []string, err error) {
	s := db.stmt["imageSelect"]
	rows, err := s.Query(postId)
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

func (db *DB) AddPostImage(postId gp.PostId, url string) (err error) {
	_, err = db.stmt["imageInsert"].Exec(postId, url)
	return
}

func (db *DB) CreateComment(postId gp.PostId, userId gp.UserId, text string) (commId gp.CommentId, err error) {
	s := db.stmt["commentInsert"]
	if res, err := s.Exec(postId, userId, text); err == nil {
		cId, err := res.LastInsertId()
		commId = gp.CommentId(cId)
		return commId, err
	} else {
		return 0, err
	}
}

func (db *DB) GetComments(postId gp.PostId, start int64, count int) (comments []gp.Comment, err error) {
	s := db.stmt["commentSelect"]
	rows, err := s.Query(postId, start, count)
	log.Println("DB hit: getComments postid, start(comment.id, comment.by, comment.text, comment.time)")
	if err != nil {
		return comments, err
	}
	defer rows.Close()
	for rows.Next() {
		var comment gp.Comment
		comment.Post = postId
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
func (db *DB) GetPost(postId gp.PostId) (post gp.Post, err error) {
	s := db.stmt["postSelect"]
	post.Id = postId
	var by gp.UserId
	var t string
	err = s.QueryRow(postId).Scan(&post.Network, &by, &t, &post.Text)
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
	post.Images, err = db.GetPostImages(postId)
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

func (db *DB) GetEventPopularity(post gp.PostId) (popularity int, err error) {
	query := "SELECT COUNT(*) FROM event_attendees WHERE post_id = ?"
	s, err := db.prepare(query)
	if err != nil {
		return
	}
	err = s.QueryRow(post).Scan(&popularity)
	if err != nil {
		return
	}
	switch {
	case popularity > 20:
		popularity = 4
	case popularity > 10:
		popularity = 3
	case popularity > 5:
		popularity = 2
	case popularity > 0:
		popularity = 1
	default:
		popularity = 0
	}
	return
}

