package cache

import "github.com/draaglom/GleepostAPI/lib/gp"

//MessageChan returns a channel containing events for userID (Contents are slices of byte containing JSON)
func (c *Cache) MessageChan(userID gp.UserID) (messages chan []byte) {
	messages = make(chan []byte)
	go c.Subscribe(messages, userID)
	return
}
