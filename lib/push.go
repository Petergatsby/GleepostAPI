package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/gcm"
	"github.com/draaglom/apns"
	"log"
	"time"
)

func (api *API) notify(user gp.UserId) {
	url := "gateway.sandbox.push.apple.com:2195"
	if api.Config.APNS.Production {
		url = "gateway.push.apple.com:2195"
	}
	client := apns.NewClient(url, api.Config.APNS.CertFile, api.Config.APNS.KeyFile)
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
			pn.DeviceToken = device.Id
			pn.AddPayload(payload)
			resp := client.Send(pn)
			log.Println("Success:", resp.Success)
			log.Println("Error:", resp.Error)
		}
	}
}

func (api *API) notificationPush(user gp.UserId) {
	notifications, err := api.GetUserNotifications(user)
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
			err = api.iosBadge(device.Id, badge)
			if err != nil {
				log.Println("Error sending push notification:", err)
			} else {
				count += 1
			}
		case device.Type == "android":
			err = api.androidNotification(device.Id, badge, user)
			if err != nil {
				log.Println("Error sending push notification:", err)
			} else {
				count += 1
			}
		}
	}
	log.Printf("Badged %d's %d devices\n", user, count)
}

func (api *API) groupPush(adder gp.User, addees []gp.UserId, group gp.Network) (err error) {
	log.Println("Notifying ", len(addees), " users that they've been added to a group")
	for _, u := range addees {
		devices, e := api.GetDevices(u)
		if e != nil {
			log.Println(e)
			return
		}
		count := 0
		for _, device := range devices {
			switch {
			case device.Type == "ios":
				err = api.iOSGroupNotification(device.Id, group, adder.Name, u)
				if err != nil {
					log.Println("Error sending group push notification:", err)
				} else {
					count += 1
				}
			case device.Type == "android":
				err = api.androidPushGroup(device.Id, group, adder.Name, u)
				if err != nil {
					log.Println("Error sending group push notification:", err)
				} else {
					count += 1
				}
			}
		}
		log.Printf("Group notified %d's %d devices\n", u, count)
	}
	return
}

//addedPush sends a push notification to addee to notify them they've been added by adder.
func (api *API) addedPush(adder gp.User, addee gp.UserId) (err error) {
	log.Printf("Notifiying user %d that they've been added by %s (%d)\n", addee, adder.Name, adder.Id)
	devices, e := api.GetDevices(addee)
	if e != nil {
		log.Println(e)
		return
	}
	count := 0
	for _, device := range devices {
		switch {
		case device.Type == "ios":
			err = api.iOSAddedNotification(device.Id, adder, addee)
			if err != nil {
				log.Println("Error sending added push notification:", err)
			} else {
				count += 1
			}
		case device.Type == "android":
			err = api.androidAddedNotification(device.Id, adder, addee)
			if err != nil {
				log.Println("Error sending added push notification:", err)
			} else {
				count += 1
			}
		}
	}
	log.Printf("Notified %d's %d devices\n", addee, count)
	return
}

