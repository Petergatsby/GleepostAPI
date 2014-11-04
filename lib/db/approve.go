package db

import "github.com/draaglom/GleepostAPI/lib/gp"

func (db *DB) ApproveAccess(userID gp.UserID, netID gp.NetworkID) (perm gp.ApprovePermission, err error) {
	q := "SELECT role_level FROM user_network JOIN network ON network.master_group = user_network.network_id WHERE user_network.user_id = ? AND network.id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	var level int
	err = s.QueryRow(userID, netID).Scan(&level)
	if err != nil {
		return
	}
	if level > 0 {
		perm.ApproveAccess = true
	}
	if level > 1 {
		perm.LevelChange = true
	}
	return perm, nil

}
