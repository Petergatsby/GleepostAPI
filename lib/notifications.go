package lib

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/cache"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/psc"
	"github.com/draaglom/GleepostAPI/lib/push"
	"github.com/draaglom/apns"
	"github.com/draaglom/gcm"
)

func toLocKey(notificationType string) (locKey string) {
	switch {
	case notificationType == "added_group":
		return "GROUP"
	default:
		return notificationType
	}
}

//GetUserNotifications returns all unseen notifications for this user, and the seen ones as well if includeSeen is true.
func (api *API) GetUserNotifications(id gp.UserID, includeSeen bool) (notifications []gp.Notification, err error) {
	return api.getUserNotifications(id, includeSeen)
}

//MarkNotificationsSeen marks all notifications up to upTo seen for this user.
func (api *API) MarkNotificationsSeen(id gp.UserID, upTo gp.NotificationID) (err error) {
	return api.markNotificationsSeen(id, upTo)
}

//createNotification creates a new gleepost notification. location is the id of the object where the notification happened - a post id if the notification is "liked" or "commented", or a network id if the notification type is "added_group". Otherwise, the location will be ignored.
func (n NotificationObserver) createNotification(ntype string, by gp.UserID, recipient gp.UserID, postID gp.PostID, netID gp.NetworkID, preview string) (err error) {
	if len(preview) > 97 {
		preview = preview[:97] + "..."
	}
	notification, err := n._createNotification(ntype, by, recipient, postID, netID, preview)
	if err == nil {
		n.push(notification, recipient)
		go n.cache.PublishEvent("notification", "/notifications", notification, []string{NotificationChannelKey(recipient)})
	}
	return
}

//NotificationChannelKey returns the channel used for this user's notifications.
func NotificationChannelKey(id gp.UserID) (channel string) {
	return fmt.Sprintf("n:%d", id)
}

//NotificationObserver has the responsibility of producing Notifications for users.
type NotificationObserver struct {
	events chan NotificationEvent
	db     *sql.DB
	sc     *psc.StatementCache
	cache  *cache.Cache
	pusher *push.Pusher
}

//Notify tells the NotificationObserver an event has happened, potentially triggering a notification.
func (n NotificationObserver) Notify(e NotificationEvent) {
	n.events <- e
}

//NotificationEvent is any event which might trigger a notification.
type NotificationEvent interface {
	notify(NotificationObserver) error
}

//NewObserver creates a NotificationObserver
func NewObserver(db *sql.DB, cache *cache.Cache, pusher *push.Pusher, sc *psc.StatementCache) NotificationObserver {
	events := make(chan NotificationEvent)
	n := NotificationObserver{events: events, db: db, sc: sc, cache: cache, pusher: pusher}
	go n.spin()
	return n
}

func (n NotificationObserver) spin() {
	for {
		event := <-n.events
		event.notify(n)
	}
}

type postEvent struct {
	userID  gp.UserID
	netID   gp.NetworkID
	postID  gp.PostID
	pending bool
}

func (p postEvent) notify(n NotificationObserver) error {
	creator, err := n.networkCreator(p.netID)
	if err == nil && (creator == p.userID) && !p.pending {
		users, err := getNetworkUsers(n.sc, p.netID)
		if err != nil {
			return err
		}
		for _, u := range users {
			if u.ID != p.userID {
				n.createNotification("group_post", p.userID, u.ID, p.postID, p.netID, "")
			}
		}
	}
	return nil
}

type approvedEvent struct {
	userID      gp.UserID
	recipientID gp.UserID
	postID      gp.PostID
}

func (a approvedEvent) notify(n NotificationObserver) error {
	err := n.createNotification("approved_post", a.userID, a.recipientID, a.postID, 0, "")
	return err
}

type rejectedEvent struct {
	userID      gp.UserID
	recipientID gp.UserID
	postID      gp.PostID
}

func (r rejectedEvent) notify(n NotificationObserver) error {
	err := n.createNotification("rejected_post", r.userID, r.recipientID, r.postID, 0, "")
	return err
}

type addedGroupEvent struct {
	userID  gp.UserID
	addeeID gp.UserID
	netID   gp.NetworkID
}

func (a addedGroupEvent) notify(n NotificationObserver) error {
	err := n.createNotification("added_group", a.userID, a.addeeID, 0, a.netID, "")
	return err
}

type commentEvent struct {
	userID      gp.UserID
	recipientID gp.UserID
	postID      gp.PostID
	text        string
}

func (c commentEvent) notify(n NotificationObserver) (err error) {
	if c.userID != c.recipientID {
		err = n.createNotification("commented", c.userID, c.recipientID, c.postID, 0, c.text)
	}
	return err
}

type likeEvent struct {
	userID      gp.UserID
	recipientID gp.UserID
	postID      gp.PostID
}

