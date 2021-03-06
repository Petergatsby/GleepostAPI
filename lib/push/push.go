package push

import (
	"log"

	"github.com/Petergatsby/GleepostAPI/lib/conf"
	"github.com/draaglom/apns"
)

//Pusher is able to push notifications to iOS and android devices.
type Pusher interface {
	CheckFeedbackService(Feedbacker)
	IOSPush(*apns.PushNotification) error
}

type realPusher struct {
	APNSconfig conf.APNSConfig
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

func (f *fakePusher) IOSPush(*apns.PushNotification) error {
	return nil
}

//New constructs a Pusher from a Config
func New(conf conf.PusherConfig) (pusher Pusher) {
	log.Println("Building pusher")
	p := new(realPusher)
	p.APNSconfig = conf.APNS
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
}
