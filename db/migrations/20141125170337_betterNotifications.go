package main

import (
	"database/sql"
)

//Up20141125170337 is executed when this migration is applied
func Up20141125170337(txn *sql.Tx) {
	_, err := txn.Query("UPDATE notifications SET post_id = location_id WHERE post_id IS NULL AND type IN ('commented', 'liked', 'approved_post', 'rejected_post')")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("ALTER TABLE notifications ADD network_id INT(10) UNSIGNED NULL")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("UPDATE notifications SET network_id = location_id WHERE network_id IS NULL AND type IN('group_post', 'added_group')")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("ALTER TABLE notifications DROP COLUMN location_id")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("ALTER TABLE notifications ADD preview_text VARCHAR(100) NULL")
	if err != nil {
		txn.Rollback()
		return
	}
}

//Down20141125170337 is executed when this migration is rolled back
func Down20141125170337(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE notifications ADD location_id INT(10) UNSIGNED NULL")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("UPDATE notifications SET location_id = network_id WHERE type IN ('group_post', 'added_group')")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("ALTER TABLE notifications DROP COLUMN network_id")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("UPDATE notifications SET location_id = post_id WHERE type IN ('commented', 'liked', 'approved_post', 'rejected_post')")
	if err != nil {
		txn.Rollback()
		return
	}

}
