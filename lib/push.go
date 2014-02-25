package lib

import (
	"github.com/draaglom/apns"
	"github.com/alexjlockwood/gcm"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"log"
	"time"
)

type Pusher struct {
	config gp.APNSConfig
}

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
	log.Printf("Badged %d's %d devices\n", count)
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

	sender := gcm.Sender{ApiKey: api.Config.GCM.APIKey}
	response, err := sender.Send(msg, 2)
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
		d.Body = message.Text[:64] + "..."
	} else {
		d.Body = message.Text
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
