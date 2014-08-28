package lib

import (
	"errors"
	"log"
	"time"

	"github.com/anachronistic/apns"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/gcm"
)

func (api *API) notify(user gp.UserID) {
	payload := apns.NewPayload()
	payload.Alert = "Sup"
	payload.Badge = 1337
	payload.Sound = "default"

	devices, err := api.GetDevices(user)
	if err != nil {
		log.Println(err)
	}
	for _, device := range devices {
		if device.Type == "ios" {
			pn := apns.NewPushNotification()
			pn.DeviceToken = device.ID
			pn.AddPayload(payload)
			err := api.push.IOSPush(pn)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (api *API) notificationPush(user gp.UserID) {
	notifications, err := api.GetUserNotifications(user, false)
	if err != nil {
		log.Println(err)
		return
	}
	badge := len(notifications)
	unread, err := api.UnreadMessageCount(user)
	if err == nil {
		badge += unread
	}
	log.Printf("Badging %d with %d notifications (%d from unread)\n", user, badge, unread)

	devices, err := api.GetDevices(user)
	if err != nil {
		log.Println(err)
	}
	count := 0
	for _, device := range devices {
		switch {
		case device.Type == "ios":
			err = api.iosBadge(device.ID, badge)
			if err != nil {
				log.Println("Error sending push notification:", err)
			} else {
				count++
			}
		case device.Type == "android":
			err = api.androidNotification(device.ID, badge, user)
			if err != nil {
				log.Println("Error sending push notification:", err)
			} else {
				count++
			}
		}
	}
	log.Printf("Badged %d's %d devices\n", user, count)
}

func (api *API) newConversationPush(initiator gp.User, other gp.UserID, conv gp.ConversationID) (err error) {
	log.Printf("Notifiying user %d that they've got a new conversation with %s (%d)\n", other, initiator.Name, initiator.ID)
	devices, e := api.GetDevices(other)
	if e != nil {
		log.Println(e)
		return
	}
	count := 0
	for _, device := range devices {
		switch {
		case device.Type == "ios":
			err = api.iOSNewConversationNotification(device.ID, conv, other, initiator)
			if err != nil {
				log.Println("Error sending new conversation push notification:", err)
			} else {
				count++
			}
		case device.Type == "android":
			err = api.androidNewConversationNotification(device.ID, conv, other, initiator)
			if err != nil {
				log.Println("Error sending new conversation push notification:", err)
			} else {
				count++
			}
		}
	}
	log.Printf("Notified %d's %d devices\n", other, count)
	return

}

func (api *API) messagePush(message gp.Message, convID gp.ConversationID) {
	log.Println("Trying to send a push notification")
	recipients := api.GetParticipants(convID, false)
	for _, user := range recipients {
		if user.ID != message.By.ID {
			log.Println("Trying to send a push notification to", user.Name)
			devices, err := api.GetDevices(user.ID)
			if err != nil {
				log.Println(err)
			}
			count := 0
			for _, device := range devices {
				log.Println("Sending push notification to device: ", device)
				switch {
				case device.Type == "ios":
					err = api.iosPushMessage(device.ID, message, convID, user.ID)
					if err != nil {
						log.Println(err)
					} else {
						count++
					}
				case device.Type == "android":
					err = api.androidPushMessage(device.ID, message, convID, user.ID)
					if err != nil {
						log.Println(err)
					} else {
						count++
					}
				}
			}
			log.Printf("Sent notification to %s's %d devices\n", user.Name, count)
		}
	}
}

//iosBadge sets this device's badge, or returns an error.
func (api *API) iosBadge(device string, badge int) (err error) {
	payload := apns.NewPayload()
	payload.Badge = badge
	pn := apns.NewPushNotification()
	pn.DeviceToken = device
	pn.AddPayload(payload)
	err = api.push.IOSPush(pn)
	return
}

//androidNotification sends a "You have new notifications" push to this device.
//user is included because GCM doesn't really like deregistering, so we include the
//recipient id in the notification so the app can filter itself.
func (api *API) androidNotification(device string, count int, user gp.UserID) (err error) {
	data := map[string]interface{}{"count": count, "for": user}
	msg := gcm.NewMessage(data, device)
	msg.CollapseKey = "New Notification"

	err = api.push.AndroidPush(msg)
	return
}

func (api *API) iosPushMessage(device string, message gp.Message, convID gp.ConversationID, user gp.UserID) (err error) {
	payload := apns.NewPayload()
	d := apns.NewAlertDictionary()
	d.LocKey = "MSG"
	d.LocArgs = []string{message.By.Name}
	if len(message.Text) > 64 {
		d.LocArgs = append(d.LocArgs, message.Text[:64]+"...")
	} else {
		d.LocArgs = append(d.LocArgs, message.Text)
	}
	payload.Alert = d
	payload.Sound = "default"
	notifications, err := api.GetUserNotifications(user, false)
	if err != nil {
		log.Println(err)
		return
	}
	payload.Badge = len(notifications)
	unread, err := api.UnreadMessageCount(user)
	if err == nil {
		payload.Badge += unread
	}
	log.Printf("Badging %d with %d notifications (%d from unread messages)", user, payload.Badge, unread)
	pn := apns.NewPushNotification()
	pn.DeviceToken = device
	pn.AddPayload(payload)
	pn.Set("conv", convID)
	err = api.push.IOSPush(pn)
	return
}

func (api *API) androidPushMessage(device string, message gp.Message, convID gp.ConversationID, user gp.UserID) (err error) {
	data := map[string]interface{}{"type": "MSG", "sender": message.By.Name, "sender-id": message.By.ID, "conv": convID, "for": user}
	if len(message.Text) > 3200 {
		data["text"] = message.Text[:3200] + "..."
	} else {
		data["text"] = message.Text
	}
	msg := gcm.NewMessage(data, device)
	msg.TimeToLive = 0
	return api.push.AndroidPush(msg)
}

//CheckFeedbackService receives any bad device tokens from APNS and deletes the device.
func (api *API) CheckFeedbackService() {
	url := "feedback.sandbox.push.apple.com:2196"
	if api.Config.APNS.Production {
		url = "feedback.push.apple.com:2196"
	}
	client := apns.NewClient(url, api.Config.APNS.CertFile, api.Config.APNS.KeyFile)
	log.Println("Connected to feedback service", url)
	go client.ListenForFeedback()
	for {
		select {
		case resp := <-apns.FeedbackChannel:
			log.Println("Bad device:", resp.DeviceToken, resp.Timestamp)
			api.DeviceFeedback(resp.DeviceToken, resp.Timestamp)
		case <-apns.ShutdownChannel:
			log.Println("feedback service ended")
			return
		}
	}
}

//FeedbackDaemon checks the APNS feedback service every frequency seconds.
func (api *API) FeedbackDaemon(frequency int) {
	duration := time.Duration(frequency) * time.Second
	c := time.Tick(duration)
	for {
		<-c
		go api.CheckFeedbackService()
	}
}

func (api *API) iOSNewConversationNotification(device string, conv gp.ConversationID, user gp.UserID, with gp.User) (err error) {
	payload := apns.NewPayload()
	d := apns.NewAlertDictionary()
	d.LocKey = "NEW_CONV"
	d.LocArgs = []string{with.Name}
	payload.Alert = d
	payload.Sound = "default"
	notifications, err := api.GetUserNotifications(user, false)
	if err != nil {
		log.Println(err)
		return
	}
	payload.Badge = len(notifications)
	unread, err := api.UnreadMessageCount(user)
	if err == nil {
		payload.Badge += unread
	}
	log.Printf("Badging %d with %d notifications (%d from unread messages)", user, payload.Badge, unread)
	pn := apns.NewPushNotification()
	pn.DeviceToken = device
	pn.AddPayload(payload)
	pn.Set("conv", conv)
	err = api.push.IOSPush(pn)
	return
}

func (api *API) androidNewConversationNotification(device string, conv gp.ConversationID, user gp.UserID, with gp.User) (err error) {
	data := map[string]interface{}{"type": "NEW_CONV", "with": with.Name, "with-id": with.ID, "conv": conv, "for": user}
	msg := gcm.NewMessage(data, device)
	msg.TimeToLive = 0
	msg.CollapseKey = "You have a new conversation!"
	return api.push.AndroidPush(msg)
}

//MassNotification sends an update notification to all devices which, when pressed, prompts the user to update if version > installed version.
func (api *API) MassNotification(message string, version string, platform string) (count int, err error) {
	devices, err := api.db.GetAllDevices(platform)
	if err != nil {
		return
	}
	if len(devices) == 0 {
		return 0, errors.New("no devices on that platform")
	}
	for _, device := range devices {
		switch {
		case device.Type == "ios":
			err = api.iOSUpdateNotification(device, message, version)
			if err == nil {
				count++
			} else {
				log.Println(err)
			}
		default:
		}
	}
	return
}

func (api *API) iOSUpdateNotification(device gp.Device, message string, version string) (err error) {
	payload := apns.NewPayload()
	payload.Alert = message
	payload.Sound = "default"
	notifications, err := api.GetUserNotifications(device.User, false)
	if err != nil {
		log.Println(err)
		return
	}
	payload.Badge = len(notifications)
	unread, err := api.UnreadMessageCount(device.User)
	if err == nil {
		payload.Badge += unread
	}
	log.Printf("mass notification: badging %d with %d notifications (%d from unread messages)", device.User, payload.Badge, unread)
	pn := apns.NewPushNotification()
	pn.DeviceToken = device.ID
	pn.AddPayload(payload)
	pn.Set("version", version)
	err = api.push.IOSPush(pn)
	return
}

func (api *API) badgeCount(user gp.UserID) (count int, err error) {
	notifications, err := api.GetUserNotifications(user, false)
	if err != nil {
		log.Println(err)
		return
	}
	count = len(notifications)
	unread, e := api.UnreadMessageCount(user)
	if e == nil {
		count += unread
	} else {
		log.Println(err)
	}
	return
}

func (api *API) toIOS(notification interface{}, recipient gp.UserID, device string, newPush bool) (pn *apns.PushNotification, err error) {
	alert := true
	payload := apns.NewPayload()
	d := apns.NewAlertDictionary()
	pn = apns.NewPushNotification()
	pn.DeviceToken = device
	badge, err := api.badgeCount(recipient)
	if err != nil {
		log.Println("Error getting badge:", err)
	} else {
		payload.Badge = badge
	}
	switch n := notification.(type) {
	case gp.GroupNotification:
		d.LocKey = "GROUP"
		var group gp.Group
		group, err = api.getNetwork(n.Group)
		if err != nil {
			log.Println(err)
			return
		}
		d.LocArgs = []string{n.By.Name, group.Name}
		pn.Set("group-id", group.ID)
	case gp.Notification:
		switch {
		case n.Type == "accepted_you":
			d.LocKey = "accepted_you"
			d.LocArgs = []string{n.By.Name}
			pn.Set("accepter-id", n.By.ID)
		case n.Type == "added_you":
			d.LocKey = "added_you"
			d.LocArgs = []string{n.By.Name}
			pn.Set("adder-id", n.By.ID)
		default:
			alert = false
		}
	case gp.PostNotification:
		switch {
		case n.Type == "liked" && newPush:
			d.LocKey = "liked"
			d.LocArgs = []string{n.By.Name}
			pn.Set("liker-id", n.By.ID)
			pn.Set("post-id", n.Post)
		case n.Type == "commented" && newPush:
			d.LocKey = "commented"
			d.LocArgs = []string{n.By.Name}
			pn.Set("commenter-id", n.By.ID)
			pn.Set("post-id", n.Post)
		default:
			alert = false
		}
	}
	if alert {
		payload.Alert = d
		payload.Sound = "default"
	}
	pn.AddPayload(payload)
	log.Println(pn)
	return
}

func (api *API) toAndroid(notification interface{}, recipient gp.UserID, device string, newPush bool) (msg *gcm.Message, err error) {
	unknown := false
	var CollapseKey string
	var data map[string]interface{}
	switch n := notification.(type) {
	case gp.GroupNotification:
		var group gp.Group
		group, err = api.getNetwork(n.Group)
		if err != nil {
			log.Println(err)
			return
		}
		data = map[string]interface{}{"type": "GROUP", "adder": n.By.Name, "group-id": n.Group, "group-name": group.Name, "for": recipient}
		CollapseKey = "You've been added to a group"
	case gp.Notification:
		switch {
		case n.Type == "added_you":
			data = map[string]interface{}{"type": "added_you", "adder": n.By.Name, "adder-id": n.By.ID, "for": recipient}
			CollapseKey = "Someone added you to their contacts."
		case n.Type == "accepted_you":
			data = map[string]interface{}{"type": "accepted_you", "accepter": n.By.Name, "accepter-id": n.By.ID, "for": recipient}
			CollapseKey = "Someone accepted your contact request."
		default:
			unknown = true
		}
	case gp.PostNotification:
		switch {
		case n.Type == "liked" && newPush:
			data = map[string]interface{}{"type": "liked", "liker": n.By.Name, "liker-id": n.By.ID, "for": recipient, "post-id": n.Post}
			CollapseKey = "Someone liked your post."
		case n.Type == "commented" && newPush:
			data = map[string]interface{}{"type": "commented", "commenter": n.By.Name, "commenter-id": n.By.ID, "for": recipient, "post-id": n.Post}
			CollapseKey = "Someone commented on your post."
		default:
			unknown = true
		}
	}
	if unknown {
		var count int
		count, err = api.badgeCount(recipient)
		if err != nil {
			log.Println(err)
			return
		}
		data = map[string]interface{}{"count": count, "for": recipient}
		CollapseKey = "New Notification"
	}
	msg = gcm.NewMessage(data, device)
	msg.CollapseKey = CollapseKey
	msg.TimeToLive = 0
	log.Println(msg)
	return
}

//Push takes a gleepost notification and sends it as a push notification to all of recipient's devices.
func (api *API) Push(notification interface{}, recipient gp.UserID) {
	devices, err := api.GetDevices(recipient)
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
			err = api.push.IOSPush(pn)
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
			err = api.push.AndroidPush(pn)
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
