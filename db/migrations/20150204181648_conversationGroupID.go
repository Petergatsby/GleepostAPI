package main

import (
	"database/sql"
)

//Up20150204181648 is executed when this migration is applied
func Up20150204181648(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE conversations ADD group_id INT(10) UNSIGNED NULL")
	if err != nil {
		txn.Rollback()
		return
	}
}

//Down20150204181648 is executed when this migration is rolled back
func Down20150204181648(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE conversations DROP COLUMN group_id")
	if err != nil {
		txn.Rollback()
		return
	}
}
