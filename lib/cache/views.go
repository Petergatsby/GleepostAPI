package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func (c *Cache) PublishViewCounts(counts ...gp.PostViewCount) {
	conn := c.pool.Get()
	defer conn.Close()
	log.Println(counts)
	for _, cnt := range counts {
		viewChan := PostViewChannel(cnt.Post)
		event := gp.Event{Type: "views", Location: "/posts/" + strconv.Itoa(int(cnt.Post))}
		event.Data = cnt
		JSONview, _ := json.Marshal(event)
		conn.Send("PUBLISH", viewChan, JSONview)
	}
}

//PostViewChannel returns the namme of the channel for this post's events
func PostViewChannel(post gp.PostID) string {
	return fmt.Sprintf("posts.%d.views", post)
}
