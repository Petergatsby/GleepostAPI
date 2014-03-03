package cache

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/garyburd/redigo/redis"
	"fmt"
)

/********************************************************************
		Networks
********************************************************************/

func (c *Cache) GetUserNetworks(userId gp.UserId) (networks []gp.Network, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("users:%d:networks", userId)
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
			network, e := c.GetNetwork(gp.NetworkId(net))
			if e != nil {
				return networks, e
			}
			networks = append(networks, network)
		}
	}
	return networks, nil
}

//SetUserNetworks 
func (c *Cache) SetUserNetworks(userId gp.UserId, networks ...gp.Network) {
	conn := c.pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("users:%d:networks", userId)
	for _, n := range networks {
		conn.Send("SADD", baseKey+":id")
		go c.SetNetwork(n)
	}
	conn.Flush()
}

//SetNetwork adds network to the cache.
func (c *Cache) SetNetwork(network gp.Network) {
	conn := c.pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("networks:%d", network.Id)
	conn.Send("MSET", baseKey+":id", network.Id, baseKey+":name", network.Name)
	conn.Flush()
}

//GetNetwork returns the network with id netId from the cache, or err if it isn't there.
func (c *Cache) GetNetwork(netId gp.NetworkId) (network gp.Network, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("networks:%d", netId)
	reply, err := redis.Values(conn.Do("MGET", key+":id", key+":name"))
	if err != nil {
		return
	}
	if _, err = redis.Scan(reply, &network.Id, &network.Name); err != nil {
		return
	}
	if network.Id == 0 {
		err = redis.Error("Cache miss")
	}
	return
}
