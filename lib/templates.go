package lib

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"strings"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//AdminCreateTemplateFromPost saves a Post as a Template, so it can be used again.
func (api *API) AdminCreateTemplateFromPost(admin gp.UserID, post gp.PostID) (templateID gp.TemplateID, err error) {
	if !api.isAdmin(admin) {
		err = ENOTALLOWED
		return
	}
	p, err := api.getPostFull(admin, post)
	if err != nil {
		return
	}
	return api.createTemplateFromPost(p)
}

//CreateTemplateFromPost saves a Post as a Template, so it can be used again.
func (api *API) createTemplateFromPost(post gp.PostFull) (templateID gp.TemplateID, err error) {
	template, err := json.MarshalIndent(post, "", "\t")
	templateID, err = api.db.CreateTemplate(1, string(template))
	return
}

//CreatePostFromTemplate creates a new post in this network, generating it from this template.
func (api *API) CreatePostFromTemplate(network gp.NetworkID, template string) (post gp.PostID, err error) {
	//Parse template
	//Insert this network

	return
}

//AdminPrefillUniversity adds posts generated from this template set to this university, filling in any instances of <university> with universityName.
func (api *API) AdminPrefillUniversity(admin gp.UserID, network gp.NetworkID, universityName string) (err error) {
	if !api.isAdmin(admin) {
		err = ENOTALLOWED
		return
	}
	return api.prefillUniversity(network, 1, universityName)
}

//PrefillUniversity adds posts generated from this template set to this university, filling in any instances of <university> with universityName.
func (api *API) prefillUniversity(network gp.NetworkID, templateSet gp.TemplateGroupID, universityName string) (err error) {
	//Retreive all posts in this template set
	templates, err := api.db.GetTemplateSet(templateSet)
	if err != nil {
		return
	}
	domain, err := api.db.NetworkDomain(network)
	if err != nil {
		return
	}
	up := userPool{users: make(map[gp.UserID]gp.UserID), network: network, networkDomain: domain, api: api}
	for _, tpl := range templates {
		post := gp.PostFull{}
		err = json.Unmarshal([]byte(tpl), &post)
		if err != nil {
			return
		}
		post.Text = strings.Replace(post.Text, "<university>", universityName, -1)
		//Get attribs in a usable form
		attribs := make(map[string]string)
		shouldFuturize := false
		if err == nil {
			for k, v := range post.Attribs {
				s, ok := v.(string)
				switch {
				case ok && s == "event-time":
					shouldFuturize = true
					fallthrough
				case ok:
					attribs[k] = s

				}
			}
		}
		attribs["title"] = strings.Replace(attribs["title"], "<university>", universityName, -1)
		if shouldFuturize {
			post.Attribs["meta-future"] = randomFuture()
		}
		user, e := up.DuplicateUser(post.By.ID)
		if e != nil {
			return e
		}
		image := ""
		if len(post.Images) > 0 {
			image = post.Images[0]
		}
		//Get tags in a usable form
		var tags []string
		for _, cat := range post.Categories {
			tags = append(tags, cat.Tag)
		}
		var id gp.PostID
		id, _, err = api.addPostWithImage(user, network, post.Text, attribs, true, image, tags...)
		if err != nil {
			return
		}
		if len(post.Videos) > 0 {
			api.addPostVideo(user, id, post.Videos[0].ID)
		}
	}
	return nil
}

//userPool maps arbitrary user ids (for uniqueness) to particular real users (who will have a different ID once they're created)
type userPool struct {
	users         map[gp.UserID]gp.UserID
	network       gp.NetworkID
	networkDomain string
	api           *API
}

//DuplicateUser creates a new user in this userPool, or returns the existing one if it's an id we've seen before.
func (up *userPool) DuplicateUser(userID gp.UserID) (u gp.UserID, err error) {
	u, ok := up.users[userID]
	if ok {
		return u, nil
	}
	//TODO(patrick): Add check here if this email address already exists. Maybe change the map to email[userID]
	var userProfile gp.Profile
	userProfile, err = up.api.getProfile(userID, userID)
	if err != nil {
		return
	}
	names := strings.Split(userProfile.FullName, " ")
	lastName := ""
	if len(names) > 1 {
		lastName = names[1]
	}
	email := userProfile.Name + "." + lastName + "@" + up.networkDomain
	u, err = up.api.createUserSpecial(userProfile.Name, lastName, email, "TestingPass", true, up.network)
	if err != nil {
		return
	}
	err = up.api.setProfileImage(userID, userProfile.Avatar)
	if err != nil {
		return
	}
	up.users[userID] = u
	return u, nil
}

func randomFuture() string {
	return fmt.Sprintf("%dh%dm", rand.Intn(10), rand.Intn(59))
}
