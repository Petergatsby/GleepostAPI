package db

import (
	"database/sql"
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//SearchUsersInNetwork returns users whose name begins with first and last within netId.
func (db *DB) SearchUsersInNetwork(first, last string, netID gp.NetworkID) (users []gp.User, err error) {
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

func (db *DB) SearchGroups(name string) (groups []gp.Group, err error) {
	q := "SELECT id, name, cover_img, `desc`, creator " +
		"FROM network " +
		"WHERE user_group = 1 " +
		"AND name LIKE ?"
	name = "%" + name + "%"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(name)
	if err != nil {
		return
	}
	defer rows.Close()
	var img, desc sql.NullString
	var creator sql.NullInt64
	for rows.Next() {
		group := gp.Group{}
		err = rows.Scan(&group.ID, &group.Name, &img, &desc, &creator)
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
		groups = append(groups, group)
	}
	return
}