func (l likeEvent) notify(n NotificationObserver) (err error) {
	if l.userID != l.recipientID {
		err = n.createNotification("liked", l.userID, l.recipientID, l.postID, 0, "")
	}
	return err
}

type voteEvent struct {
	userID gp.UserID
	postID gp.PostID
}

func (v voteEvent) notify(n NotificationObserver) (err error) {
	owner, err := postOwner(n.sc, v.postID)
	if err != nil {
		return
	}
	if v.userID != owner {
		err = n.createNotification("poll_vote", v.userID, owner, v.postID, 0, "")
	}
	return
}

type requestEvent struct {
	userID  gp.UserID
	groupID gp.NetworkID
}

func (r requestEvent) notify(n NotificationObserver) (err error) {
	admins, err := api.getNetworkAdmins(r.groupID)
	if err != nil {
		return
	}
	creator, err := api.networkCreator(r.groupID)
	if err != nil {
		return
	}
	err = n.createNotification("group_request", r.userID, creator.ID, 0, r.groupID, "")
	if err != nil {
		return
	}
	for _, admin := range admins {
		err = n.createNotification("group_request", r.userID, admin.ID, 0, r.groupID, "")
		if err != nil {
			return
		}
	}
	return nil
}

//Push takes a gleepost notification and sends it as a push notification to all of recipient's devices.
func (n NotificationObserver) push(notification gp.Notification, recipient gp.UserID) {
	devices, err := getDevices(n.sc, recipient, "gleepost")
	if err != nil {
		log.Println(err)
		return
	}
	count := 0
	for _, device := range devices {
		switch {
		case device.Type == "ios":
			pn, err := n.toIOS(notification, recipient, device.ID)
			if err != nil {
				log.Println("Error generating push notification:", err)
			}
			err = n.pusher.IOSPush(pn)
			if err != nil {
				log.Println("Error sending push notification:", err)
			} else {
				count++
			}
		case device.Type == "android":
			pn, err := n.toAndroid(notification, recipient, device.ID)
			if err != nil {
				log.Println("Error generating push notification:", err)
			}
			err = n.pusher.AndroidPush(pn)
			if err != nil {
				log.Println("Error sending push notification:", err)
			} else {
				count++
			}
		}
	}
	if count == len(devices) {
		log.Printf("Successfully sent %d notifications to %d\n", count, recipient)
	} else {
		log.Printf("Failed to send some notifications (%d of %d were successes) to %d\n", count, len(devices), recipient)
	}
}

func (n NotificationObserver) toIOS(notification gp.Notification, recipient gp.UserID, device string) (pn *apns.PushNotification, err error) {
	payload := apns.NewPayload()
	d := apns.NewAlertDictionary()
	pn = apns.NewPushNotification()
	pn.DeviceToken = device
	badge, err := n.badgeCount(recipient)
	if err != nil {
		return
	}
	payload.Badge = badge
	d.LocKey = toLocKey(notification.Type)
	d.LocArgs = []string{notification.By.Name}
	switch {
	case notification.Type == "added_group" || notification.Type == "group_post":
		var name string
		name, err = groupName(n.sc, notification.Group)
		if err != nil {
			return
		}
		d.LocArgs = append(d.LocArgs, name)
		pn.Set("group-id", notification.Group)
	case notification.Type == "accepted_you":
		pn.Set("accepter-id", notification.By.ID)
	case notification.Type == "added_you":
		pn.Set("adder-id", notification.By.ID)
	case notification.Type == "liked":
		pn.Set("liker-id", notification.By.ID)
		pn.Set("post-id", notification.Post)
	case notification.Type == "commented":
		pn.Set("commenter-id", notification.By.ID)
		pn.Set("post-id", notification.Post)
	case notification.Type == "approved_post":
		pn.Set("approver-id", notification.By.ID)
		pn.Set("post-id", notification.Post)
	case notification.Type == "rejected_post":
		pn.Set("rejecter-id", notification.By.ID)
		pn.Set("post-id", notification.Post)
	case notification.Type == "poll_vote":
		pn.Set("voter-id", notification.By.ID)
		pn.Set("post-id", notification.Post)
	}
	pn.AddPayload(payload)
	return
}

