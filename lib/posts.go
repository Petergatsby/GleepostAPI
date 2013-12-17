package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"time"
)

func (api *API)GetCommentCount(id gp.PostId) (count int) {
	count, err := api.cache.GetCommentCount(id)
	if err != nil {
		count = api.db.GetCommentCount(id)
	}
	return count
}

func (api *API)CreateComment(postId gp.PostId, userId gp.UserId, text string) (commId gp.CommentId, err error) {
	post, err := api.GetPost(postId)
	if err != nil {
		return
	}
	commId, err = api.db.CreateComment(postId, userId, text)
	if err == nil {
		user, e := api.GetUser(userId)
		if e != nil {
			return commId, e
		}
		comment := gp.Comment{Id: commId, Post: postId, By: user, Time: time.Now().UTC(), Text: text}
		go api.createNotification("commented", userId, post.By.Id, true, postId)
		go api.cache.AddComment(postId, comment)
	}
	return commId, err
}

func (api *API)GetPostImages(postId gp.PostId) (images []string) {
	images, _ = api.db.GetPostImages(postId)
	return
}

func (api *API)AddPostImage(postId gp.PostId, url string) (err error) {
	return api.db.AddPostImage(postId, url)
}

func (api *API)AddPost(userId gp.UserId, text string) (postId gp.PostId, err error) {
	networks, err := api.GetUserNetworks(userId)
	if err != nil {
		return
	}
	postId, err = api.db.AddPost(userId, text, networks[0].Id)
	if err == nil {
		user, err := api.db.GetUser(userId)
		if err == nil {
			post := gp.Post{Id:postId, By:user, Text:text, Time:time.Now().UTC()}
			go api.cache.AddPost(post)
			go api.cache.AddPostToNetwork(post, networks[0].Id)
		}
	}
	return
}

func (api *API)GetPosts(netId gp.NetworkId, index int64, sel string) (posts []gp.PostSmall, err error) {
	conf := gp.GetConfig()
	ps, err := api.cache.GetPosts(netId, index, conf.PostPageSize, sel)
	if err != nil {
		posts, err = api.db.GetPosts(netId, index, conf.PostPageSize, sel)
		go api.cache.AddAllPostsFromDB(netId, api.db)
	} else {
		var post gp.PostSmall
		for _, p := range ps {
			post, err = api.PostSmall(p)
			if err != nil {
				return
			}
			posts = append(posts, post)
		}
	}
	return
}

func (api *API)PostSmall(p gp.PostCore) (post gp.PostSmall, err error) {
	post.Id = p.Id
	post.By = p.By
	post.Time = p.Time
	post.Text = p.Text
	post.Images = api.GetPostImages(p.Id)
	post.CommentCount = api.GetCommentCount(p.Id)
	post.LikeCount, err = api.likeCount(p.Id)
	if err != nil {
		return
	}
	return
}

func (api *API)GetComments(id gp.PostId, start int64) (comments []gp.Comment, err error) {
	conf := gp.GetConfig()
	if start+int64(conf.CommentPageSize) <= int64(conf.CommentCache) {
		comments, err = api.cache.GetComments(id, start)
		if err != nil {
			comments, err = api.db.GetComments(id, start, conf.CommentPageSize)
			go api.cache.AddAllCommentsFromDB(id, api.db)
		}
	} else {
		comments, err = api.db.GetComments(id, start, conf.CommentPageSize)
	}
	return
}

func (api *API)GetPost(postId gp.PostId) (post gp.Post, err error) {
	return api.db.GetPost(postId)
}

func (api *API)GetPostFull(postId gp.PostId) (post gp.PostFull, err error) {
	post.Post, err = api.GetPost(postId)
	if err != nil {
		return
	}
	post.Comments, err = api.GetComments(postId, 0)
	if err != nil {
		return
	}
	post.Likes, err = api.GetLikes(postId)
	return
}

func (api *API)AddLike(user gp.UserId, postId gp.PostId) (err error) {
	//TODO: add like to redis
	post, err := api.GetPost(postId)
	if err != nil {
		return
	} else {
		err = api.db.CreateLike(user, postId)
		if err != nil {
			return
		} else {
			api.createNotification("liked", user, post.By.Id, true, postId)
		}
	}
	return
}

func (api *API)DelLike(user gp.UserId, post gp.PostId) (err error) {
	return api.db.RemoveLike(user, post)
}

func (api *API)GetLikes(post gp.PostId) (likes []gp.LikeFull, err error) {
	l, err := api.db.GetLikes(post)
	if err != nil {
		return
	}
	for _, like := range l {
		lf := gp.LikeFull{}
		lf.User, err = api.GetUser(like.UserID)
		if err != nil {
			return
		}
		lf.Time = like.Time
		likes = append(likes, lf)
	}
	return
}

func (api *API)hasLiked(user gp.UserId, post gp.PostId) (liked bool, err error) {
	return api.db.HasLiked(user, post)
}

func (api *API)likeCount(post gp.PostId) (count int, err error) {
	return api.db.LikeCount(post)
}

