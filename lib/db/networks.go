package db

import (
	"database/sql"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//GetRules returns all the network matching rules for every network.
func (db *DB) GetRules() (rules []gp.Rule, err error) {
	ruleSelect := "SELECT network_id, rule_type, rule_value FROM net_rules"
	s, err := db.prepare(ruleSelect)
	if err != nil {
		return
	}
	rows, err := s.Query()
	log.Println("DB hit: GetRules()")
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

//GetUserUniversity returns this user's primary network (ie, their university)
func (db *DB) GetUserUniversity(id gp.UserID) (network gp.GroupMembership, err error) {
	s, err := db.prepare("SELECT user_network.network_id, network.name, user_network.role, user_network.role_level, network.cover_img, network.`desc`, network.creator, network.privacy FROM user_network JOIN network ON user_network.network_id = network.id WHERE user_network.user_id = ? AND network.is_university = 1 ")
	if err != nil {
		return
	}
	var img, desc, privacy sql.NullString
	var creator sql.NullInt64
	err = s.QueryRow(id).Scan(&network.ID, &network.Group.Network.Name, &network.Role.Name, &network.Role.Level, &img, &desc, &creator, &privacy)
	if img.Valid {
		network.Image = img.String
	}
	if desc.Valid {
		network.Desc = desc.String
	}
	if creator.Valid {
		u, err := db.GetUser(gp.UserID(creator.Int64))
		if err == nil {
			network.Creator = &u
		}
		network.MemberCount, _ = db.GroupMemberCount(network.ID)
		//TODO(patrick) - maybe don't display group conversation id if you're not a member.
		network.Conversation, _ = db.GroupConversation(network.ID)
		network.UnreadCount, _ = db.UserConversationUnread(id, network.Conversation)
	}
	if privacy.Valid {
		network.Privacy = privacy.String
	}

	return
}

//GetUserNetworks returns all the networks id is a member of, optionally only returning user-created networks.
func (db *DB) GetUserNetworks(id gp.UserID, userGroupsOnly bool) (networks []gp.GroupMembership, err error) {
	networks = make([]gp.GroupMembership, 0)
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
		"WHERE user_id = ? "
	if userGroupsOnly {
		networkSelect += "AND network.user_group = 1 "
	}
	networkSelect += "ORDER BY last_activity DESC"
	s, err := db.prepare(networkSelect)
	if err != nil {
		return
	}
	rows, err := s.Query(id)
	log.Printf("DB hit: GetUserNetworks(%d, %t)\n", id, userGroupsOnly)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var network gp.GroupMembership
		var img, desc sql.NullString
		var creator sql.NullInt64
		var privacy sql.NullString
		var lastActivity string
		err = rows.Scan(&network.ID, &network.Role.Name, &network.Role.Level, &network.Group.Name, &img, &desc, &creator, &privacy, &lastActivity)
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
			u, err := db.GetUser(gp.UserID(creator.Int64))
			if err == nil {
				network.Creator = &u
			}
			network.MemberCount, _ = db.GroupMemberCount(network.ID)
			//TODO(patrick) - maybe don't display group conversation id if you're not a member.
			network.Conversation, _ = db.GroupConversation(network.ID)
			network.UnreadCount, _ = db.UserConversationUnread(id, network.Conversation)
		}
		if privacy.Valid {
			network.Privacy = privacy.String
		}
		networks = append(networks, network)
	}
	return
}

//SubjectiveMembershipCount is the number of groups user belongs to, from the point of view of perspective.
//That is: the public / private groups they're a part of, plus the secret groups that perspective is also in.
func (db *DB) SubjectiveMembershipCount(perspective, user gp.UserID) (count int, err error) {
	q := "SELECT COUNT(*) FROM user_network JOIN network ON user_network.network_id = network.id "
	q += "WHERE user_group = 1 AND parent = (SELECT network_id FROM user_network WHERE user_id = ? LIMIT 1) "
	q += "AND (privacy != 'secret' OR network.id IN (SELECT network_id FROM user_network WHERE user_id = ?)) "
	q += "AND user_network.user_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(perspective, perspective, user).Scan(&count)
	return

}

//SubjectiveMemberships returns all the groups this user is a member of, as far as perspective is concerned.
func (db *DB) SubjectiveMemberships(perspective, user gp.UserID) (groups []gp.GroupMembership, err error) {
	groups = make([]gp.GroupMembership, 0)
	q := "SELECT user_network.network_id, user_network.role, user_network.role_level, network.name, network.cover_img, network.`desc`, network.creator, network.privacy FROM user_network JOIN network ON user_network.network_id = network.id "
	q += "WHERE user_group = 1 AND parent = (SELECT network_id FROM user_network WHERE user_id = ? LIMIT 1) "
	q += "AND (privacy != 'secret' OR network.id IN (SELECT network_id FROM user_network WHERE user_id = ?)) "
	q += "AND user_network.user_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(perspective, perspective, user)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var network gp.GroupMembership
		var img, desc sql.NullString
		var creator sql.NullInt64
		var privacy sql.NullString
		err = rows.Scan(&network.ID, &network.Role.Name, &network.Role.Level, &network.Group.Name, &img, &desc, &creator, &privacy)
		if err != nil {
			return
		}
		if img.Valid {
			network.Image = img.String
		}
		if desc.Valid {
			network.Desc = desc.String
		}
		if creator.Valid {
			u, err := db.GetUser(gp.UserID(creator.Int64))
			if err == nil {
				network.Creator = &u
			}
			network.MemberCount, _ = db.GroupMemberCount(network.ID)
		}
		if privacy.Valid {
			network.Privacy = privacy.String
		}
		groups = append(groups, network)
	}
	return
}