func (n NotificationObserver) toAndroid(notification gp.Notification, recipient gp.UserID, device string) (msg *gcm.Message, err error) {
	var CollapseKey string
	data := make(map[string]interface{})
	data["type"] = toLocKey(notification.Type)
	data["for"] = recipient
	switch {
	case notification.Type == "added_group" || notification.Type == "group_post":
		var name string
		name, err = groupName(n.sc, notification.Group)
		if err != nil {
			return
		}
		data["group-id"] = notification.Group
		data["group-name"] = name
		switch {
		case notification.Type == "group_post":
			data["poster"] = notification.By.Name
			CollapseKey = "Somoene posted in your group."
		default:
			data["adder"] = notification.By.Name
			CollapseKey = "You've been added to a group"
		}
	case notification.Type == "added_you":
		data["adder"] = notification.By.Name
		data["adder-id"] = notification.By.ID
		CollapseKey = "Someone added you to their contacts."
	case notification.Type == "accepted_you":
		data["accepter"] = notification.By.Name
		data["accepter-id"] = notification.By.ID
		CollapseKey = "Someone accepted your contact request."
	case notification.Type == "liked":
		data["liker"] = notification.By.Name
		data["liker-id"] = notification.By.ID
		data["post-id"] = notification.Post
		CollapseKey = "Someone liked your post."
	case notification.Type == "commented":
		data["commenter"] = notification.By.Name
		data["commenter-id"] = notification.By.ID
		data["post-id"] = notification.Post
		CollapseKey = "Someone commented on your post."
	case notification.Type == "poll_vote":
		data["voter"] = notification.By.Name
		data["voter-id"] = notification.By.ID
		data["post-id"] = notification.Post
		CollapseKey = "Someone voted in your poll."
	default:
		return nil, errors.New("Unknown notification type")
	}
	msg = gcm.NewMessage(data, device)
	msg.CollapseKey = CollapseKey
	msg.TimeToLive = 0
	return
}

func (n NotificationObserver) badgeCount(user gp.UserID) (count int, err error) {
	count, err = unreadNotificationCount(n.sc, user)
	if err != nil {
		log.Println(err)
		return
	}
	unread, e := unreadMessageCount(n.sc, user, true)
	if e == nil {
		count += unread
	} else {
		log.Println(e)
	}
	log.Printf("Badging %d with %d notifications (%d from unread)\n", user, count, unread)
	return
}

func unreadNotificationCount(sc *psc.StatementCache, userID gp.UserID) (count int, err error) {
	s, err := sc.Prepare("SELECT COUNT(*) FROM notifications WHERE recipient = ? AND seen = 0")
	if err != nil {
		return
	}
	err = s.QueryRow(userID).Scan(&count)
	return
}

//GetUserNotifications returns all the notifications for a given user, optionally including the seen ones.
func (api *API) getUserNotifications(id gp.UserID, includeSeen bool) (notifications []gp.Notification, err error) {
	notifications = make([]gp.Notification, 0)
	var notificationSelect string
	if !includeSeen {
		notificationSelect = "SELECT id, type, time, `by`, post_id, network_id, preview_text, seen FROM notifications WHERE recipient = ? AND seen = 0 ORDER BY `id` DESC"
	} else {
		notificationSelect = "SELECT id, type, time, `by`, post_id, network_id, preview_text, seen FROM notifications WHERE recipient = ? ORDER BY `id` DESC LIMIT 0, 20"
	}
	s, err := api.sc.Prepare(notificationSelect)
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
		notification.By, err = api.getUser(by)
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
func (api *API) markNotificationsSeen(user gp.UserID, upTo gp.NotificationID) (err error) {
	s, err := api.sc.Prepare("UPDATE notifications SET seen = 1 WHERE recipient = ? AND id <= ?")
	if err != nil {
		return
	}
	_, err = s.Exec(user, upTo)
	return
}

//CreateNotification creates a notification ntype for recipient, "from" by, with an optional post, network and preview text.
//TODO: All this stuff should not be in the db layer.
func (n NotificationObserver) _createNotification(ntype string, by gp.UserID, recipient gp.UserID, postID gp.PostID, netID gp.NetworkID, preview string) (notification gp.Notification, err error) {
	var res sql.Result
	notificationInsert := "INSERT INTO notifications (type, time, `by`, recipient, post_id, network_id, preview_text) VALUES (?, NOW(), ?, ?, ?, ?, ?)"
	var s *sql.Stmt
	notification = gp.Notification{
		Type: ntype,
		Time: time.Now().UTC(),
		Seen: false,
	}
	notification.By, err = getUser(n.sc, by)
	if err != nil {
		return
	}
	s, err = n.sc.Prepare(notificationInsert)
	if err != nil {
		return
	}
	res, err = s.Exec(ntype, by, recipient, postID, netID, preview)
	if err != nil {
		return
	}
	id, iderr := res.LastInsertId()
	if iderr != nil {
		return notification, iderr
	}
	notification.ID = gp.NotificationID(id)
	if postID > 0 {
		notification.Post = postID
	}
	if netID > 0 {
		notification.Group = netID
	}
	if len(preview) > 0 {
		notification.Preview = preview
	}
	return notification, nil
}

func (n NotificationObserver) networkCreator(netID gp.NetworkID) (creator gp.UserID, err error) {
	qCreator := "SELECT creator FROM network WHERE id = ?"
	s, err := n.sc.Prepare(qCreator)
	if err != nil {
		return
	}
	err = s.QueryRow(netID).Scan(&creator)
	return
}
