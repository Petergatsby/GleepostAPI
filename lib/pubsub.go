package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/cache"
)

func EventSubscribe(subscriptions []string) (events gp.MsgQueue) {
	commands := make(chan gp.QueueCommand)
	messages := make(chan []byte)
	events = gp.MsgQueue{Commands: commands, Messages: messages}
	return cache.EventSubscribe(subscriptions)
}
