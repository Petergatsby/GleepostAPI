package main

import (
	"database/sql"
	"log"
)

// Up20150909144523 is executed when this migration is applied
func Up20150909144523(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ALTER COLUMN group_badge_threshold SET DEFAULT 0")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down20150909144523 is executed when this migration is rolled back
func Down20150909144523(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ALTER COLUMN group_badge_threshold DROP DEFAULT")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}
