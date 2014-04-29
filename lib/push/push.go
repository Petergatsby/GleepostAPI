package push

import (
	"encoding/json"
	"github.com/anachronistic/apns"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/gcm"
	"log"
)

type Pusher struct {
	APNSconfig gp.APNSConfig
	GCMconfig  gp.GCMConfig
}

func New(conf gp.Config) (pusher *Pusher) {
	pusher = new(Pusher)
	pusher.APNSconfig = conf.APNS
	pusher.GCMconfig = conf.GCM
	return
}

func (pusher *Pusher) AndroidPush(msg *gcm.Message) (err error) {
	m, _ := json.Marshal(msg)
	log.Printf("%s\n", m)
	sender := &gcm.Sender{ApiKey: pusher.GCMconfig.APIKey}
	response, err := sender.SendNoRetry(msg)
	log.Println(response)
	return
}

func (pusher *Pusher) IOSPush(pn *apns.PushNotification) (err error) {
	url := "gateway.sandbox.push.apple.com:2195"
	if pusher.APNSconfig.Production {
		url = "gateway.push.apple.com:2195"
	}
	client := apns.NewClient(url, pusher.APNSconfig.CertFile, pusher.APNSconfig.KeyFile)
	resp := client.Send(pn)
	if !resp.Success {
		log.Println("Failed to send push notification to:", pn.DeviceToken)
	}
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

type Pushable interface {
	IOSNotification() *apns.PushNotification
	AndroidNotification() *gcm.Message
}
