package main

import (
	"database/sql"
)

//Up20150122160128 is executed when this migration is applied
func Up20150122160128(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ADD new_message_threshold DATETIME NOT NULL")
	if err != nil {
		txn.Rollback()
	}
}

//Down20150122160128 is executed when this migration is rolled back
func Down20150122160128(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users DROP COLUMN new_message_threshold")
	if err != nil {
		txn.Rollback()
	}
}
