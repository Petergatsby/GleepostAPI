package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
)

//EventSubscribe subscribes to all the subscriptions, returning a MsgQueue of their contents.
//I don't know why it's here, it seems a little redundant.
func (api *API) EventSubscribe(subscriptions []string) (events gp.MsgQueue) {
	events = api.broker.EventSubscribe(subscriptions)
	return
}
