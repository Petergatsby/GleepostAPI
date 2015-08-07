package lib

import (
	"database/sql"
	"log"
	"strings"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/psc"
	"github.com/go-sql-driver/mysql"
)

//ENoRole is given when you try to specify a role which doesn't exist.
var ENoRole = gp.APIerror{Reason: "Invalid role"}

//NobodyAdded is returned when you call UserAddToGroup with no arguments.
var NobodyAdded = gp.APIerror{Reason: "Must add either user(s), facebook user(s) or an email"}

//NoSuchNetwork occurs when performing an action against a network which doesn't exist (or the user can't see).
var NoSuchNetwork = gp.APIerror{Reason: "No such network"}

//NoSuchRequest occurs when attempting to reject a non-existent request to join a network.
var NoSuchRequest = gp.APIerror{Reason: "No such request"}

//AlreadyRejected occurs when attempting to reject a group-join request which is already rejected.
var AlreadyRejected = gp.APIerror{Reason: "Request is already rejected"}

//AlreadyAccepted occurs when attempting to reject a group-join request which is already accepted.
var AlreadyAccepted = gp.APIerror{Reason: "Request is already accepted"}

var levels = map[string]int{
	"creator":       9,
	"administrator": 8,
	"member":        1,
}

//UserGetUserGroups is the same as GetUserNetworks, except it omits "official" networks (ie, universities)
func (api *API) UserGetUserGroups(perspective, user gp.UserID, index int64) (groups []gp.GroupSubjective, err error) {
	groups = make([]gp.GroupSubjective, 0)
	switch {
	case perspective == user:
		groups, err = api.groupsByActivity(user, index, api.Config.GroupPageSize)
		return
	default:
		shared, err := api.sameUniversity(perspective, user)
		switch {
		case err != nil:
			return groups, err
		case !shared:
			return groups, &ENOTALLOWED
		default:
			return api.subjectiveMemberships(perspective, user, index, api.Config.GroupPageSize)
		}
	}
}

//UserAddToGroup adds these gleepost users to this group (if you're allowed) and invites the rest via facebook / email.
func (api *API) UserAddToGroup(adder gp.UserID, group gp.NetworkID, addees []gp.UserID, fbinvites []uint64, emailInvites []string) (err error) {
	added := false
	if len(addees) > 0 {
		added = true
		_, err = api.UserAddUsersToGroup(adder, addees, group)
		if err != nil {
			return
		}
	}
	if len(fbinvites) > 0 {
		added = true
		_, err = api.UserAddFBUsersToGroup(adder, fbinvites, group)
		if err != nil {
			return
		}
	}
	if len(emailInvites) > 0 {
		added = true
		for _, email := range emailInvites {
			err = api.UserInviteEmail(adder, group, email)
			if err != nil {
				return
			}
		}
	}
	if !added {
		return NobodyAdded
	}
	return nil
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
			return api.userSetRole(recipient, network, gp.Role{Name: role, Level: lev})
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
			api.notifObserver.Notify(addedGroupEvent{userID: adder, addeeID: addee, netID: group})
			e := api.groupAddConvParticipants(adder, addee, group)
			if e != nil {
				log.Println("Error adding new group members to conversation:", e)
			}
			err = api.setRequestStatus(addee, group, "accepted", adder)
			if err != nil {
				log.Println(err)
				return
			}
			api.esIndexGroup(group)
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

		api.esIndexGroup(group)
		return
	default:
		return &ENOTALLOWED
	}
}

func (api *API) joinGroupConversation(userID gp.UserID, group gp.NetworkID) (err error) {
	convID, err := api.groupConversation(group)
	if err != nil {
		return
	}
	err = api.addConversationParticipants(userID, []gp.UserID{userID}, convID)
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
	conv, err := api.groupConversation(group)
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
	parent, err := api.networkParent(netID)
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

func (api *API) assignNetworks(user gp.UserID, email string) (networks int, err error) {
	rules, e := api.getRules()
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
	return
}

//UserGetNetwork returns the information about a network, if userID is a member of it; ENOTALLOWED otherwise.
func (api *API) UserGetNetwork(userID gp.UserID, netID gp.NetworkID) (network gp.GroupSubjective, err error) {
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
		network.UnreadCount, err = api.userConversationUnread(userID, network.Conversation)
		if err != nil {
			log.Println(err)
			return
		}
		var role gp.Role
		role, err = api.userRole(userID, netID)
		if err == nil {
			network.YourRole = &role
			//LastActivity
			var lastActivity time.Time
			lastActivity, err = api.networkLastActivity(userID, netID)
			if err != nil {
				log.Println(err)
				return
			} else {
				network.LastActivity = &lastActivity
			}
			network.NewPosts, err = api.groupNewPosts(userID, netID)
			if err != nil {
				return
			}
		} else {
			status, err := api.pendingRequestExists(userID, netID)
			if err == nil && (status == "pending" || status == "rejected") {
				network.PendingRequest = true
			}
		}
		return network, nil
	}
}

