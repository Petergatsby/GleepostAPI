package main

import (
	"database/sql"
	"log"
)

// Up20150608133539 is executed when this migration is applied
func Up20150608133539(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE conversation_participants ADD muted BOOL NOT NULL DEFAULT 0")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
}

// Down20150608133539 is executed when this migration is rolled back
func Down20150608133539(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE conversation_participants DROP COLUMN muted")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
}
