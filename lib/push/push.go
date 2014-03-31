package push

import (
	"encoding/json"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/gcm"
	"log"
)

type Pusher struct {
	APNSconfig gp.APNSConfig
	GCMconfig gp.GCMConfig
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