//CreateGroup creates a group and adds the creator as a member.
func (api *API) CreateGroup(userID gp.UserID, name, url, desc, privacy, category string) (network gp.Group, err error) {
	exists, eupload := api.userUploadExists(userID, url)
	switch {
	case eupload != nil:
		return network, eupload
	case !exists && len(url) > 0:
		return network, &ENOTALLOWED
	default:
		var primary gp.GroupSubjective
		primary, err = api.getUserUniversity(userID)
		if err != nil {
			return
		}
		privacy = strings.ToLower(privacy)
		if privacy != "public" && privacy != "private" && privacy != "secret" {
			privacy = "private"
		}
		network, err = api.createNetwork(name, primary.ID, url, desc, userID, true, privacy, category)
		if err != nil {
			return
		}
		err = api.setNetwork(userID, network.ID)
		if err != nil {
			return
		}
		err = api.userSetRole(userID, network.ID, gp.Role{Name: "creator", Level: 9})
		if err != nil {
			return
		}
		var user gp.User
		user, err = api.users.byID(userID)
		if err != nil {
			return
		}
		var conversation gp.Conversation
		conversation, err = api.createConversation(userID, []gp.User{user}, false, network.ID)
		if err == nil {
			network.Conversation = conversation.ID
		} else {
			log.Println(err)
		}
		api.esIndexGroup(network.ID)
		return
	}
}

//sameUniversity returns true if both users a and b are in the same university.
func (api *API) sameUniversity(a, b gp.UserID) (shared bool, err error) {
	unia, err := api.getUserUniversity(a)
	if err != nil {
		return
	}
	unib, err := api.getUserUniversity(b)
	if err != nil {
		return
	}
	return unia.ID == unib.ID, nil
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
		return api.nm.getNetworkAdmins(netID)
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
		return getNetworkUsers(api.sc, netID)
	case errin != nil:
		return users, errin
	case errgroup != nil:
		return users, errgroup
	case !in || !group:
		return users, &ENOTALLOWED
	default:
		return getNetworkUsers(api.sc, netID)
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
		err = api.leaveNetwork(userID, netID)
		if err == nil {
			convID, e := api.groupConversation(netID)
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
		err = api.createInvite(userID, netID, email, token)
		if err != nil {
			return
		}
		go api.issueInviteEmail(email, userID, netID, token)
		return
	}
}

//UserIsNetworkOwner returns true if userID created netID, and err if the database is down.
func (api *API) userIsNetworkOwner(userID gp.UserID, netID gp.NetworkID) (owner bool, err error) {
	creator, err := api.nm.networkCreator(netID)
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
		return api.setNetworkImage(netID, url)
	}
}

//AdminCreateUniversity creates a new university with this name, accepting users registered with emails in these domains.
func (api *API) AdminCreateUniversity(userID gp.UserID, name string, domains ...string) (university gp.Network, err error) {
	admin := api.isAdmin(userID)
	if !admin {
		err = ENOTALLOWED
		return
	}
	university, err = api.createUniversity(name)
	if err != nil {
		return
	}
	err = api.addNetworkRules(university.ID, domains...)
	return
}

//GetUserUniversity returns this user's primary network (ie, their university)
func (api *API) getUserUniversity(id gp.UserID) (network gp.GroupSubjective, err error) {
	s, err := api.sc.Prepare("SELECT user_network.network_id, network.name, user_network.role, user_network.role_level, network.cover_img, network.`desc`, network.creator, network.privacy FROM user_network JOIN network ON user_network.network_id = network.id WHERE user_network.user_id = ? AND network.is_university = 1 ")
	if err != nil {
		return
	}
	var img, desc, privacy sql.NullString
	var creator sql.NullInt64
	var role gp.Role
	err = s.QueryRow(id).Scan(&network.ID, &network.Group.Network.Name, &role.Name, &role.Level, &img, &desc, &creator, &privacy)
	if img.Valid {
		network.Image = img.String
	}
	if desc.Valid {
		network.Desc = desc.String
	}
	if creator.Valid {
		u, err := api.users.byID(gp.UserID(creator.Int64))
		if err == nil {
			network.Creator = &u
		}
		network.MemberCount, _ = api.groupMemberCount(network.ID)
		//TODO(patrick) - maybe don't display group conversation id if you're not a member.
		network.Conversation, _ = api.groupConversation(network.ID)
		network.UnreadCount, _ = api.userConversationUnread(id, network.Conversation)
	}
	if privacy.Valid {
		network.Privacy = privacy.String
	}
	network.TheirRole = &role
	return
}

