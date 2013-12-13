package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/cache"
)

func EventSubscribe(subscriptions []string) (events gp.MsgQueue) {
	events = cache.EventSubscribe(subscriptions)
	return
}
