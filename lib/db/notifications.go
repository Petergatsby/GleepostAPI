package db

import (
	"database/sql"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//GetUserNotifications returns all the notifications for a given user, optionally including the seen ones.
func (db *DB) GetUserNotifications(id gp.UserID, includeSeen bool) (notifications []interface{}, err error) {
	notifications = make([]interface{}, 0)
	var notificationSelect string
	if !includeSeen {
		notificationSelect = "SELECT id, type, time, `by`, location_id, seen FROM notifications WHERE recipient = ? AND seen = 0 ORDER BY `id` DESC"
	} else {
		notificationSelect = "SELECT id, type, time, `by`, location_id, seen FROM notifications WHERE recipient = ? ORDER BY `id` DESC LIMIT 0, 20"
	}
	s, err := db.prepare(notificationSelect)
	if err != nil {
		return
	}
	rows, err := s.Query(id)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var notification gp.Notification
		var t string
		var location sql.NullInt64
		var by gp.UserID
		if err = rows.Scan(&notification.ID, &notification.Type, &t, &by, &location, &notification.Seen); err != nil {
			return
		}
		notification.Time, err = time.Parse(mysqlTime, t)
		if err != nil {
			return
		}
		notification.By, err = db.GetUser(by)
		if err != nil {
			log.Println(err)
			continue
		}
		if location.Valid {
			switch {
			case notification.Type == "liked":
				fallthrough
			case notification.Type == "approved_post":
				fallthrough
			case notification.Type == "rejected_post":
				fallthrough
			case notification.Type == "commented":
				np := gp.PostNotification{Notification: notification, Post: gp.PostID(location.Int64)}
				notifications = append(notifications, np)
			case notification.Type == "group_post":
				fallthrough
			case notification.Type == "added_group":
				ng := gp.GroupNotification{Notification: notification, Group: gp.NetworkID(location.Int64)}
				notifications = append(notifications, ng)
			default:
				notifications = append(notifications, notification)
			}
		} else {
			notifications = append(notifications, notification)
		}
	}
	return
}

//MarkNotificationsSeen records that this user has seen all their notifications.
func (db *DB) MarkNotificationsSeen(user gp.UserID, upTo gp.NotificationID) (err error) {
	s, err := db.prepare("UPDATE notifications SET seen = 1 WHERE recipient = ? AND id <= ?")
	if err != nil {
		return
	}
	_, err = s.Exec(user, upTo)
	return
}

//CreateNotification creates a notification ntype for recipient, "from" by, with a location which is interpreted as a post id if ntype is like/comment.
//TODO: All this stuff should not be in the db layer.
func (db *DB) CreateNotification(ntype string, by gp.UserID, recipient gp.UserID, location uint64) (notification interface{}, err error) {
	var res sql.Result
	notificationInsert := "INSERT INTO notifications (type, time, `by`, recipient) VALUES (?, NOW(), ?, ?)"
	notificationInsertLocation := "INSERT INTO notifications (type, time, `by`, recipient, location_id) VALUES (?, NOW(), ?, ?, ?)"
	var s *sql.Stmt
	n := gp.Notification{
		Type: ntype,
		Time: time.Now().UTC(),
		Seen: false,
	}
	n.By, err = db.GetUser(by)
	if err != nil {
		return
	}
	switch {
	case ntype == "liked":
		fallthrough
	case ntype == "commented":
		fallthrough
	case ntype == "group_post":
		fallthrough
	case ntype == "approved_post":
		fallthrough
	case ntype == "rejected_post":
		fallthrough
	case ntype == "added_group":
		s, err = db.prepare(notificationInsertLocation)
		if err != nil {
			return
		}
		res, err = s.Exec(ntype, by, recipient, location)
	default:
		s, err = db.prepare(notificationInsert)
		if err != nil {
			return
		}
		res, err = s.Exec(ntype, by, recipient)
	}
	if err != nil {
		return
	}
	id, iderr := res.LastInsertId()
	if iderr != nil {
		return n, iderr
	}
	n.ID = gp.NotificationID(id)
	switch {
	case ntype == "liked":
		fallthrough
	case ntype == "approved_post":
		fallthrough
	case ntype == "rejected_post":
		fallthrough
	case ntype == "commented":
		np := gp.PostNotification{Notification: n, Post: gp.PostID(location)}
		return np, nil
	case ntype == "group_post":
		fallthrough
	case ntype == "added_group":
		ng := gp.GroupNotification{Notification: n, Group: gp.NetworkID(location)}
		return ng, nil
	default:
		return n, nil
	}
}
