package cache

import (
	"fmt"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/garyburd/redigo/redis"
)

//SetBusyStatus records if this user is busy or not.
func (c *Cache) SetBusyStatus(id gp.UserID, busy bool) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:busy", id)
	conn.Send("SET", key, busy)
	conn.Flush()
}

//UserPing marks this user as busy for the next timeout seconds.
func (c *Cache) UserPing(id gp.UserID, timeout int) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:busy", id)
	conn.Send("SETEX", key, timeout, 1)
	conn.Flush()
}

//UserIsOnline returns true if this user is online.
//Should this use users:userID:busy??
func (c *Cache) UserIsOnline(id gp.UserID) (online bool) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:busy", id)
	online, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false
	}
	return
}