//MasterGroup returns the id of the group which administrates this network, or NoSuchGroup if there is none.
func (api *API) masterGroup(netID gp.NetworkID) (master gp.NetworkID, err error) {
	q := "SELECT master_group FROM network WHERE id = ? AND master_group IS NOT NULL"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(netID).Scan(&master)
	if err == sql.ErrNoRows {
		err = NoSuchGroup
	}
	return
}

//GetRules returns all the network matching rules for every network.
func (api *API) getRules() (rules []gp.Rule, err error) {
	defer api.Statsd.Time(time.Now(), "gleepost.getRules.db")
	ruleSelect := "SELECT network_id, rule_type, rule_value FROM net_rules"
	s, err := api.sc.Prepare(ruleSelect)
	if err != nil {
		return
	}
	rows, err := s.Query()
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var rule gp.Rule
		if err = rows.Scan(&rule.NetworkID, &rule.Type, &rule.Value); err != nil {
			return
		}
		rules = append(rules, rule)
	}
	return
}

//groupsByActivity returns all the networks id is a member of, optionally only returning user-created networks.
func (api *API) groupsByActivity(id gp.UserID, index int64, count int) (networks []gp.GroupSubjective, err error) {
	defer api.Statsd.Time(time.Now(), "gleepost.networks.byUser.db")
	networks = make([]gp.GroupSubjective, 0)
	networkSelect := "SELECT user_network.network_id, user_network.role, " +
		"user_network.role_level, network.name, " +
		"network.cover_img, network.`desc`, network.creator, network.privacy, " +
		"GREATEST( " +
		"COALESCE((SELECT MAX(`timestamp`) FROM chat_messages WHERE conversation_id = conversations.id), '0000-00-00 00:00:00'), " +
		"COALESCE((SELECT MAX(`time`) FROM wall_posts WHERE network_id = user_network.network_id), '0000-00-00 00:00:00') " +
		") AS last_activity " +
		"FROM user_network " +
		"INNER JOIN network ON user_network.network_id = network.id " +
		"JOIN conversations ON conversations.group_id = network.id " +
		"WHERE user_id = ? " +
		"AND network.user_group = 1 " +
		"ORDER BY last_activity DESC LIMIT ?, ?"
	s, err := api.sc.Prepare(networkSelect)
	if err != nil {
		return
	}
	rows, err := s.Query(id, index, count)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var network gp.GroupSubjective
		var img, desc sql.NullString
		var creator sql.NullInt64
		var privacy sql.NullString
		var lastActivity string
		var role gp.Role
		err = rows.Scan(&network.ID, &role.Name, &role.Level, &network.Group.Name, &img, &desc, &creator, &privacy, &lastActivity)
		if err != nil {
			return
		}
		t, e := time.Parse(mysqlTime, lastActivity)
		if e == nil {
			network.LastActivity = &t
		}
		if img.Valid {
			network.Image = img.String
		}
		if desc.Valid {
			network.Desc = desc.String
		}
		if creator.Valid {
			u, err := api.users.byID(gp.UserID(creator.Int64))
			if err == nil {
				network.Creator = &u
			}
			network.MemberCount, _ = api.groupMemberCount(network.ID)
			network.Conversation, _ = api.groupConversation(network.ID)
			network.UnreadCount, _ = api.userConversationUnread(id, network.Conversation)
			network.NewPosts, err = api.groupNewPosts(id, network.ID)
			if err != nil {
				log.Println(err)
			}
			status, err := api.pendingRequestExists(id, network.ID)
			if err == nil && (status == "pending" || status == "rejected") {
				network.PendingRequest = true
			}
		}
		if privacy.Valid {
			network.Privacy = privacy.String
		}
		network.YourRole = &role
		networks = append(networks, network)
	}
	return
}

func (api *API) networkLastActivity(perspective gp.UserID, netID gp.NetworkID) (lastActivity time.Time, err error) {
	q := "SELECT " +
		"GREATEST( " +
		"COALESCE((SELECT MAX(`timestamp`) FROM chat_messages WHERE conversation_id = conversations.id), '0000-00-00 00:00:00'), " +
		"COALESCE((SELECT MAX(`time`) FROM wall_posts WHERE network_id = ?), '0000-00-00 00:00:00') " +
		") AS last_activity " +
		"FROM network " +
		"JOIN conversations ON conversations.group_id = network.id " +
		"WHERE network.id = ? "
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	var _time string
	err = s.QueryRow(netID, netID).Scan(&_time)
	if err != nil {
		return
	}
	lastActivity, err = time.Parse(mysqlTime, _time)
	return
}

