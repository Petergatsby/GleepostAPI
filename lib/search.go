package lib

import (
	"database/sql"
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
	in, err := api.userInNetwork(user, netID)
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
		return api.searchUsersInNetwork(first, last, netID)
	}
}

//UserSearchUsersInPrimaryNetwork returns all the users with names beginning with first, last in netId, or ENOTALLOWED if user isn't part of this network.
//last may be omitted but first must be at least 2 characters.
func (api *API) UserSearchUsersInPrimaryNetwork(userID gp.UserID, first, last string) (users []gp.FullNameUser, err error) {
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	return api.userSearchUsersInNetwork(userID, first, last, primary.ID)

}

//UserSearchGroups searches all the groups in userID's university. It will error out if this user is not in at least one network.
func (api *API) UserSearchGroups(userID gp.UserID, name string) (groups []gp.Group, err error) {
	groups = make([]gp.Group, 0)
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	return api.searchGroups(primary.ID, name)
}

//SearchUsersInNetwork returns users whose name begins with first and last within netId.
func (api *API) searchUsersInNetwork(first, last string, netID gp.NetworkID) (users []gp.FullNameUser, err error) {
	users = make([]gp.FullNameUser, 0)
	search := "SELECT id, avatar, firstname, lastname, official " +
		"FROM users JOIN user_network ON users.id = user_network.user_id " +
		"WHERE network_id = ? " +
		"AND firstname LIKE ? " +
		"AND lastname LIKE ?"
	first += "%"
	last += "%"
	log.Println(search, first, last)
	s, err := api.sc.Prepare(search)
	if err != nil {
		return
	}
	rows, err := s.Query(netID, first, last)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var av, last sql.NullString
		var user gp.FullNameUser
		err = rows.Scan(&user.ID, &av, &user.Name, &last, &user.Official)
		if err != nil {
			return
		}
		if av.Valid {
			user.Avatar = av.String
		}
		if last.Valid {
			user.FullName = user.Name + " " + last.String
		}
		users = append(users, user)
	}
	return
}

//SearchGroups searches for groups which are 'within' parent; it currently just matches %name%.
func (api *API) searchGroups(parent gp.NetworkID, name string) (groups []gp.Group, err error) {
	groups = make([]gp.Group, 0)
	q := "SELECT id, name, cover_img, `desc`, creator, privacy " +
		"FROM network " +
		"WHERE user_group = 1 " +
		"AND parent = ? " +
		"AND privacy != 'secret' " +
		"AND name LIKE ?"
	name = "%" + name + "%"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(parent, name)
	if err != nil {
		return
	}
	defer rows.Close()
	var img, desc, privacy sql.NullString
	var creator sql.NullInt64
	for rows.Next() {
		group := gp.Group{}
		err = rows.Scan(&group.ID, &group.Name, &img, &desc, &creator, &privacy)
		if err != nil {
			return
		}
		if img.Valid {
			group.Image = img.String
		}
		if creator.Valid {
			u, err := api.getUser(gp.UserID(creator.Int64))
			if err == nil {
				group.Creator = &u
			}
			group.MemberCount, _ = api.groupMemberCount(group.ID)
		}
		if desc.Valid {
			group.Desc = desc.String
		}
		if privacy.Valid {
			group.Privacy = privacy.String
		}
		groups = append(groups, group)
	}
	return
}
