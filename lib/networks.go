package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"strings"
)

func (api *API) GetUserNetworks(id gp.UserId) (nets []gp.Network, err error) {
	nets, err = api.cache.GetUserNetwork(id)
	if err != nil {
		nets, err = api.db.GetUserNetworks(id)
		if err != nil {
			return
		}
		if len(nets) == 0 {
			return nets, gp.APIerror{"User has no networks!"}
		}
		api.cache.SetUserNetwork(id, nets[0])
	}
	return
}

func (api *API) UserInNetwork(id gp.UserId, network gp.NetworkId) (in bool, err error) {
	networks, err := api.db.GetUserNetworks(id)
	if err != nil {
		return false, err
	}
	for _, n := range networks {
		if n.Id == network {
			return true, nil
		}
	}
	return false, nil
}

func (api *API) setNetwork(userId gp.UserId, netId gp.NetworkId) (err error) {
	return api.db.SetNetwork(userId, netId)
}

func (api *API) assignNetworks(user gp.UserId, email string) (networks int, err error) {
	if api.Config.RegisterOverride {
		api.setNetwork(user, 1911) //Highlands and Islands :D
	} else {
		rules, e := api.db.GetRules()
		if e != nil {
			return 0, e
		}
		for _, rule := range rules {
			if rule.Type == "email" && strings.HasSuffix(email, rule.Value) {
				e := api.setNetwork(user, rule.NetworkID)
				if e != nil {
					return networks, e
				}
				networks++
			}
		}
	}
	return
}

