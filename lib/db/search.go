package db

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	"database/sql"
)

//SearchUsersInNetwork returns users whose name begins with first and last within netId.
func (db *DB) SearchUsersInNetwork(first, last string, netId gp.NetworkId) (users []gp.User, err error) {
	search := "SELECT id, name, avatar, firstname " +
		"FROM users JOIN user_network ON users.id = user_network.network_id " +
		"WHERE network_id = ? " +
		"AND firstname LIKE ? " +
		"AND lastname LIKE ?"
	first+= "%"
	last+= "%"
	s, err := db.prepare(search)
	if err != nil {
		return
	}
	rows, err := s.Query(netId, first, last)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var av, first sql.NullString
		var user gp.User
		err = rows.Scan(&user.Id, &user.Name, &av, &first)
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