func (api *API) groupNewPosts(userID gp.UserID, groupID gp.NetworkID) (count int, err error) {
	q := "SELECT COUNT(DISTINCT id) FROM wall_posts " +
		"WHERE wall_posts.network_id = ? " +
		"AND wall_posts.id > " +
		"(SELECT COALESCE(MAX(post_views.post_id), 0) FROM post_views " +
		"JOIN wall_posts ON post_views.post_id = wall_posts.id " +
		"WHERE post_views.user_id = ? AND wall_posts.network_id = ?) "
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(groupID, userID, groupID).Scan(&count)
	return
}

//SubjectiveMembershipCount is the number of groups user belongs to, from the point of view of perspective.
//That is: the public / private groups they're a part of, plus the secret groups that perspective is also in.
func (api *API) subjectiveMembershipCount(perspective, user gp.UserID) (count int, err error) {
	q := "SELECT COUNT(*) FROM user_network JOIN network ON user_network.network_id = network.id "
	q += "WHERE user_group = 1 AND parent = (SELECT network_id FROM user_network WHERE user_id = ? LIMIT 1) "
	q += "AND (privacy != 'secret' OR network.id IN (SELECT network_id FROM user_network WHERE user_id = ?)) "
	q += "AND user_network.user_id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(perspective, perspective, user).Scan(&count)
	return

}

//SubjectiveMemberships returns all the groups this user is a member of, as far as perspective is concerned.
func (api *API) subjectiveMemberships(perspective, user gp.UserID, index int64, count int) (groups []gp.GroupSubjective, err error) {
	groups = make([]gp.GroupSubjective, 0)
	q := "SELECT user_network.network_id, user_network.role, user_network.role_level, network.name, network.cover_img, network.`desc`, network.creator, network.privacy FROM user_network JOIN network ON user_network.network_id = network.id "
	q += "WHERE user_group = 1 AND parent = (SELECT network_id FROM user_network WHERE user_id = ? LIMIT 1) "
	q += "AND (privacy != 'secret' OR network.id IN (SELECT network_id FROM user_network WHERE user_id = ?)) "
	q += "AND user_network.user_id = ? "
	q += "LIMIT ?, ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(perspective, perspective, user, index, count)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var network gp.GroupSubjective
		var img, desc sql.NullString
		var creator sql.NullInt64
		var privacy sql.NullString
		var role gp.Role
		err = rows.Scan(&network.ID, &role.Name, &role.Level, &network.Group.Name, &img, &desc, &creator, &privacy)
		if err != nil {
			return
		}
		network.TheirRole = &role
		if img.Valid {
			network.Image = img.String
		}
		if desc.Valid {
			network.Desc = desc.String
		}
		if creator.Valid {
			u, err := api.users.byID(gp.UserID(creator.Int64))
			if err == nil {
				network.Creator = &u
			}
			network.MemberCount, _ = api.groupMemberCount(network.ID)
		}
		if privacy.Valid {
			network.Privacy = privacy.String
		}
		var yourRole gp.Role
		yourRole, err = api.userRole(perspective, network.ID)
		if err == nil {
			network.YourRole = &yourRole
			var lastActivity time.Time
			lastActivity, err = api.networkLastActivity(perspective, network.ID)
			if err == nil {
				network.LastActivity = &lastActivity
			}
			network.NewPosts, err = api.groupNewPosts(perspective, network.ID)
			if err != nil {
				return
			}
			network.Conversation, _ = api.groupConversation(network.ID)
			network.UnreadCount, err = api.userConversationUnread(perspective, network.Conversation)
		}
		status, err := api.pendingRequestExists(perspective, network.ID)
		if err == nil && (status == "pending" || status == "rejected") {
			network.PendingRequest = true
		}
		groups = append(groups, network)
	}
	err = nil
	return
}

//SetNetwork idempotently makes userID a member of networkID
func (api *API) setNetwork(userID gp.UserID, networkID gp.NetworkID) (err error) {
	networkInsert := "REPLACE INTO user_network (user_id, network_id) VALUES (?, ?)"
	s, err := api.sc.Prepare(networkInsert)
	if err != nil {
		return
	}
	_, err = s.Exec(userID, networkID)
	return
}

