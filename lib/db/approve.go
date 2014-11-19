package db

import (
	"database/sql"
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

//NoSuchGroup is returned when a lookup of a group's master-group fails.
var NoSuchGroup = gp.APIerror{Reason: "No such group"}

//MasterGroup returns the id of the group which administrates this network, or NoSuchGroup if there is none.
func (db *DB) MasterGroup(netID gp.NetworkID) (master gp.NetworkID, err error) {
	q := "SELECT master_group FROM network WHERE id = ? AND MASTER IS NOT NULL"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(netID).Scan(&master)
	if err == sql.ErrNoRows {
		err = NoSuchGroup
	}
	return
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
		categories = "party"
	case level == 2:
		categories = "event"
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
func (db *DB) PendingPosts(netID gp.NetworkID) (pending []gp.PostSmall, err error) {
	pending = make([]gp.PostSmall, 0)
	//This query assumes pending = 1 and rejected = 2
	q := "SELECT wall_posts.id, wall_posts.`by`, time, text, network_id " +
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
	return db.scanPostRows(rows, false)
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
	_, err = s.Exec(postID)
	return
}

//GetNetworkApproved returns the 20 most recent approved posts in this network.
func (db *DB) GetNetworkApproved(netID gp.NetworkID, mode int, index int64, count int) (approved []gp.PostSmall, err error) {
	approved = make([]gp.PostSmall, 0)
	q := "SELECT wall_posts.id, wall_posts.`by`, time, text, network_id " +
		"FROM wall_posts JOIN post_reviews ON post_reviews.post_id = wall_posts.id " +
		"WHERE wall_posts.deleted = 0 AND pending = 0 AND post_reviews.action = 'approved' " +
		"AND network_id = ? "
	switch {
	case mode == gp.OSTART:
		q += "ORDER BY post_reviews.timestamp DESC LIMIT ?, ?"
	case mode == gp.OAFTER:
		q += "AND wall_posts.time > (SELECT time FROM wall_posts WHERE post_id = ?) " +
			"ORDER BY post_reviews.timestamp DESC LIMIT 0, ?"
	case mode == gp.OBEFORE:
		q += "AND wall_posts.time < (SELECT time FROM wall_posts WHERE post_id = ?) " +
			"ORDER BY post_reviews.timestamp DESC LIMIT 0, ?"
	}
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(netID, index, count)
	if err != nil {
		return
	}
	defer rows.Close()
	return db.scanPostRows(rows, false)
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
	q = "UPDATE wall_posts SET pending = 2 WHERE id = ?"
	s, err = db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(postID)
	return
}

//ResubmitPost marks this post as 'pending' again.
func (db *DB) ResubmitPost(userID gp.UserID, postID gp.PostID, reason string) (err error) {
	s, err := db.prepare("INSERT INTO post_reviews (post_id, action, `by`, reason) VALUES (?, 'edited', ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(postID, userID, reason)
	if err != nil {
		return
	}
	s, err = db.prepare("UPDATE wall_posts SET pending = 1 WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(postID)
	return
}

//GetNetworkRejected returns the posts in this network which have been rejected.
func (db *DB) GetNetworkRejected(netID gp.NetworkID, mode int, index int64, count int) (rejected []gp.PostSmall, err error) {
	rejected = make([]gp.PostSmall, 0)
	q := "SELECT wall_posts.id, wall_posts.`by`, time, text, network_id " +
		"FROM wall_posts JOIN post_reviews ON post_reviews.post_id = wall_posts.id " +
		"WHERE wall_posts.deleted = 0 AND pending = 2 AND post_reviews.action = 'rejected' " +
		"AND network_id = ? "
	switch {
	case mode == gp.OSTART:
		q += "ORDER BY post_reviews.timestamp DESC LIMIT ?, ?"
	case mode == gp.OAFTER:
		q += "AND wall_posts.time > (SELECT time FROM wall_posts WHERE post_id = ?) " +
			"ORDER BY post_reviews.timestamp DESC LIMIT 0, ?"
	case mode == gp.OBEFORE:
		q += "AND wall_posts.time < (SELECT time FROM wall_posts WHERE post_id = ?) " +
			"ORDER BY post_reviews.timestamp DESC LIMIT 0, ?"
	}
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(netID, index, count)
	if err != nil {
		return
	}
	defer rows.Close()
	return db.scanPostRows(rows, false)

}

//UserPendingPosts returns all this user's pending posts.
func (db *DB) UserPendingPosts(userID gp.UserID) (pending []gp.PostSmall, err error) {
	pending = make([]gp.PostSmall, 0)
	//This query assumes pending = 1 and rejected = 2
	q := "SELECT DISTINCT wall_posts.id, wall_posts.`by`, time, text, network_id " +
		"FROM wall_posts " +
		"LEFT JOIN post_reviews ON wall_posts.id = post_reviews.post_id " +
		"WHERE deleted = 0 AND pending > 0 AND wall_posts.`by` = ? " +
		"ORDER BY CASE WHEN MAX(post_reviews.timestamp) IS NULL THEN wall_posts.time ELSE MAX(post_reviews.timestamp) END DESC "
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(userID)
	if err != nil {
		return
	}
	defer rows.Close()
	return db.scanPostRows(rows, false)
}
