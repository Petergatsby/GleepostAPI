package main

import (
	"database/sql"
)

//Up20141105180258 is executed when this migration is applied
func Up20141105180258(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE wall_posts ADD pending BOOLEAN NOT NULL DEFAULT 0")
	if err != nil {
		txn.Rollback()
	}
}

//Down20141105180258 is executed when this migration is rolled back
func Down20141105180258(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE wall_posts DROP COLUMN pending")
	if err != nil {
		txn.Rollback()
	}
}
