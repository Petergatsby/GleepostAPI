package lib

import (
	"encoding/json"
	"fmt"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/mattbaird/elastigo/lib"
)

//UserSearchUsersInNetwork returns all the users with names beginning with first, last in netId, or ENOTALLOWED if user isn't part of this network.
//last may be omitted but first must be at least 2 characters.
func (api *API) userSearchUsersInNetwork(user gp.UserID, query string, netID gp.NetworkID) (users []gp.PublicProfile, err error) {
	users = make([]gp.PublicProfile, 0)
	in, err := api.UserInNetwork(user, netID)
	switch {
	case err != nil:
		return
	case !in:
		return users, &ENOTALLOWED
	default:
		return api.searchUsersInNetwork(query, netID)
	}
}

//UserSearchUsersInPrimaryNetwork returns all the users with names beginning with first, last in netId, or ENOTALLOWED if user isn't part of this network.
//last may be omitted but first must be at least 2 characters.
func (api *API) UserSearchUsersInPrimaryNetwork(userID gp.UserID, query string) (users []gp.PublicProfile, err error) {
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	return api.userSearchUsersInNetwork(userID, query, primary.ID)
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

//SearchUsersInNetwork returns users whose name begins with first and last within netId.
func (api *API) searchUsersInNetwork(query string, netID gp.NetworkID) (users []gp.PublicProfile, err error) {
	users = make([]gp.PublicProfile, 0)
	c := elastigo.NewConn()
	c.Domain = api.Config.ElasticSearch
	esQuery := userQuery(query, netID)
	results, err := c.Search("gleepost", "users", nil, esQuery)
	if err != nil {
		return
	}
	for _, hit := range results.Hits.Hits {
		var user gp.PublicProfile
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
