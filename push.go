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
	notifications, err := getUserNotifications(user)
	if err != nil {
		log.Println(err)
		return
	}
	notificationCount := len(notifications)

	client := apns.NewClient("gateway.sandbox.push.apple.com:2195", conf.APNS.CertFile, conf.APNS.KeyFile)
	payload := apns.NewPayload()
	payload.Badge = notificationCount

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
