package main

import (
	"database/sql"
	"log"
)

// Up20150831171048 is executed when this migration is applied
func Up20150831171048(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE user_network ADD join_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP")
	if err != nil {
		txn.Rollback()
		log.Println(err)
		return
	}
}

// Down20150831171048 is executed when this migration is rolled back
func Down20150831171048(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE user_network DROP COLUMN join_time")
	if err != nil {
		txn.Rollback()
		log.Println(err)
	}
}
