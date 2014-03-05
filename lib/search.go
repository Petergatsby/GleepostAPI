package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"log"
	"strings"
)

var ETOOSHORT = gp.APIerror{Reason: "Your query must be at least 2 characters long"}

//UserSearchUsersInNetwork returns all the users with names beginning with first, last in netId, or ENOTALLOWED if user isn't part of this network.
//last may be omitted but first must be at least 2 characters.
func (api *API) UserSearchUsersInNetwork(user gp.UserId, first, last string, netId gp.NetworkId) (users []gp.User, err error) {
	in, err := api.UserInNetwork(user, netId)
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
		log.Printf("Searching network %d for user %s %s\n", netId, first, last)
		return api.db.SearchUsersInNetwork(first, last, netId)
	}
}
