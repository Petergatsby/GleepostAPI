package main

import (
	"database/sql"
	"log"
)

// Up20150831164637 is executed when this migration is applied
func Up20150831164637(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE user_network ADD seen_upto INT(10) UNSIGNED DEFAULT 0")
	if err != nil {
		txn.Rollback()
		log.Println(err)
		return
	}
}

// Down20150831164637 is executed when this migration is rolled back
func Down20150831164637(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE user_network DROP COLUMN seen_upto")
	if err != nil {
		txn.Rollback()
		log.Println(err)
	}
}