//SetNetwork idempotently makes userID a member of networkID
func (db *DB) SetNetwork(userID gp.UserID, networkID gp.NetworkID) (err error) {
	networkInsert := "REPLACE INTO user_network (user_id, network_id) VALUES (?, ?)"
	s, err := db.prepare(networkInsert)
	if err != nil {
		return
	}
	_, err = s.Exec(userID, networkID)
	return
}

//GetNetwork returns the network netId. If userID is 0, it will omit the group's unread count.
func (db *DB) GetNetwork(netID gp.NetworkID) (network gp.Group, err error) {
	networkSelect := "SELECT name, cover_img, `desc`, creator, user_group, privacy " +
		"FROM network " +
		"WHERE network.id = ?"
	s, err := db.prepare(networkSelect)
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
		u, err := db.GetUser(gp.UserID(creator.Int64))
		if err == nil {
			network.Creator = &u
		}
		network.MemberCount, _ = db.GroupMemberCount(network.ID)
		network.Conversation, _ = db.GroupConversation(network.ID)
	}
	if privacy.Valid {
		network.Privacy = privacy.String
	}
	return
}

//CreateNetwork creates a new network. usergroup indicates that the group is user-defined (created by a user rather than system-defined networks such as universities)
func (db *DB) CreateNetwork(name string, parent gp.NetworkID, url, desc string, creator gp.UserID, usergroup bool, privacy string) (group gp.Group, err error) {
	networkInsert := "INSERT INTO network (name, parent, cover_img, `desc`, creator, user_group, privacy) VALUES (?, ?, ?, ?, ?, ?, ?)"
	s, err := db.prepare(networkInsert)
	if err != nil {
		return
	}
	res, err := s.Exec(name, parent, url, desc, creator, usergroup, privacy)
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
	u, err := db.GetUser(creator)
	if err == nil {
		group.Creator = &u
	} else {
		log.Println("Error getting user:", err)
	}
	return
}

//IsGroup returns false if netId isn't a user group, and ErrNoRows if netId doesn't exist.
func (db *DB) IsGroup(netID gp.NetworkID) (group bool, err error) {
	isgroup := "SELECT user_group FROM network WHERE id = ?"
	s, err := db.prepare(isgroup)
	if err != nil {
		return
	}
	err = s.QueryRow(netID).Scan(&group)
	return
}

