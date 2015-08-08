package lib

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/mattbaird/elastigo/lib"
)

//UserSearchGroups searches all the groups in userID's university. It will error out if this user is not in at least one network.
func (api *API) UserSearchGroups(userID gp.UserID, name, category string) (groups []gp.GroupSubjective, err error) {
	groups = make([]gp.GroupSubjective, 0)
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	gs, err := api.searchGroups(primary.ID, name, category)
	if err != nil {
		return
	}
	for _, g := range gs {
		var group gp.GroupSubjective
		group.Group = g
		group.UnreadCount, err = api.userConversationUnread(userID, group.Conversation)
		if err != nil {
			return
		}
		var role gp.Role
		role, err = api.userRole(userID, group.ID)
		if err == nil {
			group.YourRole = &role
			var lastActivity time.Time
			lastActivity, err = api.networkLastActivity(userID, group.ID)
			if err != nil {
				log.Println(err)
				return
			} else {
				group.LastActivity = &lastActivity
			}
			group.NewPosts, err = api.groupNewPosts(userID, group.ID)
			if err != nil {
				return
			}
		}
		status, err := api.pendingRequestExists(userID, group.ID)
		if err == nil && (status == "pending" || status == "rejected") {
			group.PendingRequest = true
		}
		groups = append(groups, group)
	}
	return groups, nil
}

//SearchGroups searches for groups which are sub-groups of `parent` - ie, groups in a particular university.
func (api *API) searchGroups(parent gp.NetworkID, query, category string) (groups []gp.Group, err error) {
	groups = make([]gp.Group, 0)
	c := elastigo.NewConn()
	c.Domain = api.Config.ElasticSearch
	fields := []string{"name", "name.partial", "name.metaphone", "description"}
	groupQuery := esgroupquery{}
	parentTerm := make(map[string]string)
	parentTerm["parent"] = fmt.Sprintf("%d", parent)
	groupQuery.Query.Filtered.Filter.Must = []map[string]string{parentTerm}
	if len(category) > 0 {
		categoryTerm := make(map[string]string)
		categoryTerm["category"] = category
		groupQuery.Query.Filtered.Filter.Must = append(groupQuery.Query.Filtered.Filter.Must, categoryTerm)
	}
	for _, field := range fields {
		match := make(map[string]string)
		matcher := matcher{Match: match}
		matcher.Match[field] = query
		groupQuery.Query.Filtered.Query.Bool.Should = append(groupQuery.Query.Filtered.Query.Bool.Should, matcher)
	}
	q, _ := json.Marshal(groupQuery)
	log.Printf("%s", q)
	results, err := c.Search("gleepost", "networks", nil, groupQuery)
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

type esgroupquery struct {
	Query innergroupquery `json:"query"`
}

type innergroupquery struct {
	Filtered andfiltered `json:"filtered"`
}

type andfiltered struct {
	Filter boolmustfilter  `json:"filter"`
	Query  innerinnerquery `json:"query"`
}

type boolmustfilter struct {
	Must []map[string]string `json:"must"`
}
