package lib

import (
	"errors"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/apns"
	"github.com/draaglom/gcm"
)

func (api *API) messagePush(message gp.Message, convID gp.ConversationID) {
	recipients, err := api.getParticipants(convID, false)
	if err != nil {
		log.Println(err)
		return
	}
	for _, user := range recipients {
		if user.ID != message.By.ID {
			devices, err := getDevices(api.sc, user.ID, "gleepost")
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
			log.Printf("Sent %d notifications successfully to %s's %d devices\n", count, user.Name, len(devices))
		}
	}
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
	payload.Badge, err = api.badgeCount(user)
	pn := apns.NewPushNotification()
	pn.DeviceToken = device
	pn.AddPayload(payload)
	pn.Set("conv", convID)
	if message.Group > 0 {
		pn.Set("group", message.Group)
	}
	pn.Set("profile_image", message.By.Avatar)
	pusher, ok := api.pushers["gleepost"]
	if ok {
		err = pusher.IOSPush(pn)
	}
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
	pusher, ok := api.pushers["gleepost"]
	if ok {
		err = pusher.AndroidPush(msg)
	}
	return
}

//FeedbackDaemon checks the APNS feedback service every frequency seconds.
func (api *API) FeedbackDaemon(frequency int) {
	duration := time.Duration(frequency) * time.Second
	c := time.Tick(duration)
	for {
		<-c
		for _, psh := range api.pushers {
			go psh.CheckFeedbackService(api.DeviceFeedback)
		}
	}
}

//SendUpdateNotification sends an update notification to all devices which, when pressed, prompts the user to update if version > installed version.
func (api *API) SendUpdateNotification(userID gp.UserID, message, version, platform string) (count int, err error) {
	if !api.isAdmin(userID) {
		err = ENOTALLOWED
		return
	}
	return api.massNotification(message, version, platform)
}

//MassNotification sends an update notification to all devices which, when pressed, prompts the user to update if version > installed version.
func (api *API) massNotification(message string, version string, platform string) (count int, err error) {
	devices, err := api.getAllDevices(platform)
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
	payload.Badge, err = api.badgeCount(device.User)
	if err != nil {
		log.Println(err)
	}
	pn := apns.NewPushNotification()
	pn.DeviceToken = device.ID
	pn.AddPayload(payload)
	pn.Set("version", version)
	pusher, ok := api.pushers["gleepost"]
	if ok {
		pusher.IOSPush(pn)
	}
	return
}

func (api *API) badgeCount(user gp.UserID) (count int, err error) {
	count, err = api.userUnreadNotifications(user)
	if err != nil {
		log.Println(err)
		return
	}
	unread, e := api.UnreadMessageCount(user, true)
	if e == nil {
		count += unread
	} else {
		log.Println(e)
	}
	return
}
