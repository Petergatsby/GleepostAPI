package main

import (
	"database/sql"
	"log"
)

// Up is executed when this migration is applied
func Up_20150204183513(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE conversation_participants ADD deletion_threshold INT(10) UNSIGNED NOT NULL DEFAULT 0")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}

}

// Down is executed when this migration is rolled back
func Down_20150204183513(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE conversation_participants DROP COLUMN deletion_threshold")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}

}