//GetNetwork returns the network netId. If userID is 0, it will omit the group's unread count.
func (api *API) getNetwork(netID gp.NetworkID) (network gp.Group, err error) {
	networkSelect := "SELECT name, cover_img, `desc`, creator, user_group, privacy " +
		"FROM network " +
		"WHERE network.id = ?"
	s, err := api.sc.Prepare(networkSelect)
	if err != nil {
		return
	}
	var coverImg, desc, privacy sql.NullString
	var creator sql.NullInt64
	var userGroup bool
	err = s.QueryRow(netID).Scan(&network.Name, &coverImg, &desc, &creator, &userGroup, &privacy)
	if err != nil {
		return
	}
	network.ID = netID
	if coverImg.Valid {
		network.Image = coverImg.String
	}
	if desc.Valid {
		network.Desc = desc.String
	}
	if creator.Valid {
		u, err := api.users.byID(gp.UserID(creator.Int64))
		if err == nil {
			network.Creator = &u
		}
		network.MemberCount, _ = api.groupMemberCount(network.ID)
		network.Conversation, _ = api.groupConversation(network.ID)
	}
	if privacy.Valid {
		network.Privacy = privacy.String
	}
	return
}

//CreateNetwork creates a new network. usergroup indicates that the group is user-defined (created by a user rather than system-defined networks such as universities)
func (api *API) createNetwork(name string, parent gp.NetworkID, url, desc string, creator gp.UserID, usergroup bool, privacy, category string) (group gp.Group, err error) {
	networkInsert := "INSERT INTO network (name, parent, cover_img, `desc`, creator, user_group, privacy, category) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	s, err := api.sc.Prepare(networkInsert)
	if err != nil {
		return
	}
	res, err := s.Exec(name, parent, url, desc, creator, usergroup, privacy, category)
	if err != nil {
		return
	}
	id, _ := res.LastInsertId()
	group.ID = gp.NetworkID(id)
	group.Name = name
	group.Image = url
	group.Desc = desc
	group.Privacy = privacy
	group.MemberCount = 1
	group.Category = category
	u, err := api.users.byID(creator)
	if err == nil {
		group.Creator = &u
	} else {
		log.Println("Error getting user:", err)
	}
	return
}

//IsGroup returns false if netId isn't a user group, and ErrNoRows if netId doesn't exist.
func (api *API) isGroup(netID gp.NetworkID) (group bool, err error) {
	isgroup := "SELECT user_group FROM network WHERE id = ?"
	s, err := api.sc.Prepare(isgroup)
	if err != nil {
		return
	}
	err = s.QueryRow(netID).Scan(&group)
	return
}

//GetNetworkAdmins returns all the administrators of the group netID
func (nm *NetworkManager) getNetworkAdmins(netID gp.NetworkID) (users []gp.UserRole, err error) {
	users = make([]gp.UserRole, 0)
	memberQuery := "SELECT user_id, users.avatar, users.firstname, users.official, user_network.role, user_network.role_level FROM user_network JOIN users ON user_network.user_id = users.id WHERE user_network.network_id = ? AND user_network.role = 'administrator'"
	s, err := nm.sc.Prepare(memberQuery)
	if err != nil {
		return
	}
	rows, err := s.Query(netID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var user gp.UserRole
		var av sql.NullString
		err = rows.Scan(&user.ID, &av, &user.User.Name, &user.User.Official, &user.Role.Name, &user.Role.Level)
		if err != nil {
			return
		}
		if av.Valid {
			user.Avatar = av.String
		}
		users = append(users, user)
	}
	return
}

//GetNetworkUsers returns all the members of the group netId
func getNetworkUsers(sc *psc.StatementCache, netID gp.NetworkID) (users []gp.UserRole, err error) {
	users = make([]gp.UserRole, 0)
	memberQuery := "SELECT user_id, users.avatar, users.firstname, users.official, user_network.role, user_network.role_level FROM user_network JOIN users ON user_network.user_id = users.id WHERE user_network.network_id = ?"
	s, err := sc.Prepare(memberQuery)
	if err != nil {
		return
	}
	rows, err := s.Query(netID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var user gp.UserRole
		var av sql.NullString
		err = rows.Scan(&user.ID, &av, &user.User.Name, &user.User.Official, &user.Role.Name, &user.Role.Level)
		if err != nil {
			return
		}
		if av.Valid {
			user.Avatar = av.String
		}
		users = append(users, user)
	}
	return
}

//LeaveNetwork idempotently removes userID from the network netID.
func (api *API) leaveNetwork(userID gp.UserID, netID gp.NetworkID) (err error) {
	leaveQuery := "DELETE FROM user_network WHERE user_id = ? AND network_id = ?"
	s, err := api.sc.Prepare(leaveQuery)
	if err != nil {
		return
	}
	_, err = s.Exec(userID, netID)
	return
}

