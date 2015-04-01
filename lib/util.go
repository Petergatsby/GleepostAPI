package lib

import (
	"errors"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

var (
	//ErrNoUsers happens when you try to DuplicateUsers with no users.
	ErrNoUsers = errors.New("must supply at least one user to be duplicated")
	//ErrNoPosts happens when you try to DuplicatePosts with no posts.
	ErrNoPosts = errors.New("must supply at least one post to be duplicated")
)

//DuplicateUsers takes a list of users and copies them into another network, with a random email address and the password "TestingPass".
func (api *API) duplicateUsers(into gp.NetworkID, users ...gp.UserID) (copiedUsers []gp.UserID, err error) {
	if len(users) == 0 {
		err = ErrNoUsers
		return
	}
	for _, u := range users {
		var user gp.Profile
		user, err = api.getProfile(u, u)
		if err != nil {
			return
		}
		names := strings.Split(user.FullName, " ")
		lastName := ""
		if len(names) > 1 {
			lastName = names[1]
		}
		email := strconv.FormatUint(uint64(rand.New(rand.NewSource(time.Now().UnixNano())).Uint32()), 10) + "@gleepost.com"
		var userID gp.UserID
		userID, err = api.createUserSpecial(user.Name, lastName, email, "TestingPass", true, into)
		if err != nil {
			return
		}
		copiedUsers = append(copiedUsers, userID)
		//SetAvatar
		err = api.setProfileImage(userID, user.Avatar)
		if err != nil {
			return
		}
	}
	return
}

//UserDuplicatePosts takes a bunch of posts and copies them into another network, ie for demos. It can also copy their owners. If regEx is set, it will replace all matches in the post attribs and body with replacement.
func (api *API) UserDuplicatePosts(admin gp.UserID, into gp.NetworkID, copyUsers bool, regEx, replacement string, posts ...gp.PostID) (duplicates []gp.PostID, err error) {
	if !api.isAdmin(admin) {
		err = ENOTALLOWED
		return
	}
	return api.duplicatePosts(into, copyUsers, regEx, replacement, posts...)
}

func (api *API) duplicatePosts(into gp.NetworkID, copyUsers bool, regEx, replacement string, posts ...gp.PostID) (duplicates []gp.PostID, err error) {
	var re *regexp.Regexp
	if len(regEx) > 0 {
		re, err = regexp.Compile(regEx)
		if err != nil {
			return
		}
	}
	if len(posts) == 0 {
		err = ErrNoPosts
		return
	}
	for _, p := range posts {
		var post gp.Post
		post, err = api.GetPost(p)
		if err != nil {
			return
		}
		if len(regEx) > 0 {
			post.Text = re.ReplaceAllString(post.Text, replacement)
		}
		var userID gp.UserID
		if copyUsers {
			var userIDs []gp.UserID
			userIDs, err = api.duplicateUsers(into, post.By.ID)
			if err != nil {
				return
			}
			userID = userIDs[0]
		} else {
			userID = post.By.ID
		}
		//Get attribs in a usable form
		attribs := make(map[string]string)
		atts := make(map[string]interface{})
		atts, err = api.getPostAttribs(post.ID)
		if err == nil {
			for k, v := range atts {
				s, ok := v.(string)
				switch {
				case ok && len(regEx) > 0:
					attribs[k] = re.ReplaceAllString(s, replacement)
				case ok:
					attribs[k] = s
				default:
				}
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
		var video gp.VideoID
		if len(post.Videos) > 0 {
			video = post.Videos[0].ID
		}
		var id gp.PostID
		id, _, err = api.UserAddPost(userID, into, post.Text, attribs, video, true, image, tags...)
		if err != nil {
			return
		}
		duplicates = append(duplicates, id)
	}
	return
}

//KeepPostsInFuture checks a list of posts every PollInterval and pushes them into the future if neccessary
func (api *API) KeepPostsInFuture(pollInterval time.Duration) {
	t := time.Tick(pollInterval)
	for {
		err := api.db.KeepPostsInFuture()
		if err != nil {
			log.Println(err)
		}
		<-t
	}
}
