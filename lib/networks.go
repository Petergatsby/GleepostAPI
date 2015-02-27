package lib

import (
	"log"
	"strings"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//ENoRole is given when you try to specify a role which doesn't exist.
var ENoRole = gp.APIerror{Reason: "Invalid role"}

var levels = map[string]int{
	"creator":       9,
	"administrator": 8,
	"member":        1,
}

//GetUserNetworks returns all networks this user belongs to, or an error if zhe belongs to none.
func (api *API) getUserNetworks(id gp.UserID) (nets []gp.GroupMembership, err error) {
	nets, err = api.db.GetUserNetworks(id, false)
	if err != nil {
		return
	}
	if len(nets) == 0 {
		return nets, gp.APIerror{Reason: "User has no networks!"}
	}
	return
}

//UserGetUserGroups is the same as GetUserNetworks, except it omits "official" networks (ie, universities)
func (api *API) UserGetUserGroups(perspective, user gp.UserID) (groups []gp.GroupMembership, err error) {
	groups = make([]gp.GroupMembership, 0)
	switch {
	case perspective == user:
		groups, err = api.db.GetUserNetworks(user, true)
		return
	default:
		shared, err := api.haveSharedNetwork(perspective, user)
		switch {
		case err != nil:
			return groups, err
		case !shared:
			return groups, &ENOTALLOWED
		default:
			return api.db.SubjectiveMemberships(perspective, user)
		}
	}
}

//userInNetwork returns true if user id is a member of network, false if not and err when there's a db problem.
func (api *API) userInNetwork(id gp.UserID, network gp.NetworkID) (in bool, err error) {
	networks, err := api.db.GetUserNetworks(id, false)
	if err != nil {
		return false, err
	}
	for _, n := range networks {
		if n.ID == network {
			return true, nil
		}
	}
	return false, nil
}

//isGroup returns false if this network isn't a group (ie isn't user-created) and error if the group doesn't exist.
func (api *API) isGroup(netID gp.NetworkID) (group bool, err error) {
	return api.db.IsGroup(netID)
}

//UserAddUsersToGroup adds all addees to the group until the first error.
func (api *API) UserAddUsersToGroup(adder gp.UserID, addees []gp.UserID, group gp.NetworkID) (count int, err error) {
	for _, addee := range addees {
		if adder == addee {
			err = api.UserJoinGroup(adder, group)
		} else {
			err = api.UserAddUserToGroup(adder, addee, group)
		}
		if err == nil {
			count++
		} else {
			return
		}
	}
	return
}

//UserChangeRole marks recipient with a new role in this network, if actor is allowed to give it. Valid roles are currently "member" or "administrator".
func (api *API) UserChangeRole(actor, recipient gp.UserID, network gp.NetworkID, role string) (err error) {
	lev, ok := levels[role]
	if !ok {
		return ENoRole
	}
	//To start with, for simplicity: You can only add/remove roles less / equal to your own.
	has, err := api.userHasRole(actor, network, role)
	switch {
	case err != nil:
		return
	case !has:
		return &ENOTALLOWED
	default:
		otherRole, err := api.userRole(recipient, network)
		myRole, err2 := api.userRole(actor, network)
		switch {
		case err != nil || err2 != nil:
			return &ENOTALLOWED
		//You can't change the role of someone higher-level than you.
		case otherRole.Level > myRole.Level:
			return &ENOTALLOWED
		default:
			return api.db.UserSetRole(recipient, network, gp.Role{Name: role, Level: lev})
		}
	}
}

//UserHasRole returns true if this user has at least this role (or greater) in this group.
func (api *API) userHasRole(user gp.UserID, network gp.NetworkID, roleName string) (has bool, err error) {
	lev, ok := levels[roleName]
	if !ok {
		return false, ENoRole
	}
	role, err := api.userRole(user, network)
	if err != nil {
		if err == gp.ENOSUCHUSER {
			err = nil
			return
		}
		return
	}
	if role.Level < lev {
		return false, nil
	}
	return true, nil
}

func (api *API) userRole(user gp.UserID, network gp.NetworkID) (role gp.Role, err error) {
	return api.db.UserRole(user, network)
}

//UserAddUserToGroup adds addee to group iff adder is in group and group is not a university network (we don't want people to be able to get into universities they're not part of)
//TODO: Check addee exists
//TODO: Suppress re-add push notification.
func (api *API) UserAddUserToGroup(adder, addee gp.UserID, group gp.NetworkID) (err error) {
	in, neterr := api.userInNetwork(adder, group)
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
			e := api.createNotification("added_group", adder, addee, 0, group, "")
			if e != nil {
				log.Println("Error creating notification:", e)
			}
			e = api.groupAddConvParticipants(adder, addee, group)
			if e != nil {
				log.Println("Error adding new group members to conversation:", e)
			}
		}
		return
	}
}

