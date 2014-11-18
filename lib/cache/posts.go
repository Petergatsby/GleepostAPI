package cache

import (
	"fmt"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/db"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/garyburd/redigo/redis"
)

//AddPosts adds all the posts to this network.
func (c *Cache) addPosts(net gp.NetworkID, posts []gp.Post) (err error) {
	for _, post := range posts {
		go c.AddPost(post)
		err = c.AddPostToNetwork(post, net)
		if err != nil {
			return
		}
	}
	return
}

//AddPost adds a post into the cache but doesn't record its membership in a network.
func (c *Cache) AddPost(post gp.Post) {
	conn := c.pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("posts:%d", post.ID)
	conn.Send("MSET", baseKey+":by", post.By.ID, baseKey+":time", post.Time.Format(time.RFC3339), baseKey+":text", post.Text)
	conn.Flush()
}

//AddPostToNetwork records that this post is in network.
func (c *Cache) AddPostToNetwork(post gp.Post, network gp.NetworkID) (err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("networks:%d:posts", network)
	exists, _ := redis.Bool(conn.Do("EXISTS", key))
	if !exists { //Without this we might get stuck with only recent posts in cache
		return ErrEmptyCache
	}
	conn.Send("ZADD", key, post.Time.Unix(), post.ID)
	conn.Flush()
	return nil
}

//GetPost fetches the core details of a post from the cache, or returns an error if it's not in the cache (maybe)
func (c *Cache) GetPost(postID gp.PostID) (post gp.PostCore, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	baseKey := fmt.Sprintf("posts:%d", postID)
	values, err := redis.Values(conn.Do("MGET", baseKey+":by", baseKey+":time", baseKey+":text"))
	if err != nil {
		return post, err
	}
	var by gp.UserID
	var t string
	if _, err = redis.Scan(values, &by, &t, &post.Text); err != nil {
		return post, err
	}
	post.ID = postID
	post.By, err = c.GetUser(by)
	if err != nil {
		return post, err
	}
	post.Time, _ = time.Parse(time.RFC3339, t)
	return post, nil
}

//GetPosts returns posts in this network in a manner mirroring db.NewGetPosts.
//TODO: Return posts which don't embed a user
func (c *Cache) getPosts(id gp.NetworkID, mode int, index int64, count int) (posts []gp.PostCore, err error) {
	conn := c.pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("networks:%d:posts", id)
	var start, finish int
	switch {
	case mode == gp.OBEFORE:
		rindex := -1
		rindex, err = redis.Int(conn.Do("ZREVRANK", key, index))
		if err != nil {
			return
		}
		if rindex < 1 {
			return posts, ErrEmptyCache
		}
		start = rindex + 1
		finish = rindex + count
	case mode == gp.OAFTER:
		rindex := -1
		rindex, err = redis.Int(conn.Do("ZREVRANK", key, index))
		if err != nil {
			return
		}
		if rindex < 1 {
			return posts, ErrEmptyCache
		}
		start = rindex - count
		if start < 0 {
			start = 0
		}
		finish = rindex - 1
	default:
		start = int(index)
		finish = int(index) + count - 1
	}
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, finish))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return posts, ErrEmptyCache
	}
	for len(values) > 0 {
		curr := -1
		values, err = redis.Scan(values, &curr)
		if err != nil {
			return
		}
		if curr == -1 {
			return
		}
		postID := gp.PostID(curr)
		post, err := c.GetPost(postID)
		if err != nil {
			return posts, err
		}
		posts = append(posts, post)
	}
	return
}

//AddPostsFromDB refills an empty cache from the database.
func (c *Cache) addPostsFromDB(netID gp.NetworkID, db *db.DB) {
	posts, err := db.GetPosts(netID, 1, 0, c.config.PostCache, "")
	if err != nil {
		log.Println(err)
	}
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("networks:%d:posts", netID)
	for _, post := range posts {
		baseKey := fmt.Sprintf("posts:%d", post.ID)
		conn.Send("MSET", baseKey+":by", post.By.ID, baseKey+":time", post.Time.Format(time.RFC3339), baseKey+":text", post.Text)
		conn.Send("ZADD", key, post.Time.Unix(), post.ID)
		conn.Flush()
	}
}