//CreateInvite stores an invite for a particular email to a particular network.
func (api *API) createInvite(userID gp.UserID, netID gp.NetworkID, email string, token string) (err error) {
	inviteQuery := "INSERT INTO group_invites (group_id, inviter, email, `key`) VALUES (?, ?, ?, ?)"
	s, err := api.sc.Prepare(inviteQuery)
	if err != nil {
		return
	}
	_, err = s.Exec(netID, userID, email, token)
	return
}

//SetNetworkImage updates a network's profile image.
func (api *API) setNetworkImage(netID gp.NetworkID, url string) (err error) {
	networkUpdate := "UPDATE network SET cover_img = ? WHERE id = ?"
	s, err := api.sc.Prepare(networkUpdate)
	if err != nil {
		return
	}
	_, err = s.Exec(url, netID)
	return
}

//NetworkCreator returns the user who created this network.
func (nm *NetworkManager) networkCreator(netID gp.NetworkID) (creator gp.UserID, err error) {
	qCreator := "SELECT creator FROM network WHERE id = ?"
	s, err := nm.sc.Prepare(qCreator)
	if err != nil {
		return
	}
	err = s.QueryRow(netID).Scan(&creator)
	return
}

//InviteExists returns true if there is a matching invite for email:invite (that's not already accepted)
func (api *API) inviteExists(email, invite string) (exists bool, err error) {
	q := "SELECT COUNT(*) FROM group_invites WHERE `email` = ? AND `key` = ? AND `accepted` = 0"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(email, invite).Scan(&exists)
	return
}

//AcceptAllInvites marks all invites as accepted for this email address.
func (api *API) acceptAllInvites(userID gp.UserID, email string) (err error) {
	q := "REPLACE INTO user_network (user_id, network_id) SELECT ?, group_id FROM group_invites WHERE email = ? AND accepted = 0"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(userID, email)
	if err != nil {
		return
	}
	q = "UPDATE group_invites SET accepted = 1 WHERE email = ?"
	s, err = api.sc.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(email)
	return
}

//AssignNetworksFromFBInvites adds user to all networks which this facebook id has been invited to.
//TODO: only do un-accepted invites (!)
func (api *API) assignNetworksFromFBInvites(user gp.UserID, facebook uint64) (err error) {
	q := "REPLACE INTO user_network (user_id, network_id) SELECT ?, network_id FROM fb_group_invites WHERE facebook_id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(user, facebook)
	return
}

//AcceptAllFBInvites marks all invites for this facebook user as accepted.
func (api *API) acceptAllFBInvites(facebook uint64) (err error) {
	q := "UPDATE fb_group_invites SET accepted = 1 WHERE facebook_id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(facebook)
	return
}

//UserAddFBUserToGroup records that this facebook user has been invited to netID.
func (api *API) userAddFBUserToGroup(user gp.UserID, fbuser uint64, netID gp.NetworkID) (err error) {
	q := "INSERT INTO fb_group_invites (inviter_user_id, facebook_id, network_id) VALUES (?, ?, ?)"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(user, fbuser, netID)
	return
}

//SetNetworkParent records that this network is a sub-network of parent (at the moment just used for visibility).
func (api *API) setNetworkParent(network, parent gp.NetworkID) (err error) {
	q := "UPDATE network SET parent = ? WHERE id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(parent, network)
	return
}

//NetworkParent returns the ID of this network's parent, or zero if it has none.
func (api *API) networkParent(netID gp.NetworkID) (parent gp.NetworkID, err error) {
	q := "SELECT parent FROM network WHERE id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(netID).Scan(&parent)
	return
}

//UserRole gives this user's role:level in this network, or ENOSUCHUSER if the user isn't part of the network.
func (api *API) userRole(user gp.UserID, network gp.NetworkID) (role gp.Role, err error) {
	q := "SELECT role, role_level FROM user_network WHERE user_id = ? AND network_id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(user, network).Scan(&role.Name, &role.Level)
	if err != nil && err == sql.ErrNoRows {
		err = gp.ENOSUCHUSER
	}
	return
}

//UserSetRole sets this user's Role within this network.
func (api *API) userSetRole(user gp.UserID, network gp.NetworkID, role gp.Role) (err error) {
	q := "UPDATE user_network SET role = ?, role_level = ? WHERE user_id = ? AND network_id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(role.Name, role.Level, user, network)
	return
}

//GroupMemberCount returns the number of members this group has.
func (api *API) groupMemberCount(network gp.NetworkID) (count int, err error) {
	q := "SELECT COUNT(*) FROM user_network WHERE network_id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(network).Scan(&count)
	return
}

