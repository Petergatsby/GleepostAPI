package cache

import (
	"fmt"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/garyburd/redigo/redis"
)

//GetCommentCount returns the total number of comments on this post.
func (c *Cache) GetCommentCount(id gp.PostID) (count int, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("posts:%d:comments", id)
	count, err = redis.Int(conn.Do("ZCARD", key))
	if err != nil {
		return 0, err
	}
	return count, nil
}

//AddComment places this comment in the cache.
func (c *Cache) AddComment(id gp.PostID, comment gp.Comment) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("posts:%d:comments", id)
	baseKey := fmt.Sprintf("comments:%d", comment.ID)
	conn.Send("ZADD", key, comment.Time.Unix(), comment.ID)
	conn.Send("MSET", baseKey+":by", comment.By.ID, baseKey+":text", comment.Text, baseKey+":time", comment.Time.Format(time.RFC3339))
	conn.Flush()
}

//GetComments returns the comments on this post, ordered from oldest to newest, starting from start.
func (c *Cache) GetComments(postID gp.PostID, start int64, count int) (comments []gp.Comment, err error) {
	comments = make([]gp.Comment, 0)
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("posts:%d:comments", postID)
	values, err := redis.Values(conn.Do("ZREVRANGE", key, start, start+int64(count)-1))
	if err != nil {
		return
	}
	if len(values) == 0 {
		return comments, redis.Error("No conversations for this user in redis.")
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
		comment, e := c.GetComment(gp.CommentID(curr))
		if e != nil {
			return comments, e
		}
		comments = append(comments, comment)
	}
	return
}

//GetComment - a particular comment in the cache.
func (c *Cache) GetComment(commentID gp.CommentID) (comment gp.Comment, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	key := fmt.Sprintf("comments:%d", commentID)
	reply, err := redis.Values(conn.Do("MGET", key+":by", key+":text", key+":time"))
	if err != nil {
		return
	}
	var timeString string
	var by gp.UserID
	if _, err = redis.Scan(reply, &by, &comment.Text, &timeString); err != nil {
		return
	}
	comment.ID = commentID
	comment.By, err = c.GetUser(by)
	if err != nil {
		return
	}
	comment.Time, _ = time.Parse(time.RFC3339, timeString)
	return
}
