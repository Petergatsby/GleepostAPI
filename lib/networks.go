package lib

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"log"
	"strings"
)

func (api *API) GetUserNetworks(id gp.UserId) (nets []gp.Group, err error) {
	nets, err = api.db.GetUserNetworks(id, false)
	if err != nil {
		return
	}
	if len(nets) == 0 {
		return nets, gp.APIerror{"User has no networks!"}
	}
	api.cache.SetUserNetworks(id, nets...)
	return
}

//GetUserGroups is the same as GetUserNetworks, except it omits "official" networks (ie, universities)
func (api *API) GetUserGroups(id gp.UserId) (groups []gp.Group, err error) {
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

//UserAddUsersToGroup adds all addees to the group until the first error.
func (api *API) UserAddUsersToGroup(adder gp.UserId, addees []gp.UserId, group gp.NetworkId) (count int, err error) {
	for _, addee := range addees {
		err = api.UserAddUserToGroup(adder, addee, group)
		if err == nil {
			count++
		} else {
			return
		}
	}
	return
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

func (api *API) UserGetNetwork(userId gp.UserId, netId gp.NetworkId) (network gp.Group, err error) {
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

func (api *API) getNetwork(netId gp.NetworkId) (network gp.Group, err error) {
	return api.db.GetNetwork(netId)
}

//CreateGroup creates a group and adds the creator as a member.
func (api *API) CreateGroup(userId gp.UserId, name, url, desc string) (network gp.Group, err error) {
	network, err = api.db.CreateNetwork(name, url, desc, userId, true)
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

func (api *API) UserInviteEmail(userId gp.UserId, netId gp.NetworkId, email string) (err error) {
	in, neterr := api.UserInNetwork(userId, netId)
	isgroup, grouperr := api.isGroup(netId)
	switch {
	case neterr != nil:
		return neterr
	case grouperr != nil:
		return grouperr
	case !in || !isgroup:
		return &ENOTALLOWED
	default:
		//If the user already exists, add them straight into the group and don't email them.
		invitee, e := api.UserWithEmail(email)
		if e == nil {
			return api.setNetwork(invitee, netId)
		}
		token, e := RandomString()
		if e != nil {
			return e
		}
		err = api.db.CreateInvite(userId, netId, email, token)
		if err == nil {
			var from gp.User
			from, err = api.GetUser(userId)
			if err != nil {
				return
			}
			var group gp.Group
			group, err = api.getNetwork(netId)
			if err != nil {
				return
			}
			go api.issueInviteEmail(email, from, group, token)
		}
		return
	}
}

func (api *API) UserIsNetworkOwner(userId gp.UserId, netId gp.NetworkId) (owner bool, err error) {
	creator, err := api.db.NetworkCreator(netId)
	return (creator == userId), err
}

//UserSetNetworkImage sets the network's cover image to url, if userId is allowed to do so (currently, if they are the group's creator) or returns ENOTALLOWED otherwise.
func (api *API) UserSetNetworkImage(userId gp.UserId, netId gp.NetworkId, url string) (err error) {
	exists, eupload := api.UserUploadExists(userId, url)
	owner, eowner := api.UserIsNetworkOwner(userId, netId)
	switch {
	case eowner != nil:
		return eowner
	case eupload != nil:
		return eupload
	case !owner:
		return &ENOTALLOWED
	case !exists:
		//TODO: Return a different error
		return &ENOTALLOWED
	default:
		return api.db.SetNetworkImage(netId, url)
	}
}

func (api *API) InviteExists(email, invite string) (exists bool, err error) {
	return api.db.InviteExists(email, invite)
}

func (api *API) AssignNetworksFromInvites(user gp.UserId, email string) (err error) {
	return api.db.AssignNetworksFromInvites(user, email)
}

func (api *API) AcceptAllInvites(email string) (err error) {
	return api.db.AcceptAllInvites(email)
}
