package main

import (
	"github.com/anachronistic/apns"
	"github.com/draaglom/GleepostAPI/gp"
	"log"
)

func notify(user gp.UserId) {
	conf := gp.GetConfig()
	client := apns.NewClient("gateway.sandbox.push.apple.com:2195", conf.APNS.CertFile, conf.APNS.KeyFile)
	payload := apns.NewPayload()
	payload.Alert = "Sup"
	payload.Badge = 1337
	payload.Sound = "default"

	devices, err := getDevices(user)
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

func notificationPush(user gp.UserId) {
	conf := gp.GetConfig()
	client := apns.NewClient("gateway.sandbox.push.apple.com:2195", conf.APNS.CertFile, conf.APNS.KeyFile)
	payload := apns.NewPayload()

	devices, err := getDevices(user)
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

func messagePush(message gp.Message, convId gp.ConversationId) {
	conf := gp.GetConfig()
	client := apns.NewClient("gateway.sandbox.push.apple.com:2195", conf.APNS.CertFile, conf.APNS.KeyFile)
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
	recipients := getParticipants(convId)
	for _, user := range recipients {
		if user.Id != message.By.Id {
			devices, err := getDevices(user.Id)
			if err != nil {
				log.Println(err)
			}
			for _, device := range devices {
				pn := apns.NewPushNotification()
				pn.DeviceToken = device.Id
				pn.AddPayload(payload)
				pn.Set("conv", convId)
				resp := client.Send(pn)
				if resp.Error != nil {
					log.Println("Error:", resp.Error)
				}
			}
		}
	}
}
