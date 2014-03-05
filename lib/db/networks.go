package db

import (
	"database/sql"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"log"
)

/********************************************************************
		Network
********************************************************************/

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
func (db *DB) GetUserNetworks(id gp.UserId, userGroupsOnly bool) (networks []gp.Network, err error) {
	networkSelect := "SELECT user_network.network_id, network.name " +
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
	defer rows.Close()
	log.Println("DB hit: getUserNetworks userid (network.id, network.name)")
	if err != nil {
		return
	}
	for rows.Next() {
		var network gp.Network
		err = rows.Scan(&network.Id, &network.Name)
		if err != nil {
			return
		} else {
			networks = append(networks, network)
		}
	}
	return
}

func (db *DB) SetNetwork(userId gp.UserId, networkId gp.NetworkId) (err error) {
	networkInsert := "REPLACE INTO user_network (user_id, network_id) VALUES (?, ?)"
	s, err := db.prepare(networkInsert)
	if err != nil {
		return
	}
	_, err = s.Exec(userId, networkId)
	return
}

//GetNetwork returns the network netId.
//TODO: add extra details.
func (db *DB) GetNetwork(netId gp.NetworkId) (network gp.Network, err error) {
	networkSelect := "SELECT network.name " +
		"FROM network " +
		"WHERE network.id = ?"
	s, err := db.prepare(networkSelect)
	if err != nil {
		return
	}
	err = s.QueryRow(netId).Scan(&network.Name)
	if err != nil {
		return
	}
	network.Id = netId
	return
}

//CreateNetwork creates a new network. usergroup indicates that the group is user-defined (created by a user rather than system-defined networks such as universities)
func (db *DB) CreateNetwork(name string, usergroup bool) (network gp.Network, err error) {
	networkInsert := "INSERT INTO network (name, user_group) VALUES (?, ?)"
	s, err := db.prepare(networkInsert)
	if err != nil {
		return
	}
	res, err := s.Exec(name, usergroup)
	if err != nil {
		return
	}
	id, _ := res.LastInsertId()
	network.Id = gp.NetworkId(id)
	network.Name = name
	return
}

//IsGroup returns false if netId isn't a user group, and ErrNoRows if netId doesn't exist.
func (db *DB) IsGroup(netId gp.NetworkId) (group bool, err error) {
	isgroup := "SELECT user_group FROM network WHERE id = ?"
	s, err := db.prepare(isgroup)
	if err != nil {
		return
	}
	err = s.QueryRow(netId).Scan(&group)
	return
}

//GetNetworkUsers returns all the members of the group netId
func (db *DB) GetNetworkUsers(netId gp.NetworkId) (users []gp.User, err error) {
	memberQuery := "SELECT user_id, users.name, users.avatar, users.firstname FROM user_network JOIN users ON user_network.user_id = users.id WHERE user_network.network_id = ?"
	s, err := db.prepare(memberQuery)
	if err != nil {
		return
	}
	rows, err := s.Query(netId)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var user gp.User
		var av sql.NullString
		var name sql.NullString
		err = rows.Scan(&user.Id, &user.Name, &av, &name)
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

func (db *DB) LeaveNetwork(userId gp.UserId, netId gp.NetworkId) (err error) {
	leaveQuery := "DELETE FROM user_network WHERE user_id = ? AND network_id = ?"
	s, err := db.prepare(leaveQuery)
	if err != nil {
		return
	}
	_, err = s.Exec(userId, netId)
	return
}