//GetNetworkAdmins returns all the administrators of the group netID
func (db *DB) GetNetworkAdmins(netID gp.NetworkID) (users []gp.UserRole, err error) {
	users = make([]gp.UserRole, 0)
	memberQuery := "SELECT user_id, users.avatar, users.firstname, users.official, user_network.role, user_network.role_level FROM user_network JOIN users ON user_network.user_id = users.id WHERE user_network.network_id = ? AND user_network.role = 'administrator'"
	s, err := db.prepare(memberQuery)
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
func (db *DB) GetNetworkUsers(netID gp.NetworkID) (users []gp.UserRole, err error) {
	users = make([]gp.UserRole, 0)
	memberQuery := "SELECT user_id, users.avatar, users.firstname, users.official, user_network.role, user_network.role_level FROM user_network JOIN users ON user_network.user_id = users.id WHERE user_network.network_id = ?"
	s, err := db.prepare(memberQuery)
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
func (db *DB) LeaveNetwork(userID gp.UserID, netID gp.NetworkID) (err error) {
	leaveQuery := "DELETE FROM user_network WHERE user_id = ? AND network_id = ?"
	s, err := db.prepare(leaveQuery)
	if err != nil {
		return
	}
	_, err = s.Exec(userID, netID)
	return
}

//CreateInvite stores an invite for a particular email to a particular network.
func (db *DB) CreateInvite(userID gp.UserID, netID gp.NetworkID, email string, token string) (err error) {
	inviteQuery := "INSERT INTO group_invites (group_id, inviter, email, `key`) VALUES (?, ?, ?, ?)"
	s, err := db.prepare(inviteQuery)
	if err != nil {
		return
	}
	_, err = s.Exec(netID, userID, email, token)
	return
}

//SetNetworkImage updates a network's profile image.
func (db *DB) SetNetworkImage(netID gp.NetworkID, url string) (err error) {
	networkUpdate := "UPDATE network SET cover_img = ? WHERE id = ?"
	s, err := db.prepare(networkUpdate)
	if err != nil {
		return
	}
	_, err = s.Exec(url, netID)
	return
}

//NetworkCreator returns the user who created this network.
func (db *DB) NetworkCreator(netID gp.NetworkID) (creator gp.UserID, err error) {
	qCreator := "SELECT creator FROM network WHERE id = ?"
	s, err := db.prepare(qCreator)
	if err != nil {
		return
	}
	err = s.QueryRow(netID).Scan(&creator)
	return
}

//InviteExists returns true if there is a matching invite for email:invite (that's not already accepted)
func (db *DB) InviteExists(email, invite string) (exists bool, err error) {
	q := "SELECT COUNT(*) FROM group_invites WHERE `email` = ? AND `key` = ? AND `accepted` = 0"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(email, invite).Scan(&exists)
	return
}

//AcceptAllInvites marks all invites as accepted for this email address.
func (db *DB) AcceptAllInvites(email string) (err error) {
	q := "UPDATE group_invites SET accepted = 1 WHERE email = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(email)
	return
}

//AssignNetworksFromInvites adds user to all networks which email has been invited to.
//TODO: only do un-accepted invites (!)
func (db *DB) AssignNetworksFromInvites(user gp.UserID, email string) (err error) {
	q := "REPLACE INTO user_network (user_id, network_id) SELECT ?, group_id FROM group_invites WHERE email = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(user, email)
	return
}

//AssignNetworksFromFBInvites adds user to all networks which this facebook id has been invited to.
//TODO: only do un-accepted invites (!)
func (db *DB) AssignNetworksFromFBInvites(user gp.UserID, facebook uint64) (err error) {
	q := "REPLACE INTO user_network (user_id, network_id) SELECT ?, network_id FROM fb_group_invites WHERE facebook_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(user, facebook)
	return
}

//AcceptAllFBInvites marks all invites for this facebook user as accepted.
func (db *DB) AcceptAllFBInvites(facebook uint64) (err error) {
	q := "UPDATE fb_group_invites SET accepted = 1 WHERE facebook_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(facebook)
	return
}

//UserAddFBUserToGroup records that this facebook user has been invited to netID.
func (db *DB) UserAddFBUserToGroup(user gp.UserID, fbuser uint64, netID gp.NetworkID) (err error) {
	q := "INSERT INTO fb_group_invites (inviter_user_id, facebook_id, network_id) VALUES (?, ?, ?)"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(user, fbuser, netID)
	return
}

//SetNetworkParent records that this network is a sub-network of parent (at the moment just used for visibility).
func (db *DB) SetNetworkParent(network, parent gp.NetworkID) (err error) {
	q := "UPDATE network SET parent = ? WHERE id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(parent, network)
	return
}

//NetworkParent returns the ID of this network's parent, or zero if it has none.
func (db *DB) NetworkParent(netID gp.NetworkID) (parent gp.NetworkID, err error) {
	q := "SELECT parent FROM network WHERE id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(netID).Scan(&parent)
	return
}

//UserRole gives this user's role:level in this network, or ENOSUCHUSER if the user isn't part of the network.
func (db *DB) UserRole(user gp.UserID, network gp.NetworkID) (role gp.Role, err error) {
	q := "SELECT role, role_level FROM user_network WHERE user_id = ? AND network_id = ?"
	s, err := db.prepare(q)
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
func (db *DB) UserSetRole(user gp.UserID, network gp.NetworkID, role gp.Role) (err error) {
	q := "UPDATE user_network SET role = ?, role_level = ? WHERE user_id = ? AND network_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(role.Name, role.Level, user, network)
	return
}

//GroupMemberCount returns the number of members this group has.
func (db *DB) GroupMemberCount(network gp.NetworkID) (count int, err error) {
	q := "SELECT COUNT(*) FROM user_network WHERE network_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(network).Scan(&count)
	return
}

//GroupConversation returns this group's conversation ID.
func (db *DB) GroupConversation(group gp.NetworkID) (conversation gp.ConversationID, err error) {
	q := "SELECT id FROM conversations WHERE group_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(group).Scan(&conversation)
	return
}

//UserInNetwork returns true iff this user is in this network.
func (db *DB) UserInNetwork(userID gp.UserID, network gp.NetworkID) (in bool, err error) {
	s, err := db.prepare("SELECT COUNT(*) FROM user_network WHERE user_id = ? AND network_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(userID, network).Scan(&in)
	return
}

//CreateUniversity creates a new university network with this name.
func (db *DB) CreateUniversity(name string) (network gp.Network, err error) {
	s, err := db.prepare("INSERT INTO network (name, is_university, user_group) VALUES (?, 1, 0)")
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
func (db *DB) AddNetworkRules(netID gp.NetworkID, domains ...string) (err error) {
	s, err := db.prepare("INSERT INTO net_rules (network_id, rule_type, rule_value) VALUES (?, 'email', ?)")
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
func (db *DB) NetworkDomain(netID gp.NetworkID) (domain string, err error) {
	s, err := db.prepare("SELECT rule_value FROM net_rules WHERE rule_type = 'email' AND network_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(netID).Scan(&domain)

	return
}