//GroupConversation returns this group's conversation ID.
func (api *API) groupConversation(group gp.NetworkID) (conversation gp.ConversationID, err error) {
	q := "SELECT id FROM conversations WHERE group_id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(group).Scan(&conversation)
	return
}

//UserInNetwork returns true iff this user is in this network.
func (api *API) userInNetwork(userID gp.UserID, network gp.NetworkID) (in bool, err error) {
	s, err := api.sc.Prepare("SELECT COUNT(*) FROM user_network WHERE user_id = ? AND network_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(userID, network).Scan(&in)
	return
}

//CreateUniversity creates a new university network with this name.
func (api *API) createUniversity(name string) (network gp.Network, err error) {
	s, err := api.sc.Prepare("INSERT INTO network (name, is_university, user_group) VALUES (?, 1, 0)")
	if err != nil {
		return
	}
	res, err := s.Exec(name)
	if err != nil {
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		return
	}
	network.ID = gp.NetworkID(id)
	network.Name = name
	return
}

//AddNetworkRules adds filters to this network: people registering with emails in these domains will be automatically filtered into this network.
func (api *API) addNetworkRules(netID gp.NetworkID, domains ...string) (err error) {
	s, err := api.sc.Prepare("INSERT INTO net_rules (network_id, rule_type, rule_value) VALUES (?, 'email', ?)")
	if err != nil {
		return
	}
	for _, d := range domains {
		_, err = s.Exec(netID, d)
		if err != nil {
			return
		}
	}
	return nil
}

//NetworkDomain returns this network's domain.
func (api *API) networkDomain(netID gp.NetworkID) (domain string, err error) {
	s, err := api.sc.Prepare("SELECT rule_value FROM net_rules WHERE rule_type = 'email' AND network_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(netID).Scan(&domain)

	return
}

func groupName(sc *psc.StatementCache, group gp.NetworkID) (name string, err error) {
	s, err := sc.Prepare("SELECT name FROM network WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(group).Scan(&name)
	return
}

//Returns err == nil if this network is visible to this user
func (api *API) userNetIsVisible(userID gp.UserID, netID gp.NetworkID) (err error) {
	in, err := api.userInNetwork(userID, netID)
	if err != nil {
		return
	}
	if in {
		return nil
	}
	parent, err := api.networkParent(netID)
	if err != nil {
		err = NoSuchNetwork
		return
	}
	in, err = api.userInNetwork(userID, parent)
	if err != nil {
		return NoSuchNetwork
	}
	if !in {
		//Can't see a group in another university
		err = NoSuchNetwork
		return
	}
	net, err := api.getNetwork(netID)
	if err != nil {
		return
	}
	if net.Privacy == "secret" {
		//Can't see a secret group
		err = NoSuchNetwork
		return
	}
	return nil
}

//UserRequestAccess allows a user to request access to a private group. It's idempotent; requesting multiple times will silently drop the extra requests.
func (api *API) UserRequestAccess(userID gp.UserID, netID gp.NetworkID) (err error) {
	in, err := api.userInNetwork(userID, netID)
	if err != nil {
		return
	}
	if in {
		//Can't request access to a network you're already in
		err = ENOTALLOWED
		return
	}
	isGroup, err := api.isGroup(netID)
	if err != nil {
		if err == sql.ErrNoRows {
			//Can't request access to a network which doesn't exist
			err = NoSuchNetwork
		}
		return
	}
	if !isGroup {
		//Can't request access to a university
		err = NoSuchNetwork
		return
	}
	parent, err := api.networkParent(netID)
	if err != nil {
		return
	}
	in, err = api.userInNetwork(userID, parent)
	if err != nil {
		return
	}
	if !in {
		//Can't request access to a group in another university
		err = NoSuchNetwork
		return
	}
	net, err := api.getNetwork(netID)
	if err != nil {
		return
	}
	if net.Privacy == "secret" {
		//Can't request access to a secret group
		err = NoSuchNetwork
		return
	}
	if net.Privacy == "public" {
		//Can't request access to a public group
		err = ENOTALLOWED
		return
	}
	s, err := api.sc.Prepare("INSERT INTO network_requests(user_id, network_id) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(userID, netID)
	if err != nil {
		if err, ok := err.(*mysql.MySQLError); ok {
			if err.Number == 1062 {
				//Drop duplicates silently
				return nil
			}
		}
		return
	}
	api.notifObserver.Notify(requestEvent{userID: userID, groupID: netID})
	return
}

func (api *API) setRequestStatus(userID gp.UserID, groupID gp.NetworkID, status string, processor gp.UserID) (err error) {
	s, err := api.sc.Prepare("UPDATE network_requests SET status = ?, processed_by = ? WHERE user_id = ? AND network_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(status, processor, userID, groupID)
	return
}

//NetworkManager provides access to network data.
type NetworkManager struct {
	sc *psc.StatementCache
}

func (nm *NetworkManager) networkStaff(netID gp.NetworkID) (staff []gp.UserID, err error) {
	admins, err := nm.getNetworkAdmins(netID)
	if err != nil {
		return
	}
	creator, err := nm.networkCreator(netID)
	if err != nil {
		return
	}
	staff = append(staff, creator)
	for _, admin := range admins {
		staff = append(staff, admin.ID)
	}
	return
}

//GroupsByMembershipCount returns the usergroups in this user's university, sorted by membership count / id.
func (api *API) GroupsByMembershipCount(userID gp.UserID, index int64, count int) (groups []gp.GroupSubjective, err error) {
	q := "SELECT id, name, cover_img, `desc`, creator, privacy, COUNT(user_id) as cnt " +
		"FROM network " +
		"JOIN user_network ON network.id = user_network.network_id " +
		"WHERE user_group = 1 " +
		"AND privacy != 'secret' " +
		"AND parent = ? " +
		"GROUP BY network.id " +
		"ORDER BY cnt DESC, id ASC " +
		"LIMIT ?, ?"
	groups = make([]gp.GroupSubjective, 0)
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(primary.ID, index, count)
	if err != nil {
		return
	}
	defer rows.Close()
	var img, desc, privacy sql.NullString
	var creator sql.NullInt64
	for rows.Next() {
		group := gp.GroupSubjective{}
		err = rows.Scan(&group.ID, &group.Name, &img, &desc, &creator, &privacy, &group.MemberCount)
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
		}
		if desc.Valid {
			group.Desc = desc.String
		}
		if privacy.Valid {
			group.Privacy = privacy.String
		}
		var role gp.Role
		role, err = api.userRole(userID, group.ID)
		if err == nil {
			group.YourRole = &role
			group.Conversation, _ = api.groupConversation(group.ID)
			group.UnreadCount, _ = api.userConversationUnread(userID, group.Conversation)
			group.NewPosts, _ = api.groupNewPosts(userID, group.ID)
			status, err := api.pendingRequestExists(userID, group.ID)
			if err == nil && (status == "pending" || status == "rejected") {
				group.PendingRequest = true
			}

		}
		groups = append(groups, group)
	}
	return groups, nil
}

//NetworkRequests enumerates the outstanding requests to join this network.
func (api *API) NetworkRequests(userID gp.UserID, netID gp.NetworkID) (requests []gp.NetRequest, err error) {
	requests = make([]gp.NetRequest, 0)
	in, err := api.userInNetwork(userID, netID)
	if err != nil {
		return
	}
	if !in {
		err = ENOTALLOWED
		return
	}
	q := "SELECT user_id, request_time FROM network_requests WHERE network_id = ? AND status = 'pending'"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(netID)
	if err != nil {
		return
	}
	defer rows.Close()
	var id gp.UserID
	var t string
	for rows.Next() {
		req := gp.NetRequest{}
		err = rows.Scan(&id, &t)
		if err != nil {
			return
		}
		req.Requester, err = api.users.byID(id)
		if err != nil {
			return
		}
		req.ReqTime, err = time.Parse(mysqlTime, t)
		if err != nil {
			return
		}
		req.Status = "pending"
		requests = append(requests, req)
	}
	return
}

//RejectNetworkRequest marks a request to join a private group as rejected. It can only be used by group administrators, and will not inform the rejectee,
func (api *API) RejectNetworkRequest(userID gp.UserID, netID gp.NetworkID, reqID gp.UserID) (err error) {
	err = api.userNetIsVisible(userID, netID)
	if err != nil {
		return
	}

	has, err := api.userHasRole(userID, netID, "administrator")
	if err != nil {
		return
	}
	if !has {
		return ENOTALLOWED
	}
	status, err := api.pendingRequestExists(reqID, netID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = NoSuchRequest
		}
		return
	}
	switch {
	case status == "rejected":
		return AlreadyRejected
	case status == "accepted":
		return AlreadyAccepted
	default:
		err = api.setRequestStatus(reqID, netID, "rejected", userID)
		return
	}
}

func (api *API) pendingRequestExists(reqID gp.UserID, netID gp.NetworkID) (status string, err error) {
	s, err := api.sc.Prepare("SELECT status FROM network_requests WHERE user_id = ? AND network_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(reqID, netID).Scan(&status)
	return

}
