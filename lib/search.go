package lib

import (
	"log"
	"strings"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//ETOOSHORT represents a search query which is too short.
var ETOOSHORT = gp.APIerror{Reason: "Your query must be at least 2 characters long"}

//UserSearchUsersInNetwork returns all the users with names beginning with first, last in netId, or ENOTALLOWED if user isn't part of this network.
//last may be omitted but first must be at least 2 characters.
func (api *API) userSearchUsersInNetwork(user gp.UserID, first, last string, netID gp.NetworkID) (users []gp.FullNameUser, err error) {
	users = make([]gp.FullNameUser, 0)
	in, err := api.db.UserInNetwork(user, netID)
	//I don't like the idea of people being able to look for eg. %a%
	first = strings.Replace(first, "%", "", -1)
	last = strings.Replace(last, "%", "", -1)
	switch {
	case err != nil:
		return
	case !in:
		return users, &ENOTALLOWED
	case len(first) < 2:
		return users, &ETOOSHORT
	default:
		log.Printf("Searching network %d for user %s %s\n", netID, first, last)
		return api.db.SearchUsersInNetwork(first, last, netID)
	}
}

//UserSearchUsersInPrimaryNetwork returns all the users with names beginning with first, last in netId, or ENOTALLOWED if user isn't part of this network.
//last may be omitted but first must be at least 2 characters.
func (api *API) UserSearchUsersInPrimaryNetwork(userID gp.UserID, first, last string) (users []gp.FullNameUser, err error) {
	primary, err := api.db.GetUserUniversity(userID)
	if err != nil {
		return
	}
	return api.userSearchUsersInNetwork(userID, first, last, primary.ID)

}

//UserSearchGroups searches all the groups in userID's university. It will error out if this user is not in at least one network.
func (api *API) UserSearchGroups(userID gp.UserID, name string) (groups []gp.Group, err error) {
	groups = make([]gp.Group, 0)
	primary, err := api.db.GetUserUniversity(userID)
	if err != nil {
		return
	}
	return api.db.SearchGroups(primary.ID, name)
}
