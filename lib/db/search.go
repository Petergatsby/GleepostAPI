package db

import (
	"database/sql"
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//SearchUsersInNetwork returns users whose name begins with first and last within netId.
func (db *DB) SearchUsersInNetwork(first, last string, netID gp.NetworkID) (users gp.UserList, err error) {
	search := "SELECT id, name, avatar, firstname " +
		"FROM users JOIN user_network ON users.id = user_network.user_id " +
		"WHERE network_id = ? " +
		"AND firstname LIKE ? " +
		"AND lastname LIKE ?"
	first += "%"
	last += "%"
	log.Println(search, first, last)
	s, err := db.prepare(search)
	if err != nil {
		return
	}
	rows, err := s.Query(netID, first, last)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var av, first sql.NullString
		var user gp.User
		err = rows.Scan(&user.ID, &user.Name, &av, &first)
		if err != nil {
			return
		}
		if av.Valid {
			user.Avatar = av.String
		}
		if first.Valid {
			user.Name = first.String
		}
		users = append(users, user)
	}
	return
}

//SearchGroups searches for groups which are 'within' parent; it currently just matches %name%.
func (db *DB) SearchGroups(parent gp.NetworkID, name string) (groups []gp.Group, err error) {
	q := "SELECT id, name, cover_img, `desc`, creator, privacy " +
		"FROM network " +
		"WHERE user_group = 1 " +
		"AND parent = ? " +
		"AND privacy != 'secret' " +
		"AND name LIKE ?"
	name = "%" + name + "%"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(parent, name)
	if err != nil {
		return
	}
	defer rows.Close()
	var img, desc, privacy sql.NullString
	var creator sql.NullInt64
	for rows.Next() {
		group := gp.Group{}
		err = rows.Scan(&group.ID, &group.Name, &img, &desc, &creator, &privacy)
		if err != nil {
			return
		}
		if img.Valid {
			group.Image = img.String
		}
		if creator.Valid {
			u, err := db.GetUser(gp.UserID(creator.Int64))
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
		groups = append(groups, group)
	}
	return
}
