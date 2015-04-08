package lib

import (
	"fmt"

	"github.com/draaglom/GleepostAPI/lib/gp"
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
func (api *API) createNotification(ntype string, by gp.UserID, recipient gp.UserID, postID gp.PostID, netID gp.NetworkID, preview string) (err error) {
	if len(preview) > 97 {
		preview = preview[:97] + "..."
	}
	notification, err := api.db.CreateNotification(ntype, by, recipient, postID, netID, preview)
	if err == nil {
		api.push(notification, recipient)
		go api.cache.PublishEvent("notification", "/notifications", notification, []string{NotificationChannelKey(recipient)})
	}
	return
}

//NotificationChannelKey returns the channel used for this user's notifications.
func NotificationChannelKey(id gp.UserID) (channel string) {
	return fmt.Sprintf("n:%d", id)
}

type NotificationObserver struct {
	events chan NotificationEvent
	api    *API
}

func (n NotificationObserver) Notify(e NotificationEvent) {
	n.events <- e
}

type NotificationEvent interface {
}

func NewObserver() NotificationObserver {
	events := make(chan NotificationEvent)
	n := NotificationObserver{events: events}
	go n.spin()
	return n
}

func (n NotificationObserver) spin() {
	for {
		event := <-n.events
		switch e := event.(type) {
		case postEvent:
			n.api.maybeNotify(e.userID, e.netID, e.postID, e.pending)
		}
	}
}

type postEvent struct {
	userID  gp.UserID
	netID   gp.NetworkID
	postID  gp.PostID
	pending bool
}
