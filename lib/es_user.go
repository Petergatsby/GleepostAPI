package lib

import (
	"fmt"
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/mattbaird/elastigo/lib"
)

func (api *API) esIndexUser(userID gp.UserID) {
	user, err := api._getProfile(userID)
	if err != nil {
		log.Println("Error getting profile for elasticsearch index:", userID, err)
		return
	}
	user.Network, err = api.getUserUniversity(user.ID)
	if err != nil {
		return
	}
	c := elastigo.NewConn()
	c.Domain = api.Config.ElasticSearch
	c.Index("gleepost", "users", fmt.Sprintf("%d", user.ID), nil, user)
}

func (api *API) esBulkIndexUsers() {
	c := elastigo.NewConn()
	c.Domain = api.Config.ElasticSearch
	indexer := c.NewBulkIndexerErrors(10, 60)

	indexer.Start()
	defer indexer.Stop()
	s, err := api.sc.Prepare("SELECT id FROM users")
	if err != nil {
		log.Println("error running elasticsearch dump:", err)
		return
	}
	rows, err := s.Query()
	if err != nil {
		log.Println("error running elasticsearch dump:", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var userID gp.UserID
		err = rows.Scan(&userID)
		if err != nil {
			log.Println("error running elasticsearch dump:", err)
			continue
		}
		user, err := api._getProfile(userID)
		if err != nil {
			log.Println("Error getting profile for elasticsearch index:", userID, err)
			continue
		}
		user.Network, err = api.getUserUniversity(user.ID)
		if err != nil {
			log.Println("Error getting user university for elasticsearch index:", userID, err)
			continue
		}
		indexer.Index("gleepost", "users", fmt.Sprintf("%d", userID), "", nil, user, true)
	}
	log.Println("All users indexed in ElasticSearch")
	return

}
