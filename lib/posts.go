package lib

import (
	"github.com/draaglom/GleepostAPI/lib/db"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/cache"
	"time"
)

func GetCommentCount(id gp.PostId) (count int) {
	count, err := cache.GetCommentCount(id)
	if err != nil {
		count = db.GetCommentCount(id)
	}
	return count
}

func CreateComment(postId gp.PostId, userId gp.UserId, text string) (commId gp.CommentId, err error) {
	post, err := GetPost(postId)
	if err != nil {
		return
	}
	commId, err = db.CreateComment(postId, userId, text)
	if err == nil {
		user, e := GetUser(userId)
		if e != nil {
			return commId, e
		}
		comment := gp.Comment{Id: commId, Post: postId, By: user, Time: time.Now().UTC(), Text: text}
		go createNotification("commented", userId, post.By.Id, true, postId)
		go cache.AddComment(postId, comment)
	}
	return commId, err
}

func GetPostImages(postId gp.PostId) (images []string) {
	images, _ = db.GetPostImages(postId)
	return
}

func AddPostImage(postId gp.PostId, url string) (err error) {
	return db.AddPostImage(postId, url)
}

func AddPost(userId gp.UserId, text string) (postId gp.PostId, err error) {
	networks, err := GetUserNetworks(userId)
	if err != nil {
		return
	}
	postId, err = db.AddPost(userId, text, networks[0].Id)
	if err == nil {
		go cache.AddNewPost(userId, text, postId, networks[0].Id)
	}
	return
}

func GetPosts(netId gp.NetworkId, index int64, sel string) (posts []gp.PostSmall, err error) {
	conf := gp.GetConfig()
	posts, err = cache.GetPosts(netId, index, conf.PostPageSize, sel)
	if err != nil {
		posts, err = db.GetPosts(netId, index, conf.PostPageSize, sel)
		go cache.AddAllPosts(netId)
	}
	return
}

func GetComments(id gp.PostId, start int64) (comments []gp.Comment, err error) {
	conf := gp.GetConfig()
	if start+int64(conf.CommentPageSize) <= int64(conf.CommentCache) {
		comments, err = cache.GetComments(id, start)
		if err != nil {
			comments, err = db.GetComments(id, start, conf.CommentPageSize)
			go cache.AddAllComments(id)
		}
	} else {
		comments, err = db.GetComments(id, start, conf.CommentPageSize)
	}
	return
}

func GetPost(postId gp.PostId) (post gp.Post, err error) {
	return db.GetPost(postId)
}

func GetPostFull(postId gp.PostId) (post gp.PostFull, err error) {
	post.Post, err = GetPost(postId)
	if err != nil {
		return
	}
	post.Comments, err = GetComments(postId, 0)
	if err != nil {
		return
	}
	post.Likes, err = GetLikes(postId)
	return
}

func AddLike(user gp.UserId, postId gp.PostId) (err error) {
	//TODO: add like to redis
	post, err := GetPost(postId)
	if err != nil {
		return
	} else {
		err = db.CreateLike(user, postId)
		if err != nil {
			return
		} else {
			createNotification("liked", user, post.By.Id, true, postId)
		}
	}
	return
}

func DelLike(user gp.UserId, post gp.PostId) (err error) {
	return db.RemoveLike(user, post)
}

func GetLikes(post gp.PostId) (likes []gp.LikeFull, err error) {
	l, err := db.GetLikes(post)
	if err != nil {
		return
	}
	for _, like := range l {
		lf := gp.LikeFull{}
		lf.User, err = GetUser(like.UserID)
		if err != nil {
			return
		}
		lf.Time = like.Time
		likes = append(likes, lf)
	}
	return
}

func hasLiked(user gp.UserId, post gp.PostId) (liked bool, err error) {
	return db.HasLiked(user, post)
}

func likeCount(post gp.PostId) (count int, err error) {
	return db.LikeCount(post)
}

