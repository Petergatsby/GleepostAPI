package cache

import (
	"fmt"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/garyburd/redigo/redis"
)

//SetUser - cache a copy of this user.
func (c *Cache) SetUser(user gp.User) {
	conn := c.pool.Get()
	defer conn.Close()
	BaseKey := fmt.Sprintf("users:%d", user.ID)
	conn.Send("MSET", BaseKey+":name", user.Name, BaseKey+":profile_image", user.Avatar)
	conn.Flush()
}

//GetUser - retrieve a cached User, or a redis.Error if they're not in the cache.
func (c *Cache) GetUser(id gp.UserID) (user gp.User, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("users:%d", id)
	values, err := redis.Values(conn.Do("MGET", baseKey+":name", baseKey+":profile_image"))
	if err != nil {
		return user, err
	}
	if len(values) < 2 {
		return user, redis.Error("That user isn't cached!")
	}
	if _, err := redis.Scan(values, &user.Name, &user.Avatar); err != nil {
		return user, err
	}
	if user.Name == "" {
		return user, redis.Error("That user isn't cached!")
	}
	user.ID = id
	return user, nil
}

//SetProfileImage records your avatar in the cache.
func (c *Cache) SetProfileImage(id gp.UserID, url string) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:profile_image", id)
	conn.Send("SET", key, url)
	conn.Flush()
}

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
