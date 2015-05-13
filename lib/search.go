package lib

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/mattbaird/elastigo/lib"
)

//ETOOSHORT represents a search query which is too short.
var ETOOSHORT = gp.APIerror{Reason: "Your query must be at least 2 characters long"}

//UserSearchUsersInNetwork returns all the users with names beginning with first, last in netId, or ENOTALLOWED if user isn't part of this network.
//last may be omitted but first must be at least 2 characters.
func (api *API) userSearchUsersInNetwork(user gp.UserID, query string, netID gp.NetworkID) (users []gp.FullNameUser, err error) {
	users = make([]gp.FullNameUser, 0)
	in, err := api.userInNetwork(user, netID)
	switch {
	case err != nil:
		return
	case !in:
		return users, &ENOTALLOWED
	case len(query) < 2:
		return users, &ETOOSHORT
	default:
		log.Printf("Searching network %d for user %s\n", netID, query)
		return api.searchUsersInNetwork(query, netID)
	}
}

//UserSearchUsersInPrimaryNetwork returns all the users with names beginning with first, last in netId, or ENOTALLOWED if user isn't part of this network.
//last may be omitted but first must be at least 2 characters.
func (api *API) UserSearchUsersInPrimaryNetwork(userID gp.UserID, query string) (users []gp.FullNameUser, err error) {
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	return api.userSearchUsersInNetwork(userID, query, primary.ID)

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
func (api *API) searchUsersInNetwork(query string, netID gp.NetworkID) (users []gp.FullNameUser, err error) {
	users = make([]gp.FullNameUser, 0)
	c := elastigo.NewConn()
	c.Domain = api.Config.ElasticSearch
	//do not actually do this: vulnerable to injection
	esQuery := fmt.Sprintf("{ \"query\" : { \"term\" : { \"full_name\" : \"%s\" } } }", query)
	results, err := c.Search("gleepost", "users", nil, esQuery)
	if err != nil {
		return
	}
	for _, hit := range results.Hits.Hits {
		var user gp.FullNameUser
		err = json.Unmarshal(*hit.Source, &user)
		if err != nil {
			return
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
