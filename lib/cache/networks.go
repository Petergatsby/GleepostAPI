package cache

import (
	"fmt"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/garyburd/redigo/redis"
)

/********************************************************************
		Networks
********************************************************************/

//GetUserNetworks returns all the networks userID is a member of.
func (c *Cache) GetUserNetworks(userID gp.UserID) (networks []gp.Group, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:networks", userID)
	values, err := redis.Values(conn.Do("SMEMBERS", key))
	if err != nil {
		return networks, err
	}
	if len(values) == 0 {
		return networks, ErrEmptyCache
	}
	for len(values) > 0 {
		net := -1
		values, err = redis.Scan(values, &net)
		switch {
		case err != nil || net <= 0:
			return
		default:
			network, e := c.GetNetwork(gp.NetworkID(net))
			if e != nil {
				return networks, e
			}
			networks = append(networks, network)
		}
	}
	return networks, nil
}

//SetUserNetworks takes a list of networks and records in the cache that this user belongs to them.
func (c *Cache) SetUserNetworks(userID gp.UserID, networks ...gp.Group) {
	conn := c.pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("users:%d:networks", userID)
	for _, n := range networks {
		conn.Send("SADD", baseKey+":id")
		go c.SetNetwork(n)
	}
	conn.Flush()
}

//SetNetwork adds network to the cache.
func (c *Cache) SetNetwork(network gp.Group) {
	conn := c.pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("networks:%d", network.ID)
	conn.Send("MSET", baseKey+":id", network.ID, baseKey+":name", network.Name, baseKey+":image", network.Image, baseKey+":desc", network.Desc)
	if network.Creator != nil {
		conn.Send("SET", baseKey+":creator", network.Creator.ID)
	}
	conn.Flush()
}

//GetNetwork returns the network with id netId from the cache, or err if it isn't there.
func (c *Cache) GetNetwork(netID gp.NetworkID) (network gp.Group, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("networks:%d", netID)
	reply, err := redis.Values(conn.Do("MGET", key+":id", key+":name", key+":image", key+":desc", key+":creator"))
	if err != nil {
		return
	}
	var u gp.UserID
	if _, err = redis.Scan(reply, &network.ID, &network.Name, &network.Image, &network.Desc, &u); err != nil {
		return
	}
	if network.ID == 0 {
		err = redis.Error("Cache miss")
	}
	if u != 0 {
		user, err := c.GetUser(u)
		if err == nil {
			network.Creator = &user
		}
	}
	return
}
