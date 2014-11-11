package main

import (
	"database/sql"
)

// Up is executed when this migration is applied
func Up_20141107160321(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE devices ADD application VARCHAR(100) NOT NULL DEFAULT 'gleepost'")
	if err != nil {
		txn.Rollback()
	}
}

// Down is executed when this migration is rolled back
func Down_20141107160321(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE devices DROP COLUMN application")
	if err != nil {
		txn.Rollback()
	}
}