func (api *API) iOSAddedNotification(device string, adder gp.User, addee gp.UserId) (err error) {
	url := "gateway.sandbox.push.apple.com:2195"
	if api.Config.APNS.Production {
		url = "gateway.push.apple.com:2195"
	}
	client := apns.NewClient(url, api.Config.APNS.CertFile, api.Config.APNS.KeyFile)
	payload := apns.NewPayload()
	d := apns.NewAlertDictionary()
	d.LocKey = "added_you"
	d.LocArgs = []string{adder.Name}
	payload.Alert = d
	payload.Sound = "default"
	notifications, err := api.GetUserNotifications(addee)
	if err != nil {
		log.Println(err)
		return
	}
	payload.Badge = len(notifications)
	unread, err := api.UnreadMessageCount(addee)
	if err == nil {
		payload.Badge += unread
	}
	log.Printf("Added contact notification: badging %d with %d notifications (%d from unread messages)", addee, payload.Badge, unread)
	pn := apns.NewPushNotification()
	pn.DeviceToken = device
	pn.AddPayload(payload)
	pn.Set("adder-id", adder.Id)
	resp := client.Send(pn)
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

func (api *API) androidAddedNotification(device string, adder gp.User, addee gp.UserId) (err error) {
	data := map[string]interface{}{"type": "added_you", "adder": adder.Name, "adder-id": adder.Id, "for": addee}
	msg := gcm.NewMessage(data, device)
	msg.TimeToLive = 0
	msg.CollapseKey = "Someone added you to their contacts."
	return api.push.AndroidPush(msg)
}

//iosBadge sets this device's badge, or returns an error.
func (api *API) iosBadge(device string, badge int) (err error) {
	url := "gateway.sandbox.push.apple.com:2195"
	if api.Config.APNS.Production {
		url = "gateway.push.apple.com:2195"
	}
	client := apns.NewClient(url, api.Config.APNS.CertFile, api.Config.APNS.KeyFile)
	payload := apns.NewPayload()
	payload.Badge = badge
	pn := apns.NewPushNotification()
	pn.DeviceToken = device
	pn.AddPayload(payload)
	resp := client.Send(pn)
	if !resp.Success {
		log.Println("Failed to send push notification to:", device)
	}
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

//androidNotification sends a "You have new notifications" push to this device.
//user is included because GCM doesn't really like deregistering, so we include the
//recipient id in the notification so the app can filter itself.
func (api *API) androidNotification(device string, count int, user gp.UserId) (err error) {
	data := map[string]interface{}{"count": count, "for": user}
	msg := gcm.NewMessage(data, device)
	msg.CollapseKey = "New Notification"

	sender := &gcm.Sender{ApiKey: api.Config.GCM.APIKey}
	response, err := sender.SendNoRetry(msg)
	log.Println(response)
	return
}

func (api *API) messagePush(message gp.Message, convId gp.ConversationId) {
	log.Println("Trying to send a push notification")
	recipients := api.GetParticipants(convId)
	for _, user := range recipients {
		if user.Id != message.By.Id {
			log.Println("Trying to send a push notification to", user.Name)
			devices, err := api.GetDevices(user.Id)
			if err != nil {
				log.Println(err)
			}
			count := 0
			for _, device := range devices {
				log.Println("Sending push notification to device: ", device)
				switch {
				case device.Type == "ios":
					err = api.iosPushMessage(device.Id, message, convId, user.Id)
					if err != nil {
						log.Println(err)
					} else {
						count++
					}
				case device.Type == "android":
					err = api.androidPushMessage(device.Id, message, convId, user.Id)
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

func (api *API) iosPushMessage(device string, message gp.Message, convId gp.ConversationId, user gp.UserId) (err error) {
	url := "gateway.sandbox.push.apple.com:2195"
	if api.Config.APNS.Production {
		url = "gateway.push.apple.com:2195"
	}
	client := apns.NewClient(url, api.Config.APNS.CertFile, api.Config.APNS.KeyFile)
	payload := apns.NewPayload()
	d := apns.NewAlertDictionary()
	d.LocKey = "MSG"
	d.LocArgs = []string{message.By.Name}
	if len(message.Text) > 64 {
		d.LocArgs = append(d.LocArgs, message.Text[:64] + "...")
	} else {
		d.LocArgs = append(d.LocArgs, message.Text)
	}
	payload.Alert = d
	payload.Sound = "default"
	notifications, err := api.GetUserNotifications(user)
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
	pn.Set("conv", convId)
	resp := client.Send(pn)
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

func (api *API) androidPushMessage(device string, message gp.Message, convId gp.ConversationId, user gp.UserId) (err error) {
	data := map[string]interface{}{"type": "MSG", "sender": message.By.Name, "sender-id": message.By.Id, "conv": convId, "for": user}
	if len(message.Text) > 3200 {
		data["text"] = message.Text[:3200] + "..."
	} else {
		data["text"] = message.Text
	}
	msg := gcm.NewMessage(data, device)
	msg.TimeToLive = 0
	return api.push.AndroidPush(msg)
}

func (api *API) androidPushGroup(device string, group gp.Network, adder string, user gp.UserId) (err error) {
	data := map[string]interface{}{"type": "GROUP", "adder": adder, "group-id": group.Id, "group-name": group.Name, "for": user}
	msg := gcm.NewMessage(data, device)
	msg.TimeToLive = 0
	msg.CollapseKey = "You've been added to a group"
	return api.push.AndroidPush(msg)
}


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

func (api *API) iOSGroupNotification(device string, group gp.Network, adder string, addee gp.UserId) (err error) {
	url := "gateway.sandbox.push.apple.com:2195"
	if api.Config.APNS.Production {
		url = "gateway.push.apple.com:2195"
	}
	client := apns.NewClient(url, api.Config.APNS.CertFile, api.Config.APNS.KeyFile)
	payload := apns.NewPayload()
	d := apns.NewAlertDictionary()
	d.LocKey = "GROUP"
	d.LocArgs = []string{adder, group.Name}
	payload.Alert = d
	payload.Sound = "default"
	notifications, err := api.GetUserNotifications(addee)
	if err != nil {
		log.Println(err)
		return
	}
	payload.Badge = len(notifications)
	unread, err := api.UnreadMessageCount(addee)
	if err == nil {
		payload.Badge += unread
	}
	log.Printf("Group notification: badging %d with %d notifications (%d from unread messages)", addee, payload.Badge, unread)
	pn := apns.NewPushNotification()
	pn.DeviceToken = device
	pn.AddPayload(payload)
	pn.Set("group-id", group.Id)
	resp := client.Send(pn)
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

func (api *API) acceptedPush(accepter gp.User, acceptee gp.UserId) (err error) {
	log.Printf("Notifiying user %d that they've been accepted by %s (%d)\n", acceptee, accepter.Name, accepter.Id)
	devices, e := api.GetDevices(acceptee)
	if e != nil {
		log.Println(e)
		return
	}
	count := 0
	for _, device := range devices {
		switch {
		case device.Type == "ios":
			err = api.iOSAcceptedNotification(device.Id, accepter, acceptee)
			if err != nil {
				log.Println("Error sending accepted push notification:", err)
			} else {
				count += 1
			}
		case device.Type == "android":
			err = api.androidAcceptedNotification(device.Id, accepter, acceptee)
			if err != nil {
				log.Println("Error sending accepted push notification:", err)
			} else {
				count += 1
			}
		}
	}
	log.Printf("Notified %d's %d devices\n", acceptee, count)
	return
}

func (api *API) iOSAcceptedNotification(device string, accepter gp.User, acceptee gp.UserId) (err error) {
	url := "gateway.sandbox.push.apple.com:2195"
	if api.Config.APNS.Production {
		url = "gateway.push.apple.com:2195"
	}
	client := apns.NewClient(url, api.Config.APNS.CertFile, api.Config.APNS.KeyFile)
	payload := apns.NewPayload()
	d := apns.NewAlertDictionary()
	d.LocKey = "accepted_you"
	d.LocArgs = []string{accepter.Name}
	payload.Alert = d
	payload.Sound = "default"
	notifications, err := api.GetUserNotifications(acceptee)
	if err != nil {
		log.Println(err)
		return
	}
	payload.Badge = len(notifications)
	unread, err := api.UnreadMessageCount(acceptee)
	if err == nil {
		payload.Badge += unread
	}
	log.Printf("Accepted contact notification: badging %d with %d notifications (%d from unread messages)", acceptee, payload.Badge, unread)
	pn := apns.NewPushNotification()
	pn.DeviceToken = device
	pn.AddPayload(payload)
	pn.Set("accepter-id", accepter.Id)
	resp := client.Send(pn)
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

func (api *API) androidAcceptedNotification(device string, accepter gp.User, acceptee gp.UserId) (err error) {
	data := map[string]interface{}{"type": "accepted_you", "accepter": accepter.Name, "accepter-id": accepter.Id, "for": acceptee}
	msg := gcm.NewMessage(data, device)
	msg.TimeToLive = 0
	msg.CollapseKey = "Someone accepted your contact request."
	return api.push.AndroidPush(msg)
}

func (api *API) iOSNewConversationNotification(device string, conv gp.ConversationId, user gp.UserId, with gp.User) (err error) {
	url := "gateway.sandbox.push.apple.com:2195"
	if api.Config.APNS.Production {
		url = "gateway.push.apple.com:2195"
	}
	client := apns.NewClient(url, api.Config.APNS.CertFile, api.Config.APNS.KeyFile)
	payload := apns.NewPayload()
	d := apns.NewAlertDictionary()
	d.LocKey = "NEW_CONV"
	d.LocArgs = []string{with.Name}
	payload.Alert = d
	payload.Sound = "default"
	notifications, err := api.GetUserNotifications(user)
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
	resp := client.Send(pn)
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

func (api *API) androidNewConversationNotification(device string, conv gp.ConversationId, user gp.UserId, with gp.User) (err error) {
	data := map[string]interface{}{"type": "NEW_CONV", "with": with.Name, "with-id": with.Id, "conv": conv, "for": user}
	msg := gcm.NewMessage(data, device)
	msg.TimeToLive = 0
	msg.CollapseKey = "You have a new conversation!"
	return api.push.AndroidPush(msg)
}

func (api *API) newConversationPush(initiator gp.User, other gp.UserId, conv gp.ConversationId) (err error) {
	log.Printf("Notifiying user %d that they've got a new conversation with %s (%d)\n", other, initiator.Name, initiator.Id)
	devices, e := api.GetDevices(other)
	if e != nil {
		log.Println(e)
		return
	}
	count := 0
	for _, device := range devices {
		switch {
		case device.Type == "ios":
			err = api.iOSNewConversationNotification(device.Id, conv, other, initiator)
			if err != nil {
				log.Println("Error sending new conversation push notification:", err)
			} else {
				count += 1
			}
		case device.Type == "android":
			err = api.androidNewConversationNotification(device.Id, conv, other, initiator)
			if err != nil {
				log.Println("Error sending new conversation push notification:", err)
			} else {
				count += 1
			}
		}
	}
	log.Printf("Notified %d's %d devices\n", other, count)
	return

}
