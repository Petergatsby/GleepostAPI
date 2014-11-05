package db

import (
	"database/sql"
	"strings"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//ApproveAccess indicates whether you are allowed to access gleepost approve, and change its settings.
func (db *DB) ApproveAccess(userID gp.UserID, netID gp.NetworkID) (perm gp.ApprovePermission, err error) {
	q := "SELECT role_level FROM user_network JOIN network ON network.master_group = user_network.network_id WHERE network.id = ? AND user_network.user_id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	var level int
	err = s.QueryRow(netID, userID).Scan(&level)
	switch {
	case err != nil && err == sql.ErrNoRows:
		return perm, nil
	case err != nil:
		return perm, err
	default:
		if level > 0 {
			perm.ApproveAccess = true
		}
		if level > 1 {
			perm.LevelChange = true
		}
		return perm, nil
	}

}

//ApproveLevel returns this network's current approval level.
func (db *DB) ApproveLevel(netID gp.NetworkID) (level gp.ApproveLevel, err error) {
	q := "SELECT approval_level, approved_categories FROM network WHERE id = ?"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	var approvedCategories sql.NullString
	err = s.QueryRow(netID).Scan(&level.Level, &approvedCategories)
	if err != nil {
		return
	}
	cats := []string{}
	if approvedCategories.Valid {
		cats = strings.Split(approvedCategories.String, ",")
	}
	level.Categories = cats
	return level, nil
}

//SetApproveLevel updates this network's approval level.
func (db *DB) SetApproveLevel(netID gp.NetworkID, level int) (err error) {
	q := "UPDATE network SET approval_level = ?, approved_categories = ? WHERE id = ?"
	var categories string
	switch {
	case level == 0:
		categories = ""
	case level == 1:
		categories = "parties"
	case level == 2:
		categories = "events"
	case level == 3:
		categories = "all"
	default:
		return gp.APIerror{Reason: "That's not a valid approve level"}
	}
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(level, categories, netID)
	return
}
