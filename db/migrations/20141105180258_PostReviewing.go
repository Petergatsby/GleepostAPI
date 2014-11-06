package main

import (
	"database/sql"
)

// Up is executed when this migration is applied
func Up_20141105180258(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE wall_posts ADD pending BOOLEAN NOT NULL DEFAULT 0")
	if err != nil {
		txn.Rollback()
	}
}

// Down is executed when this migration is rolled back
func Down_20141105180258(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE wall_posts DROP COLUMN pending")
	if err != nil {
		txn.Rollback()
	}
}
