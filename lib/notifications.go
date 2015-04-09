package lib

import (
	"fmt"
	"log"

	"github.com/draaglom/GleepostAPI/lib/cache"
	"github.com/draaglom/GleepostAPI/lib/db"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/push"
)

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

type NotificationObserver struct {
	events chan NotificationEvent
	db     *db.DB
	cache  *cache.Cache
	pusher *push.Pusher
}

func (n NotificationObserver) Notify(e NotificationEvent) {
	n.events <- e
}

type NotificationEvent interface {
	notify(NotificationObserver) error
}

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
			pn, err := api.toIOS(notification, recipient, device.ID, api.Config.NewPushEnabled)
			if err != nil {
				log.Println("Error generating push notification:", err)
			}
			err = api.pushers["gleepost"].IOSPush(pn)
			if err != nil {
				log.Println("Error sending push notification:", err)
			} else {
				count++
			}
		case device.Type == "android":
			pn, err := api.toAndroid(notification, recipient, device.ID, api.Config.NewPushEnabled)
			if err != nil {
				log.Println("Error generating push notification:", err)
			}
			err = api.pushers["gleepost"].AndroidPush(pn)
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
