package db

import (
	"database/sql"
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//SearchUsersInNetwork returns users whose name begins with first and last within netId.
func (db *DB) SearchUsersInNetwork(first, last string, netID gp.NetworkID) (users []gp.FullNameUser, err error) {
	users = make([]gp.FullNameUser, 0)
	search := "SELECT id, avatar, firstname, lastname, official " +
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
		var av, last sql.NullString
		var user gp.FullNameUser
		err = rows.Scan(&user.ID, &av, &user.Name, &last, &user.Official)
		if err != nil {
			return
		}
		if av.Valid {
			user.Avatar = av.String
		}
		if last.Valid {
			user.FullName = user.Name + " " + last.String
		}
		users = append(users, user)
	}
	return
}

//SearchGroups searches for groups which are 'within' parent; it currently just matches %name%.
func (db *DB) SearchGroups(parent gp.NetworkID, name string) (groups []gp.Group, err error) {
	groups = make([]gp.Group, 0)
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
			group.MemberCount, _ = db.GroupMemberCount(group.ID)
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
