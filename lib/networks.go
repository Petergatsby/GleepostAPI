package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"strings"
	"log"
)

func (api *API) GetUserNetworks(id gp.UserId) (nets []gp.Network, err error) {
	nets, err = api.cache.GetUserNetworks(id)
	if err != nil {
		nets, err = api.db.GetUserNetworks(id, false)
		if err != nil {
			return
		}
		if len(nets) == 0 {
			return nets, gp.APIerror{"User has no networks!"}
		}
		api.cache.SetUserNetworks(id, nets...)
	}
	return
}

//GetUserGroups is the same as GetUserNetworks, except it omits "official" networks (ie, universities)
func (api *API) GetUserGroups(id gp.UserId) (groups []gp.Network, err error) {
	groups, err = api.db.GetUserNetworks(id, true)
	return
}

func (api *API) UserInNetwork(id gp.UserId, network gp.NetworkId) (in bool, err error) {
	networks, err := api.db.GetUserNetworks(id, false)
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

//isGroup returns false if this network isn't a group (ie isn't user-created) and error if the group doesn't exist.
func (api *API) isGroup(netId gp.NetworkId) (group bool, err error) {
	return api.db.IsGroup(netId)
}

//UserAddUserToGroup adds addee to group iff adder is in group and group is not a university network (we don't want people to be able to get into universities they're not part of)
//TODO: Check addee exists
func (api *API) UserAddUserToGroup(adder, addee gp.UserId, group gp.NetworkId) (err error) {
	in, neterr := api.UserInNetwork(adder, group)
	isgroup, grouperr := api.isGroup(group)
	switch {
	case neterr != nil:
		return neterr
	case grouperr != nil:
		return grouperr
	case !in || !isgroup:
		return &ENOTALLOWED
	default:
		err = api.setNetwork(addee, group)
		if err == nil {
			e := api.createNotification("added_group", adder, addee, uint64(group))
			if e != nil {
				log.Println("Error creating notification:", e)
			}
		}
		return
	}
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

func (api *API) UserGetNetwork(userId gp.UserId, netId gp.NetworkId) (network gp.Network, err error) {
	in, err := api.UserInNetwork(userId, netId)
	switch {
	case err != nil:
		return
	case !in:
		return network, &ENOTALLOWED
	default:
		return api.getNetwork(netId)
	}
}

func (api *API) getNetwork(netId gp.NetworkId) (network gp.Network, err error) {
	return api.db.GetNetwork(netId)
}

//CreateGroup creates a group and adds the creator as a member.
func (api *API) CreateGroup(userId gp.UserId, name string) (network gp.Network, err error) {
	network, err = api.db.CreateNetwork(name, true)
	if err != nil {
		return
	}
	err = api.db.SetNetwork(userId, network.Id)
	return
}

//HaveSharedNetwork returns true if both users a and b are in the same network.
func (api *API) HaveSharedNetwork(a gp.UserId, b gp.UserId) (shared bool, err error) {
	anets, err := api.GetUserNetworks(a)
	if err != nil {
		return
	}
	bnets, err := api.GetUserNetworks(b)
	if err != nil {
		return
	}
	for _, an := range anets {
		for _, bn := range bnets {
			if an.Id == bn.Id {
				return true, nil
			}
		}
	}
	return false, nil
}

//UserGetGroupMembers returns all the users in the group, or ENOTALLOWED if the user isn't in that group.
func (api *API) UserGetGroupMembers(userId gp.UserId, netId gp.NetworkId) (users []gp.User, err error) {
	in, errin := api.UserInNetwork(userId, netId)
	group, errgroup := api.isGroup(netId)
	switch {
	case errin != nil:
		return users, errin
	case errgroup != nil:
		return users, errgroup
	case !in || !group:
		return users, &ENOTALLOWED
	default:
		return api.db.GetNetworkUsers(netId)
	}
}

//UserLeaveGroup removes userId from group netId. If attempted on an official group it will give ENOTALLOWED (you can't leave your university...) but otherwise should always succeed.
func (api *API) UserLeaveGroup(userId gp.UserId, netId gp.NetworkId) (err error) {
	group, err := api.isGroup(netId)
	switch {
	case err != nil:
		return
	case !group:
		return &ENOTALLOWED
	default:
		return api.db.LeaveNetwork(userId, netId)
	}
}
