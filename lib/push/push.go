package push

import (
	"encoding/json"
	"log"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/apns"
	"github.com/draaglom/gcm"
)

//Pusher is able to push notifications to iOS and android devices.
type Pusher struct {
	APNSconfig conf.APNSConfig
	GCMconfig  conf.GCMConfig
	Connection *apns.Connection
}

//New constructs a Pusher from a Config
func New(conf conf.Config) (pusher *Pusher) {
	pusher = new(Pusher)
	pusher.APNSconfig = conf.APNS
	pusher.GCMconfig = conf.GCM
	url := "gateway.sandbox.push.apple.com:2195"
	if pusher.APNSconfig.Production {
		url = "gateway.push.apple.com:2195"
	}
	client := apns.NewClient(url, pusher.APNSconfig.CertFile, pusher.APNSconfig.KeyFile)
	conn := apns.NewConnection(client)
	pusher.Connection = conn
	errs := pusher.Connection.Errors()
	go func(c <-chan apns.BadPushNotification) {
		for {
			n := <-c
			log.Println(n)
		}
	}(errs)
	err := conn.Start()
	if err != nil {
		log.Println(err)
	}
	return
}

//AndroidPush sends a gcm.Message to its recipient.
func (pusher *Pusher) AndroidPush(msg *gcm.Message) (err error) {
	m, _ := json.Marshal(msg)
	log.Printf("%s\n", m)
	sender := &gcm.Sender{ApiKey: pusher.GCMconfig.APIKey}
	response, err := sender.SendNoRetry(msg)
	log.Println(response)
	return
}

//IOSPush sends an apns notification to its recipient.
func (pusher *Pusher) IOSPush(pn *apns.PushNotification) (err error) {
	pusher.Connection.Enqueue(pn)
	return nil
}

//Pushable represents something which can turn itself into a push notification.
type Pushable interface {
	IOSNotification() *apns.PushNotification
	AndroidNotification() *gcm.Message
}
