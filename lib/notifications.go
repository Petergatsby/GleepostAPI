package lib

import (
	"errors"
	"fmt"
	"log"

	"github.com/draaglom/GleepostAPI/lib/cache"
	"github.com/draaglom/GleepostAPI/lib/db"
	"github.com/draaglom/GleepostAPI/lib/gp"
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
	return api.db.GetUserNotifications(id, includeSeen)
}

//MarkNotificationsSeen marks all notifications up to upTo seen for this user.
func (api *API) MarkNotificationsSeen(id gp.UserID, upTo gp.NotificationID) (err error) {
	return api.db.MarkNotificationsSeen(id, upTo)
}

//createNotification creates a new gleepost notification. location is the id of the object where the notification happened - a post id if the notification is "liked" or "commented", or a network id if the notification type is "added_group". Otherwise, the location will be ignored.
func (n NotificationObserver) createNotification(ntype string, by gp.UserID, recipient gp.UserID, postID gp.PostID, netID gp.NetworkID, preview string) (err error) {
	if len(preview) > 97 {
		preview = preview[:97] + "..."
	}
	notification, err := n.db.CreateNotification(ntype, by, recipient, postID, netID, preview)
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
	db     *db.DB
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
func NewObserver(db *db.DB, cache *cache.Cache, pusher *push.Pusher) NotificationObserver {
	events := make(chan NotificationEvent)
	n := NotificationObserver{events: events, db: db, cache: cache, pusher: pusher}
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
	creator, err := n.db.NetworkCreator(p.netID)
	if err == nil && (creator == p.userID) && !p.pending {
		users, err := n.db.GetNetworkUsers(p.netID)
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

//Push takes a gleepost notification and sends it as a push notification to all of recipient's devices.
func (n NotificationObserver) push(notification gp.Notification, recipient gp.UserID) {
	devices, err := n.db.GetDevices(recipient, "gleepost")
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
		var group gp.Group
		group, err = n.db.GetNetwork(notification.Group)
		if err != nil {
			return
		}
		d.LocArgs = append(d.LocArgs, group.Name)
		pn.Set("group-id", group.ID)
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
		var group gp.Group
		group, err = n.db.GetNetwork(notification.Group)
		if err != nil {
			return
		}
		data["group-id"] = notification.Group
		data["group-name"] = group.Name
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
	default:
		return nil, errors.New("Unknown notification type")
	}
	msg = gcm.NewMessage(data, device)
	msg.CollapseKey = CollapseKey
	msg.TimeToLive = 0
	return
}

func (n NotificationObserver) badgeCount(user gp.UserID) (count int, err error) {
	notifications, err := n.db.GetUserNotifications(user, false)
	if err != nil {
		log.Println(err)
		return
	}
	count = len(notifications)
	unread, e := n.db.UnreadMessageCount(user, true)
	if e == nil {
		count += unread
	} else {
		log.Println(e)
	}
	log.Printf("Badging %d with %d notifications (%d from unread)\n", user, count, unread)
	return
}
