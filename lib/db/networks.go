package db

import (
	"database/sql"
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

/********************************************************************
		Network
********************************************************************/

//GetRules returns all the network matching rules for every network.
func (db *DB) GetRules() (rules []gp.Rule, err error) {
	ruleSelect := "SELECT network_id, rule_type, rule_value FROM net_rules"
	s, err := db.prepare(ruleSelect)
	if err != nil {
		return
	}
	rows, err := s.Query()
	log.Println("DB hit: validateEmail (rule.networkid, rule.type, rule.value)")
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

//GetUserNetworks returns all the networks id is a member of, optionally only returning user-created networks.
func (db *DB) GetUserNetworks(id gp.UserID, userGroupsOnly bool) (networks []gp.Group, err error) {
	networkSelect := "SELECT user_network.network_id, network.name, " +
		"network.cover_img, network.`desc`, network.creator, network.privacy " +
		"FROM user_network " +
		"INNER JOIN network ON user_network.network_id = network.id " +
		"WHERE user_id = ?"
	if userGroupsOnly {
		networkSelect += " AND network.user_group = 1"
	}
	s, err := db.prepare(networkSelect)
	if err != nil {
		return
	}
	rows, err := s.Query(id)
	log.Println("DB hit: getUserNetworks userid (network.id, network.name, cover_img, desc, creator)")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var network gp.Group
		var img, desc sql.NullString
		var creator sql.NullInt64
		var privacy sql.NullString
		err = rows.Scan(&network.ID, &network.Name, &img, &desc, &creator, &privacy)
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

//GetNetwork returns the network netId.
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

//GetNetworkUsers returns all the members of the group netId
func (db *DB) GetNetworkUsers(netID gp.NetworkID) (users []gp.User, err error) {
	memberQuery := "SELECT user_id, users.name, users.avatar, users.firstname FROM user_network JOIN users ON user_network.user_id = users.id WHERE user_network.network_id = ?"
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
		var user gp.User
		var av sql.NullString
		var name sql.NullString
		err = rows.Scan(&user.ID, &user.Name, &av, &name)
		if err != nil {
			return
		}
		if av.Valid {
			user.Avatar = av.String
		}
		if name.Valid {
			user.Name = name.String
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