//UserJoinGroup makes this user a member of the group iff the group's privacy is "public" and the group is visible to them (ie, within their university network)
func (api *API) UserJoinGroup(userID gp.UserID, group gp.NetworkID) (err error) {
	canJoin, err := api.userCanJoin(userID, group)
	switch {
	case err != nil:
		return
	case canJoin:
		err = api.setNetwork(userID, group)
		if err != nil {
			return
		}
		err = api.joinGroupConversation(userID, group)
		return
	default:
		return &ENOTALLOWED
	}
}

func (api *API) joinGroupConversation(userID gp.UserID, group gp.NetworkID) (err error) {
	convID, err := api.db.GroupConversation(group)
	if err != nil {
		return
	}
	err = api.db.AddConversationParticipants(userID, []gp.UserID{userID}, convID)
	if err != nil {
		return
	}
	conv, err := api.GetConversation(userID, convID)
	if err != nil {
		return
	}
	go api.conversationChangedEvent(conv.Conversation)
	api.addSystemMessage(convID, userID, "JOINED")
	return
}

func (api *API) groupAddConvParticipants(adder, addee gp.UserID, group gp.NetworkID) (err error) {
	conv, err := api.db.GroupConversation(group)
	if err != nil {
		return
	}
	_, err = api.UserAddParticipants(adder, conv, addee)
	return
}

//UserCanJoin returns true if the user is allowed to unilaterally join this network (ie, it is both "public" and a sub-network of one this user already belongs to.)
func (api *API) userCanJoin(userID gp.UserID, netID gp.NetworkID) (public bool, err error) {
	net, err := api.getNetwork(netID)
	if err != nil {
		return
	}
	parent, err := api.db.NetworkParent(netID)
	if err != nil {
		return
	}
	in, err := api.userInNetwork(userID, parent)
	if err != nil {
		return
	}
	if net.Privacy == "public" && in {
		return true, nil
	}
	return false, nil
}

func (api *API) setNetwork(userID gp.UserID, netID gp.NetworkID) (err error) {
	return api.db.SetNetwork(userID, netID)
}

