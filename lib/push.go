package lib

import (
	"github.com/anachronistic/apns"
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
	url := "gateway.sandbox.push.apple.com:2195"
	if api.Config.APNS.Production {
		url = "gateway.push.apple.com:2195"
	}
	client := apns.NewClient(url, api.Config.APNS.CertFile, api.Config.APNS.KeyFile)
	payload := apns.NewPayload()

	notifications, err := api.GetUserNotifications(user)
	if err != nil {
		log.Println(err)
		return
	}
	payload.Badge = len(notifications)
	log.Printf("Badging %d with %d notifications", user, payload.Badge)

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

func (api *API) messagePush(message gp.Message, convId gp.ConversationId) {
	log.Println("Trying to send a push notification")
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
	recipients := api.GetParticipants(convId)
	for _, user := range recipients {
		if user.Id != message.By.Id {
			devices, err := api.GetDevices(user.Id)
			if err != nil {
				log.Println(err)
			}
			for _, device := range devices {
				if device.Type == "ios" {
					log.Println("Sending push notification to device: ", device)
					pn := apns.NewPushNotification()
					pn.DeviceToken = device.Id
					pn.AddPayload(payload)
					pn.Set("conv", convId)
					resp := client.Send(pn)
					log.Println("Sent a message notification, the response was:", resp)
					if resp.Error != nil {
						log.Println("Error:", resp.Error)
					}
				}
			}
		}
	}
}

func (api *API) CheckFeedbackService() {
	url := "gateway.sandbox.push.apple.com:2195"
	if api.Config.APNS.Production {
		url = "gateway.push.apple.com:2195"
	}
	client := apns.NewClient(url, api.Config.APNS.CertFile, api.Config.APNS.KeyFile)
	log.Println("Connected to feedback service")
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
