package cache

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

func (c *Cache) PublishViewCounts(counts ...gp.PostViewCount) {
	conn := c.pool.Get()
	defer conn.Close()
	log.Println(counts)
	for _, cnt := range counts {
		JSONview, _ := json.Marshal(cnt)
		conn.Send("PUBLISH", postViewChannel(cnt.Post), JSONview)
	}
}

func postViewChannel(post gp.PostID) string {
	return fmt.Sprintf("posts.%d.views", post)
}
