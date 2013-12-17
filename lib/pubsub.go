package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
)

func (api *API)EventSubscribe(subscriptions []string) (events gp.MsgQueue) {
	events = api.cache.EventSubscribe(subscriptions)
	return
}
