package push

import (
	"encoding/json"
	"log"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/apns"
	"github.com/draaglom/gcm"
)

//Pusher is able to push notifications to iOS and android devices.
type Pusher interface {
	CheckFeedbackService(Feedbacker)
	AndroidPush(*gcm.Message) error
	IOSPush(*apns.PushNotification) error
}

type realPusher struct {
	APNSconfig conf.APNSConfig
	GCMconfig  conf.GCMConfig
	Connection *apns.Connection
}

//Feedbacker is a function which processes APNS feedback.
type Feedbacker func(string, uint32) error

//CheckFeedbackService receives any bad device tokens from APNS and processes the result with f.
func (pusher *realPusher) CheckFeedbackService(f Feedbacker) {
	url := "feedback.sandbox.push.apple.com:2196"
	if pusher.APNSconfig.Production {
		url = "feedback.push.apple.com:2196"
	}
	client := apns.NewClient(url, pusher.APNSconfig.CertFile, pusher.APNSconfig.KeyFile)
	go client.ListenForFeedback()
	for {
		select {
		case resp := <-apns.FeedbackChannel:
			log.Println("Bad device:", resp.DeviceToken, resp.Timestamp)
			f(resp.DeviceToken, resp.Timestamp)
		case <-apns.ShutdownChannel:
			return
		}
	}
}

type fakePusher struct{}

func (f *fakePusher) CheckFeedbackService(feed Feedbacker) {
	return
}

func (f *fakePusher) AndroidPush(*gcm.Message) error {
	return nil
}

func (f *fakePusher) IOSPush(*apns.PushNotification) error {
	return nil
}

//New constructs a Pusher from a Config
func New(conf conf.PusherConfig) (pusher Pusher) {
	log.Println("Building pusher")
	p := new(realPusher)
	p.APNSconfig = conf.APNS
	p.GCMconfig = conf.GCM
	url := "gateway.sandbox.push.apple.com:2195"
	if p.APNSconfig.Production {
		url = "gateway.push.apple.com:2195"
	}
	log.Println("Pusher is using url:", url)
	client := apns.NewClient(url, p.APNSconfig.CertFile, p.APNSconfig.KeyFile)
	conn := apns.NewConnection(client)
	p.Connection = conn
	errs := p.Connection.Errors()
	go func(c <-chan apns.BadPushNotification) {
		for {
			n := <-c
			log.Println(n)
		}
	}(errs)
	err := conn.Start()
	if err != nil {
		log.Println("ERROR STARTING APNS CONNECTION:", err)
	}
	pusher = p
	return
}

//NewFake gives a pusher which simply blackholes every notification.
func NewFake() Pusher {
	return &fakePusher{}
}

//AndroidPush sends a gcm.Message to its recipient.
func (pusher *realPusher) AndroidPush(msg *gcm.Message) (err error) {
	m, _ := json.Marshal(msg)
	log.Printf("%s\n", m)
	sender := &gcm.Sender{ApiKey: pusher.GCMconfig.APIKey}
	response, err := sender.SendNoRetry(msg)
	log.Println(response)
	return
}

//IOSPush sends an apns notification to its recipient.
func (pusher *realPusher) IOSPush(pn *apns.PushNotification) (err error) {
	if pusher != nil {
		pusher.Connection.Enqueue(pn)
	}
	return nil
}

//Pushable represents something which can turn itself into a push notification.
type Pushable interface {
	IOSNotification() *apns.PushNotification
	AndroidNotification() *gcm.Message
}
