package lib

import (
	"fmt"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//GetUser returns the User with this ID. It hits the cache first, so some details may be out of date.
func (api *API) getUser(id gp.UserID) (user gp.User, err error) {
	/* Hits the api.cache then the api.db
	only I'm not 100% confident yet with what
	happens when you attempt to get a redis key
	that doesn't exist in redigo! */
	user, err = api.cache.GetUser(id)
	if err != nil {
		user, err = api.db.GetUser(id)
		if err == nil {
			api.cache.SetUser(user)
		}
	}
	return
}

//UserGetProfile returns the Profile (extended info) for the user with this ID.
func (api *API) UserGetProfile(userID, otherID gp.UserID) (user gp.Profile, err error) {
	if userID == otherID {
		return api.getProfile(userID, otherID)
	}
	shared, e := api.haveSharedNetwork(userID, otherID)
	switch {
	case e != nil:
		fallthrough
	case !shared:
		err = &ENOTALLOWED
	default:
		return api.getProfile(userID, otherID)
	}
	return
}

//getProfile returns the Profile (extended info) for the user with this ID.
func (api *API) getProfile(perspective, otherID gp.UserID) (user gp.Profile, err error) {
	user, err = api.db.GetProfile(otherID)
	if err != nil {
		return
	}
	nets, err := api.getUserNetworks(user.ID)
	if err != nil {
		return
	}
	rsvps, err := api.subjectiveRSVPCount(perspective, otherID)
	if err != nil {
		return
	}
	user.RSVPCount = rsvps
	groupCount, err := api.db.SubjectiveMembershipCount(perspective, otherID)
	if err != nil {
		return
	}
	user.GroupCount = groupCount
	postCount, err := api.db.UserPostCount(perspective, otherID)
	if err != nil {
		return
	}
	user.PostCount = postCount
	user.Network = nets[0]
	return
}

//IsAdmin returns true if tis user is a member of the Admin network specified in the config.
func (api *API) isAdmin(user gp.UserID) (admin bool) {
	in, err := api.userInNetwork(user, gp.NetworkID(api.Config.Admins))
	if err == nil && in {
		return true
	}
	return false
}

//CreateUserSpecial manually creates a user with these details, bypassing validation etc
func (api *API) UserCreateUserSpecial(creator gp.UserID, first, last, email, pass string, verified bool, primaryNetwork gp.NetworkID) (userID gp.UserID, err error) {
	if !api.isAdmin(creator) {
		err = ENOTALLOWED
		return
	}
	return api.createUserSpecial(first, last, email, pass, verified, primaryNetwork)
}

func (api *API) createUserSpecial(first, last, email, pass string, verified bool, primaryNetwork gp.NetworkID) (userID gp.UserID, err error) {
	userID, err = api.createUser(first, last, pass, email)
	if err != nil {
		return
	}
	if verified {
		err = api.db.Verify(userID)
		if err != nil {
			return
		}
	}
	err = api.setNetwork(userID, primaryNetwork)
	return
}

func (api *API) inviteURL(token, email string) string {
	if api.Config.DevelopmentMode {
		return fmt.Sprintf("https://dev.gleepost.com/?invite=%s&email=%s", token, email)
	}
	return fmt.Sprintf("https://gleepost.com/?invite=%s&email=%s", token, email)
}

func (api *API) issueInviteEmail(email string, from gp.User, group gp.Group, token string) (err error) {
	url := api.inviteURL(token, email)
	subject := fmt.Sprintf("%s has invited you to the private group \"%s\" on Gleepost.", from.Name, group.Name)
	html := "<html><body>" +
		"Don't miss out on their events - <a href=" + url + ">Click here to accept the invitation.</a><br>" +
		"On your phone? <a href=\"" + inviteCampaignIOS + "\">install the app on your iPhone here</a>" +
		" or <a href=\"" + inviteCampaignAndroid + "\">click here to get the Android app.</a>" +
		"</body></html>"
	err = api.mail.SendHTML(email, subject, html)
	return
}

//GetEmail returns this user's email address.
func (api *API) getEmail(id gp.UserID) (email string, err error) {
	return api.db.GetEmail(id)
}

//SetUserName updates this user's name.
func (api *API) SetUserName(id gp.UserID, firstName, lastName string) (err error) {
	return api.db.SetUserName(id, firstName, lastName)
}

//UserChangeTagline sets this user's tagline (obviously enough...)
func (api *API) UserChangeTagline(userID gp.UserID, tagline string) (err error) {
	return api.db.UserChangeTagline(userID, tagline)
}

//UserWithEmail returns the userID this email is associated with, or err if there isn't one.
func (api *API) userWithEmail(email string) (id gp.UserID, err error) {
	return api.db.UserWithEmail(email)
}

//UserSetProfileImage updates this user's profile image to the new url
func (api *API) UserSetProfileImage(id gp.UserID, url string) (err error) {
	exists, err := api.userUploadExists(id, url)
	if err != nil {
		return
	}
	if !exists {
		return NoSuchUpload
	}
	return api.setProfileImage(id, url)
}

func (api *API) setProfileImage(id gp.UserID, url string) (err error) {
	err = api.db.SetProfileImage(id, url)
	if err == nil {
		go api.cache.SetProfileImage(id, url)
	}
	return

}

//SetBusyStatus records whether you are busy or not.
func (api *API) SetBusyStatus(id gp.UserID, busy bool) (err error) {
	err = api.db.SetBusyStatus(id, busy)
	if err == nil {
		go api.cache.SetBusyStatus(id, busy)
	}
	return
}

//BusyStatus returns true if this user is busy.
func (api *API) BusyStatus(id gp.UserID) (busy bool, err error) {
	busy, err = api.db.BusyStatus(id)
	return
}

func (api *API) userPing(id gp.UserID) {
	api.cache.UserPing(id, api.Config.OnlineTimeout)
}

func (api *API) userIsOnline(id gp.UserID) bool {
	return api.cache.UserIsOnline(id)
}

//UserHasPosted returns true if user has ever created a post from the perspective of perspective.
//TODO: Implement a direct version
func (api *API) UserHasPosted(user gp.UserID, perspective gp.UserID) (posted bool, err error) {
	posts, err := api.GetUserPosts(user, perspective, gp.OSTART, 0, 1, "")
	if err != nil {
		return
	}
	if len(posts) > 0 {
		return true, nil
	}
	return false, nil
}