func (api *API) assignNetworks(user gp.UserID, email string) (networks int, err error) {
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

//UserGetNetwork returns the information about a network, if userID is a member of it; ENOTALLOWED otherwise.
func (api *API) UserGetNetwork(userID gp.UserID, netID gp.NetworkID) (network gp.GroupMembership, err error) {
	in, err := api.userInNetwork(userID, netID)
	switch {
	case err != nil:
		return
	case !in:
		return network, &ENOTALLOWED
	default:
		network.Group, err = api.getNetwork(netID)
		if err != nil {
			return
		}
		network.UnreadCount, err = api.db.UserConversationUnread(userID, network.Conversation)
		if err != nil {
			log.Println(err)
			return
		}
		network.Role, err = api.db.UserRole(userID, netID)
		if err != nil {
			log.Println(err)
		}
		return
	}
}

func (api *API) getNetwork(netID gp.NetworkID) (network gp.Group, err error) {
	return api.db.GetNetwork(netID)
}

//CreateGroup creates a group and adds the creator as a member.
func (api *API) CreateGroup(userID gp.UserID, name, url, desc, privacy string) (network gp.Group, err error) {
	exists, eupload := api.userUploadExists(userID, url)
	switch {
	case eupload != nil:
		return network, eupload
	case !exists && len(url) > 0:
		return network, &ENOTALLOWED
	default:
		var primary gp.GroupMembership
		primary, err = api.db.GetUserUniversity(userID)
		if err != nil {
			return
		}
		network, err = api.db.CreateNetwork(name, primary.ID, url, desc, userID, true, privacy)
		if err != nil {
			return
		}
		err = api.db.SetNetwork(userID, network.ID)
		if err != nil {
			return
		}
		err = api.db.UserSetRole(userID, network.ID, gp.Role{Name: "creator", Level: 9})
		if err != nil {
			return
		}
		var user gp.User
		user, err = api.getUser(userID)
		if err != nil {
			return
		}
		var conversation gp.Conversation
		conversation, err = api.CreateConversation(userID, []gp.User{user}, false, network.ID)
		if err == nil {
			network.Conversation = conversation.ID
		} else {
			log.Println(err)
		}
		return
	}
}

//HaveSharedNetwork returns true if both users a and b are in the same network.
func (api *API) haveSharedNetwork(a gp.UserID, b gp.UserID) (shared bool, err error) {
	anets, err := api.getUserNetworks(a)
	if err != nil {
		return
	}
	bnets, err := api.getUserNetworks(b)
	if err != nil {
		return
	}
	for _, an := range anets {
		for _, bn := range bnets {
			if an.ID == bn.ID {
				return true, nil
			}
		}
	}
	return false, nil
}

//UserGetGroupAdmins returns all the admins of the group, or ENOTALLOWED if the requesting user isn't in that group.
func (api *API) UserGetGroupAdmins(userID gp.UserID, netID gp.NetworkID) (users []gp.UserRole, err error) {
	users = make([]gp.UserRole, 0)
	in, errin := api.userInNetwork(userID, netID)
	group, errgroup := api.isGroup(netID)
	switch {
	case errin != nil:
		return users, errin
	case errgroup != nil:
		return users, errgroup
	case !in || !group:
		return users, &ENOTALLOWED
	default:
		return api.db.GetNetworkAdmins(netID)
	}
}

//UserGetGroupMembers returns all the users in the group, or ENOTALLOWED if the user isn't in that group.
func (api *API) UserGetGroupMembers(userID gp.UserID, netID gp.NetworkID) (users []gp.UserRole, err error) {
	users = make([]gp.UserRole, 0)
	in, errin := api.userInNetwork(userID, netID)
	group, errgroup := api.isGroup(netID)
	CanJoin, errJoin := api.userCanJoin(userID, netID)
	switch {
	case errJoin == nil && CanJoin:
		return api.db.GetNetworkUsers(netID)
	case errin != nil:
		return users, errin
	case errgroup != nil:
		return users, errgroup
	case !in || !group:
		return users, &ENOTALLOWED
	default:
		return api.db.GetNetworkUsers(netID)
	}
}

//UserLeaveGroup removes userId from group netId. If attempted on an official group it will give ENOTALLOWED (you can't leave your university...) but otherwise should always succeed.
func (api *API) UserLeaveGroup(userID gp.UserID, netID gp.NetworkID) (err error) {
	group, err := api.isGroup(netID)
	switch {
	case err != nil:
		return
	case !group:
		return &ENOTALLOWED
	default:
		err = api.db.LeaveNetwork(userID, netID)
		if err == nil {
			convID, e := api.db.GroupConversation(netID)
			if e != nil {
				log.Println(e)
				return
			}
			go api.UserDeleteConversation(userID, convID)
		}
		return
	}
}

//UserInviteEmail sends a group invite from userID to email, or err if something went wrong.
//If someone has already signed up with email, it just adds them to the group directly.
func (api *API) UserInviteEmail(userID gp.UserID, netID gp.NetworkID, email string) (err error) {
	in, neterr := api.userInNetwork(userID, netID)
	isgroup, grouperr := api.isGroup(netID)
	switch {
	case neterr != nil:
		return neterr
	case grouperr != nil:
		return grouperr
	case !in || !isgroup:
		return &ENOTALLOWED
	default:
		//If the user already exists, add them straight into the group and don't email them.
		invitee, e := api.userWithEmail(email)
		if e == nil {
			return api.setNetwork(invitee, netID)
		}
		token, e := randomString()
		if e != nil {
			return e
		}
		err = api.db.CreateInvite(userID, netID, email, token)
		if err == nil {
			var from gp.User
			from, err = api.getUser(userID)
			if err != nil {
				return
			}
			var group gp.Group
			group, err = api.getNetwork(netID)
			if err != nil {
				return
			}
			go api.issueInviteEmail(email, from, group, token)
		}
		return
	}
}

//UserIsNetworkOwner returns true if userID created netID, and err if the database is down.
func (api *API) userIsNetworkOwner(userID gp.UserID, netID gp.NetworkID) (owner bool, err error) {
	creator, err := api.db.NetworkCreator(netID)
	return (creator == userID), err
}

//UserSetNetworkImage sets the network's cover image to url, if userId is allowed to do so (currently, if they are the group's creator) or returns ENOTALLOWED otherwise.
func (api *API) UserSetNetworkImage(userID gp.UserID, netID gp.NetworkID, url string) (err error) {
	exists, eupload := api.userUploadExists(userID, url)
	owner, eowner := api.userIsNetworkOwner(userID, netID)
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
		return api.db.SetNetworkImage(netID, url)
	}
}

//InviteExists returns true if the email:invite pair is valid or err if the db is down.
func (api *API) inviteExists(email, invite string) (exists bool, err error) {
	return api.db.InviteExists(email, invite)
}

//AssignNetworksFromInvites finds all invites for this email address and resolves them (adds user to the groups involved)
func (api *API) assignNetworksFromInvites(user gp.UserID, email string) (err error) {
	return api.db.AssignNetworksFromInvites(user, email)
}

//AssignNetworksFromFBInvites does the same as AssignNetworksFromInvites, but for a given facebook user id.
func (api *API) assignNetworksFromFBInvites(user gp.UserID, facebook uint64) (err error) {
	return api.db.AssignNetworksFromFBInvites(user, facebook)
}

//AcceptAllInvites sets all invites to this email as "accepted" (they should not be valid any more)
func (api *API) acceptAllInvites(email string) (err error) {
	return api.db.AcceptAllInvites(email)
}

//AcceptAllFBInvites does the same as AcceptAllInvites, but for a facebook user.
func (api *API) acceptAllFBInvites(facebook uint64) (err error) {
	return api.db.AcceptAllFBInvites(facebook)
}
