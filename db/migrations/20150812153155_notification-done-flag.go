package main

import (
	"database/sql"
	"log"
)

// Up20150812153155 is executed when this migration is applied
func Up20150812153155(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE notifications ADD done BOOLEAN NOT NULL DEFAULT 0")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
}

// Down20150812153155 is executed when this migration is rolled back
func Down20150812153155(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE notifications DROP COLUMN done")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
}
