package db

import (
	"database/sql"
	"log"
	"strings"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//ApproveAccess indicates whether you are allowed to access gleepost approve, and change its settings.
func (db *DB) ApproveAccess(userID gp.UserID, netID gp.NetworkID) (perm gp.ApprovePermission, err error) {
	q := "SELECT role_level FROM user_network JOIN network ON network.master_group = user_network.network_id WHERE network.id = ? AND user_network.user_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	var level int
	err = s.QueryRow(netID, userID).Scan(&level)
	switch {
	case err != nil && err == sql.ErrNoRows:
		return perm, nil
	case err != nil:
		return perm, err
	default:
		if level > 0 {
			perm.ApproveAccess = true
		}
		if level > 1 {
			perm.LevelChange = true
		}
		return perm, nil
	}

}

//ApproveLevel returns this network's current approval level.
func (db *DB) ApproveLevel(netID gp.NetworkID) (level gp.ApproveLevel, err error) {
	q := "SELECT approval_level, approved_categories FROM network WHERE id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	var approvedCategories sql.NullString
	err = s.QueryRow(netID).Scan(&level.Level, &approvedCategories)
	if err != nil {
		return
	}
	cats := []string{}
	if approvedCategories.Valid {
		cats = strings.Split(approvedCategories.String, ",")
	}
	level.Categories = cats
	return level, nil
}

//SetApproveLevel updates this network's approval level.
func (db *DB) SetApproveLevel(netID gp.NetworkID, level int) (err error) {
	q := "UPDATE network SET approval_level = ?, approved_categories = ? WHERE id = ?"
	var categories string
	switch {
	case level == 0:
		categories = ""
	case level == 1:
		categories = "parties"
	case level == 2:
		categories = "events"
	case level == 3:
		categories = "all"
	default:
		return gp.APIerror{Reason: "That's not a valid approve level"}
	}
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(level, categories, netID)
	return
}

//PendingPosts returns all the posts in this network which are awaiting review.
func (db *DB) PendingPosts(netID gp.NetworkID) (pending []gp.PendingPost, err error) {
	pending = make([]gp.PendingPost, 0)
	//This query assumes pending = 1 and rejected = 2
	q := "SELECT wall_posts.id, wall_posts.`by`, time, text " +
		"FROM wall_posts " +
		"WHERE deleted = 0 AND pending = 1 AND network_id = ? " +
		"ORDER BY time DESC "
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(netID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var post gp.PendingPost
		var t string
		var by gp.UserID
		err = rows.Scan(&post.ID, &by, &t, &post.Text)
		if err != nil {
			return pending, err
		}
		post.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return pending, err
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
			pending = append(pending, post)
		} else {
			log.Println("Bad post: ", post)
		}
	}
	return
}

//ReviewHistory returns all the review events on this post
func (db *DB) ReviewHistory(postID gp.PostID) (history []gp.ReviewEvent, err error) {
	history = make([]gp.ReviewEvent, 0)
	q := "SELECT action, `by`, reason, `timestamp` FROM post_reviews WHERE post_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(postID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		event := gp.ReviewEvent{}
		var by gp.UserID
		var reason sql.NullString
		var t string
		err = rows.Scan(&event.Action, &by, &reason, &t)
		if err != nil {
			return
		}
		if reason.Valid {
			event.Reason = reason.String
		}
		user, UsrErr := db.GetUser(by)
		if UsrErr != nil {
			return history, UsrErr
		}
		event.By = user
		time, TimeErr := time.Parse(mysqlTime, t)
		if TimeErr != nil {
			return history, TimeErr
		}
		event.At = time
		history = append(history, event)
	}
	return
}

//PendingStatus returns the current approval status of this post. 0 = approved, 1 = pending, 2 = rejected.
func (db *DB) PendingStatus(postID gp.PostID) (pending int, err error) {
	q := "SELECT pending FROM wall_posts WHERE id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(postID).Scan(&pending)
	return
}

//ApprovePost marks this post as approved by this user.
func (db *DB) ApprovePost(userID gp.UserID, postID gp.PostID, reason string) (err error) {
	//Should be one transaction...
	q := "INSERT INTO post_reviews (post_id, action, `by`, reason) VALUES (?, 'approved', ?, ?)"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(postID, userID, reason)
	if err != nil {
		return
	}
	q2 := "UPDATE wall_posts SET pending = 0 WHERE id = ?"
	s, err = db.prepare(q2)
	if err != nil {
		return
	}
	_, err = s.Exec(postID)
	if err != nil {
		return
	}
	q3 := "UPDATE wall_posts SET time = NOW() WHERE id = ?"
	s, err = db.prepare(q3)
	if err != nil {
		return
	}
	_, err = s.Exec()
	return
}

//GetNetworkApproved returns the 20 most recent approved posts in this network.
func (db *DB) GetNetworkApproved(netID gp.NetworkID) (approved []gp.PendingPost, err error) {
	approved = make([]gp.PendingPost, 0)
	q := "SELECT wall_posts.id, wall_posts.`by`, time, text " +
		"FROM wall_posts JOIN post_reviews ON post_reviews.post_id = wall_posts.id " +
		"WHERE wall_posts.deleted = 0 AND pending = 0 AND post_reviews.action = 'approved' " +
		"AND network_id = ? " +
		"ORDER BY post_reviews.timestamp DESC LIMIT 0, 20"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(netID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var post gp.PendingPost
		var t string
		var by gp.UserID
		err = rows.Scan(&post.ID, &by, &t, &post.Text)
		if err != nil {
			return approved, err
		}
		post.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return approved, err
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
			approved = append(approved, post)
		} else {
			log.Println("Bad post: ", post)
		}
	}
	return
}

//RejectPost marks this post as 'rejected'.
func (db *DB) RejectPost(userID gp.UserID, postID gp.PostID, reason string) (err error) {
	q := "INSERT INTO post_reviews (post_id, action, `by`, reason) VALUES (?, 'rejected', ?, ?)"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(postID, userID, reason)
	if err != nil {
		return
	}
	q = "UPDATE wall_posts SET pending = 3 WHERE id = ?"
	s, err = db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(postID)
	return
}
