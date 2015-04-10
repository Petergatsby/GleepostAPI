package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//PublishViewCounts publishes the updated view counts given to it in the channel posts.:id.views, to be consumed by a websocket subscriber.
//It doesn't perform rate limiting, deduplication or sanity checking of any kind; this is the caller's responsibility.
func (c *Cache) PublishViewCounts(counts ...gp.PostViewCount) {
	conn := c.pool.Get()
	defer conn.Close()
	log.Println(counts)
	for _, cnt := range counts {
		viewChan := PostChannel(cnt.Post)
		event := gp.Event{Type: "views", Location: "/posts/" + strconv.Itoa(int(cnt.Post))}
		event.Data = cnt
		JSONview, _ := json.Marshal(event)
		conn.Send("PUBLISH", viewChan, JSONview)
	}
}

//PostChannel returns the namme of the channel for this post's events
func PostChannel(post gp.PostID) string {
	return fmt.Sprintf("posts.%d.views", post)
}
