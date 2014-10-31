package main

import (
	"database/sql"
)

// Up is executed when this migration is applied
func Up_20141031150208(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE network ADD master_group INT(10) UNSIGNED NULL")
	if err != nil {
		txn.Rollback()
	}
}

// Down is executed when this migration is rolled back
func Down_20141031150208(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE network DROP COLUMN master_group")
	if err != nil {
		txn.Rollback()
	}
}
