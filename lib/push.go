package lib

import (
	"errors"
	"log"
	"regexp"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/apns"
	"github.com/draaglom/gcm"
)

func (api *API) messagePush(message gp.Message, convID gp.ConversationID) {
	devices, err := api.pushableDevices(convID)
	if err != nil {
		log.Println(err)
		return
	}
	for _, device := range devices {
		presence, err := api.Presences.getPresence(device.User)
		if err != nil && err != noPresence {
			log.Println("Error getting user presence:", err)
		}
		if presence.Form == "desktop" && presence.At.Add(30*time.Second).After(time.Now()) {
			log.Println("Not pushing to this user (they're active on the desktop in the last 30s)")
			continue
		}
		if device.User != message.By.ID {
			switch {
			case device.Type == "ios":
				err = api.iosPushMessage(device.ID, message, convID, device.User)
				if err != nil {
					log.Println("Error sending iOS push message", err)
				}
			case device.Type == "android":
				err = api.androidPushMessage(device.ID, message, convID, device.User)
				if err != nil {
					log.Println("Error sending android push message", err)
				}
			}
		}
	}
}

func (api *API) pushableDevices(convID gp.ConversationID) (devices []gp.Device, err error) {
	defer api.Statsd.Time(time.Now(), "gleepost.pushable_devices.byConversationID.db")
	s, err := api.sc.Prepare("SELECT participant_id, device_type, device_id FROM conversation_participants JOIN users ON conversation_participants.participant_id = users.id JOIN devices ON participant_id = devices.user_id WHERE conversation_id = ? AND deleted = 0 AND muted = 0 AND application = 'gleepost'")
	if err != nil {
		return
	}
	rows, err := s.Query(convID)
	if err != nil {
		log.Println("Error getting participant device:", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		device := gp.Device{}
		if err = rows.Scan(&device.User, &device.Type, &device.ID); err != nil {
			log.Println(err)
			return
		}
		devices = append(devices, device)
	}
	return
}

var normRegex = regexp.MustCompile(`<@[a-zA-Z0-9\:]+\|(@\w+)>`)

func normalizeMessage(message string) (textified string) {
	return normRegex.ReplaceAllString(message, "$1")
}

func (api *API) iosPushMention(device string, message gp.Message, convID gp.ConversationID, user gp.UserID) (err error) {
	payload := apns.NewPayload()
	d := apns.NewAlertDictionary()
	d.LocKey = "mention"
	d.LocArgs = []string{message.By.Name}
	if len(message.Text) > 64 {
		//infer
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
