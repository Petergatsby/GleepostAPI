package lib

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/mattbaird/elastigo/lib"
)

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
	esQuery := userQuery(query, netID)
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

type esquery struct {
	Query innerquery `json:"query"`
}
type innerquery struct {
	Filtered filteredquery `json:"filtered"`
}
type filteredquery struct {
	Filter filter          `json:"filter"`
	Query  innerinnerquery `json:"query"`
}
type innerinnerquery struct {
	Bool boolquery `json:"bool"`
}
type filter struct {
	Term map[string]string `json:"term"`
}
type boolquery struct {
	Should []matcher `json:"should"`
}
type matcher struct {
	Match map[string]string `json:"match"`
}

func userQuery(query string, netID gp.NetworkID) (esQuery esquery) {
	fields := []string{"name", "name.partial", "name.metaphone", "full_name", "full_name.partial", "full_name.metaphone"}
	term := make(map[string]string)
	term["network.id"] = fmt.Sprintf("%d", netID)
	esQuery.Query.Filtered.Filter.Term = term
	for _, field := range fields {
		match := make(map[string]string)
		matcher := matcher{Match: match}
		matcher.Match[field] = query
		esQuery.Query.Filtered.Query.Bool.Should = append(esQuery.Query.Filtered.Query.Bool.Should, matcher)
	}
	return esQuery
}

//SearchGroups searches for groups which are 'within' parent; it currently just matches %name%.
func (api *API) searchGroups(parent gp.NetworkID, query string) (groups []gp.Group, err error) {
	groups = make([]gp.Group, 0)
	c := elastigo.NewConn()
	c.Domain = api.Config.ElasticSearch
	fields := []string{"name", "description"}
	esQuery := esquery{}
	term := make(map[string]string)
	term["parent"] = fmt.Sprintf("%d", parent)
	esQuery.Query.Filtered.Filter.Term = term
	for _, field := range fields {
		match := make(map[string]string)
		matcher := matcher{Match: match}
		matcher.Match[field] = query
		esQuery.Query.Filtered.Query.Bool.Should = append(esQuery.Query.Filtered.Query.Bool.Should, matcher)
	}
	results, err := c.Search("gleepost", "networks", nil, esQuery)
	if err != nil {
		return
	}
	for _, hit := range results.Hits.Hits {
		var group gp.Group
		err = json.Unmarshal(*hit.Source, &group)
		if err != nil {
			return
		}
		groups = append(groups, group)
	}
	return
}
