package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"time"
	"log"
)

func (api *API) GetPost(postId gp.PostId) (post gp.Post, err error) {
	return api.db.GetPost(postId)
}

func (api *API) GetPostFull(postId gp.PostId) (post gp.PostFull, err error) {
	post.Post, err = api.GetPost(postId)
	if err != nil {
		return
	}
	post.Categories, err = api.postCategories(postId)
	if err != nil {
		return
	}
	post.Attribs, err = api.GetPostAttribs(postId)
	if err != nil {
		return
	}
	post.CommentCount = api.GetCommentCount(postId)
	post.Comments, err = api.GetComments(postId, 0, api.Config.CommentPageSize)
	if err != nil {
		return
	}
	post.LikeCount, post.Likes, err = api.LikesAndCount(postId)
	return
}

func (api *API) GetPosts(netId gp.NetworkId, index int64, sel string, count int) (posts []gp.PostSmall, err error) {
	ps, err := api.cache.GetPosts(netId, index, count, sel)
	if err != nil {
		posts, err = api.db.GetPosts(netId, index, count, sel)
		for i, p := range posts {
			p.Likes, err = api.GetLikes(p.Id)
			if err != nil {
				return
			}
			p.Attribs, err = api.GetPostAttribs(p.Id)
			if err != nil {
				return
			}
			posts[i] = p
		}
		go api.cache.AddPostsFromDB(netId, api.db)
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

//GetPostsByCategory acts the same as getPosts but only returns posts which are in the category with tag category.
//It has no caching layer at the moment.
//Should restrict access based on user.
func (api *API) GetPostsByCategory(netId gp.NetworkId, index int64, sel string, count int, category string) (posts []gp.PostSmall, err error) {
	posts, err = api.db.GetPostsByCategory(netId, index, count, sel, category)
	if err != nil {
		return
	}
	for i, p := range posts {
		log.Println("Post!")
		p.Likes, err = api.GetLikes(p.Id)
		if err != nil {
			return
		}
		p.Attribs, err = api.GetPostAttribs(p.Id)
		if err != nil {
			return
		}
		posts[i] = p
	}
	return
}

func (api *API) PostSmall(p gp.PostCore) (post gp.PostSmall, err error) {
	post.Id = p.Id
	post.By = p.By
	post.Time = p.Time
	post.Text = p.Text
	post.Images = api.GetPostImages(p.Id)
	post.CommentCount = api.GetCommentCount(p.Id)
	post.Categories, err = api.postCategories(p.Id)
	if err != nil {
		return
	}
	post.Attribs, err = api.GetPostAttribs(p.Id)
	if err != nil {
		return
	}
	post.LikeCount, post.Likes, err = api.LikesAndCount(p.Id)
	if err != nil {
		return
	}
	return
}

func (api *API) GetComments(id gp.PostId, start int64, count int) (comments []gp.Comment, err error) {
	comments, err = api.cache.GetComments(id, start, count)
	if err != nil {
		comments, err = api.db.GetComments(id, start, count)
		go api.cache.AddAllCommentsFromDB(id, api.db)
	}
	return
}

func (api *API) GetCommentCount(id gp.PostId) (count int) {
	count, err := api.cache.GetCommentCount(id)
	if err != nil {
		count = api.db.GetCommentCount(id)
	}
	return count
}

func (api *API) GetPostImages(postId gp.PostId) (images []string) {
	images, _ = api.db.GetPostImages(postId)
	return
}

func (api *API) postCategories(post gp.PostId) (categories []gp.PostCategory, err error) {
	return api.db.PostCategories(post)
}

func (api *API) GetLikes(post gp.PostId) (likes []gp.LikeFull, err error) {
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

func (api *API) hasLiked(user gp.UserId, post gp.PostId) (liked bool, err error) {
	return api.db.HasLiked(user, post)
}

func (api *API) likeCount(post gp.PostId) (count int, err error) {
	return api.db.LikeCount(post)
}

func (api *API) LikesAndCount(post gp.PostId) (count int, likes []gp.LikeFull, err error) {
	likes, err = api.GetLikes(post)
	if err != nil {
		return
	}
	count, err = api.likeCount(post)
	return
}

func (api *API) CreateComment(postId gp.PostId, userId gp.UserId, text string) (commId gp.CommentId, err error) {
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
		if userId != post.By.Id {
			go api.createNotification("commented", userId, post.By.Id, true, postId)
		}
		go api.cache.AddComment(postId, comment)
	}
	return commId, err
}

func (api *API) AddPostImage(postId gp.PostId, url string) (err error) {
	return api.db.AddPostImage(postId, url)
}

func (api *API) AddPost(userId gp.UserId, text string, attribs map[string]string, tags ...string) (postId gp.PostId, err error) {
	networks, err := api.GetUserNetworks(userId)
	if err != nil {
		return
	}
	postId, err = api.db.AddPost(userId, text, networks[0].Id)
	if err == nil {
		if len(tags) > 0 {
			err = api.TagPost(postId, tags...)
			if err != nil {
				return
			}
		}
		if len(attribs) > 0 {
			err = api.SetPostAttribs(postId, attribs)
			if err != nil {
				return
			}
		}
		user, err := api.db.GetUser(userId)
		if err == nil {
			post := gp.Post{Id: postId, By: user, Text: text, Time: time.Now().UTC()}
			go api.cache.AddPost(post)
			go api.cache.AddPostToNetwork(post, networks[0].Id)
		}
	}
	return
}

func (api *API) AddPostWithImage(userId gp.UserId, text string, attribs map[string]string, image string, tags ...string) (postId gp.PostId, err error) {
	postId, err = api.AddPost(userId, text, attribs, tags...)
	if err != nil {
		return
	}
	exists, err := api.UserUploadExists(userId, image)
	if exists && err == nil {
		err = api.AddPostImage(postId, image)
		return
	}
	return
}

//
func (api *API) TagPost(post gp.PostId, tags ...string) (err error) {
	//TODO: Only allow the post owner to tag
	return api.tagPost(post, tags...)
}

func (api *API) tagPost(post gp.PostId, tags ...string) (err error) {
	//TODO: Stick this shit in cache
	return api.db.TagPost(post, tags...)
}

func (api *API) AddLike(user gp.UserId, postId gp.PostId) (err error) {
	//TODO: add like to redis
	post, err := api.GetPost(postId)
	if err != nil {
		return
	} else {
		err = api.db.CreateLike(user, postId)
		if err != nil {
			return
		} else {
			if user != post.By.Id {
				api.createNotification("liked", user, post.By.Id, true, postId)
			}
		}
	}
	return
}

func (api *API) DelLike(user gp.UserId, post gp.PostId) (err error) {
	return api.db.RemoveLike(user, post)
}

//SetPostAttribs associates a s<F6>
func (api *API) SetPostAttribs(post gp.PostId, attribs map[string]string) (err error) {
	return api.db.SetPostAttribs(post, attribs)
}

func (api *API) GetPostAttribs(post gp.PostId) (attribs map[string]string, err error) {
	return api.db.GetPostAttribs(post)
}
