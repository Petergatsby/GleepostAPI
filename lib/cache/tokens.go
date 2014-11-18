package cache

import (
	"fmt"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/garyburd/redigo/redis"
)

//PutToken records this token in the cache until it expires.
func (c *Cache) PutToken(token gp.Token) {
	/* Set a session token in redis.
		We use the token value as part of the redis key
	        so that a user may have more than one concurrent session
		(eg: signed in on the web and mobile at once */
	conn := c.pool.Get()
	defer conn.Close()
	expiry := int(token.Expiry.Sub(time.Now()).Seconds())
	key := fmt.Sprintf("users:%d:token:%s", token.UserID, token.Token)
	conn.Send("SETEX", key, expiry, token.Expiry)
	conn.Flush()
}

//TokenExists returns true if this id:token pair exists.
func (c *Cache) TokenExists(id gp.UserID, token string) bool {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:token:%s", id, token)
	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false
	}
	return exists
}
