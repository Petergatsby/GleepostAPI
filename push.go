package main

import (
	"github.com/anachronistic/apns"
	"log"
	"github.com/draaglom/GleepostAPI/gp"
)

func notify(user gp.UserId) {
	conf := gp.GetConfig()
	client := apns.NewClient("gateway.sandbox.push.apple.com:2195", conf.APNS.CertFile, conf.APNS.KeyFile)
	payload := apns.NewPayload()
	payload.Alert = "Sup"
	payload.Badge = 1337
	payload.Sound = "sup.mp3"

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
