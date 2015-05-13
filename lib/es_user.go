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
		log.Println(user)
	}
	user.Network, err = api.getUserUniversity(user.ID)
	if err != nil {
		return
	}
	c := elastigo.NewConn()
	c.Domain = api.Config.ElasticSearch
	c.Index("gleepost", "users", fmt.Sprintf("%d", user.ID), nil, user)
}
