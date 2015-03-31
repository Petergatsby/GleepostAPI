package main

import (
	"database/sql"
)

//Up20141107160321 is executed when this migration is applied
func Up20141107160321(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE devices ADD application VARCHAR(100) NOT NULL DEFAULT 'gleepost'")
	if err != nil {
		txn.Rollback()
	}
}

//Down20141107160321 is executed when this migration is rolled back
func Down20141107160321(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE devices DROP COLUMN application")
	if err != nil {
		txn.Rollback()
	}
}
