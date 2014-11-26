package db

import (
	"database/sql"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//GetUserNotifications returns all the notifications for a given user, optionally including the seen ones.
func (db *DB) GetUserNotifications(id gp.UserID, includeSeen bool) (notifications []gp.Notification, err error) {
	notifications = make([]gp.Notification, 0)
	var notificationSelect string
	if !includeSeen {
		notificationSelect = "SELECT id, type, time, `by`, post_id, network_id, preview_text, seen FROM notifications WHERE recipient = ? AND seen = 0 ORDER BY `id` DESC"
	} else {
		notificationSelect = "SELECT id, type, time, `by`, post_id, network_id, preview_text, seen FROM notifications WHERE recipient = ? ORDER BY `id` DESC LIMIT 0, 20"
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
		var postID, netID sql.NullInt64
		var preview sql.NullString
		var by gp.UserID
		if err = rows.Scan(&notification.ID, &notification.Type, &t, &by, &postID, &netID, &preview, &notification.Seen); err != nil {
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
		if postID.Valid {
			notification.Post = gp.PostID(postID.Int64)
		}
		if netID.Valid {
			notification.Group = gp.NetworkID(netID.Int64)
		}
		if preview.Valid {
			notification.Preview = preview.String
		}
		notifications = append(notifications, notification)
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

//CreateNotification creates a notification ntype for recipient, "from" by, with an optional post, network and preview text.
//TODO: All this stuff should not be in the db layer.
func (db *DB) CreateNotification(ntype string, by gp.UserID, recipient gp.UserID, postID gp.PostID, netID gp.NetworkID, preview string) (notification gp.Notification, err error) {
	var res sql.Result
	notificationInsert := "INSERT INTO notifications (type, time, `by`, recipient, post_id, network_id, preview_text) VALUES (?, NOW(), ?, ?, ?, ?, ?)"
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
	s, err = db.prepare(notificationInsert)
	if err != nil {
		return
	}
	res, err = s.Exec(ntype, by, recipient, postID, netID, preview)
	if err != nil {
		return
	}
	id, iderr := res.LastInsertId()
	if iderr != nil {
		return n, iderr
	}
	n.ID = gp.NotificationID(id)
	if postID > 0 {
		n.Post = postID
	}
	if netID > 0 {
		n.Group = netID
	}
	if len(preview) > 0 {
		n.Preview = preview
	}
	return n, nil
}
