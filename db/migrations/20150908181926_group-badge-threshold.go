package main

import (
	"database/sql"
	"log"
)

// Up20150908181926 is executed when this migration is applied
func Up20150908181926(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ADD group_badge_threshold DATETIME NOT NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down20150908181926 is executed when this migration is rolled back
func Down20150908181926(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users DROP COLUMN group_badge_threshold")
	if err != nil {
		txn.Rollback()
	}
}
