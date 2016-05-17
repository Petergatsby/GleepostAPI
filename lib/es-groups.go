package lib

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/Petergatsby/GleepostAPI/lib/gp"
	"github.com/mattbaird/elastigo/lib"
)

func (api *API) esIndexGroup(groupID gp.NetworkID) {
	var pgroup gp.ParentedGroup
	var err error
	pgroup.Group, err = api.getNetwork(groupID)
	if err != nil {
		log.Println("Error getting group for ElasticSearch:", groupID, err)
	}
	pgroup.Parent, err = api.networkParent(groupID)
	if err != nil {
		log.Println("Error getting group parent for ElasticSearch:", groupID, err)
	}
	c := elastigo.NewConn()
	c.Domain = api.Config.ElasticSearch
	c.Index("gleepost", "networks", fmt.Sprintf("%d", pgroup.ID), nil, pgroup)
}

func (api *API) esBulkIndexGroups() {
	c := elastigo.NewConn()
	c.Domain = api.Config.ElasticSearch
	indexer := c.NewBulkIndexerErrors(10, 60)

	indexer.Start()
	defer indexer.Stop()
	q := "SELECT id, name, cover_img, `desc`, creator, privacy, parent " +
		"FROM network " +
		"WHERE user_group = 1 " +
		"AND privacy != 'secret' "
	s, err := api.sc.Prepare(q)
	if err != nil {
		log.Println("Error preparing statement:", err)
		return
	}
	rows, err := s.Query()
	if err != nil {
		log.Println("Error dumping groups into elasticsearch:", err)
		return
	}
	var img, desc, privacy sql.NullString
	var creator sql.NullInt64
	defer rows.Close()
	for rows.Next() {
		group := gp.ParentedGroup{}
		err = rows.Scan(&group.ID, &group.Name, &img, &desc, &creator, &privacy, &group.Parent)
		if err != nil {
			log.Println("Scan err:", err)
			continue
		}
		if img.Valid {
			group.Image = img.String
		}
		if creator.Valid {
			u, err := api.users.byID(gp.UserID(creator.Int64))
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
		log.Println("Indexing group:", group.ID, group.Name)
		indexer.Index("gleepost", "networks", fmt.Sprintf("%d", group.ID), "", "", nil, group)
	}
	log.Println("All non-secret groups indexed in ElasticSearch")
	return
}

//ElasticSearchBulkReindex adds all users/groups to the search index.
func (api *API) ElasticSearchBulkReindex() {
	api.esBulkIndexUsers()
	api.esBulkIndexGroups()
}
