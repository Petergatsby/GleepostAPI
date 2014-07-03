package lib

import (
	"errors"
	"math/rand"
	"strings"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//ErrNoUsers happens when you try to DuplicateUsers with no users.
var ErrNoUsers = errors.New("must supply at least one user to be duplicated")

//ErrNoPosts happens when you try to DuplicatePosts with no posts.
var ErrNoPosts = errors.New("must supply at least one post to be duplicated")

//DuplicateUsers takes a list of users and copies them into another network, with a random email address and the password "TestingPass".
func (api *API) DuplicateUsers(into gp.NetworkID, users ...gp.UserID) (copiedUsers []gp.UserID, err error) {
	if len(users) == 0 {
		err = ErrNoUsers
		return
	}
	for _, u := range users {
		var user gp.Profile
		user, err = api.GetProfile(u)
		if err != nil {
			return
		}
		names := strings.Split(user.FullName, " ")
		lastName := ""
		if len(names) > 0 {
			lastName = names[1]
		}
		email := string(rand.Uint32()) + "@gleepost.com"
		var userID gp.UserID
		userID, err = api.CreateUserSpecial(user.Name, lastName, email, "TestingPass", true, into)
		if err != nil {
			return
		}
		copiedUsers = append(copiedUsers, userID)
		//SetAvatar
		err = api.SetProfileImage(userID, user.Avatar)
		if err != nil {
			return
		}
	}
	return
}

//DuplicatePosts takes a bunch of posts and copies them into another network, ie for demos. It can also copy their owners.
func (api *API) DuplicatePosts(into gp.NetworkID, copyUsers bool, posts ...gp.PostID) (err error) {
	if len(posts) == 0 {
		return ErrNoPosts
	}
	for _, p := range posts {
		var post gp.Post
		post, err = api.GetPost(p)
		if err != nil {
			return
		}
		var userID gp.UserID
		if copyUsers {
			var userIDs []gp.UserID
			userIDs, err = api.DuplicateUsers(into, post.By.ID)
			if err != nil {
				return
			}
			userID = userIDs[0]
		} else {
			userID = post.By.ID
		}
		//Get attribs in a usable form
		attribs := make(map[string]string)
		for k, v := range post.Attribs {
			s, ok := v.(string)
			if ok {
				attribs[k] = s
			}
		}
		//Get tags in a usable form
		var tags []string
		for _, cat := range post.Categories {
			tags = append(tags, cat.Tag)
		}
		image := ""
		if len(post.Images) > 0 {
			image = post.Images[0]
		}
		_, err = api.AddPostWithImage(userID, into, post.Text, attribs, image, tags...)
		if err != nil {
			return
		}
	}
	return
}
