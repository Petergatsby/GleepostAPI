package main

import (
	"database/sql"
)

// Up is executed when this migration is applied
func Up_20150122160128(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ADD new_message_threshold DATETIME NOT NULL")
	if err != nil {
		txn.Rollback()
	}
}

// Down is executed when this migration is rolled back
func Down_20150122160128(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users DROP COLUMNT new_message_threshold")
	if err != nil {
		txn.Rollback()
	}
}
