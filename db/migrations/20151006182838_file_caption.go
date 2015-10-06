package main

import (
	"database/sql"
	"log"
)

// Up20151006182838 is executed when this migration is applied
func Up20151006182838(txn *sql.Tx) {
	_, err := txn.Exec("ALTER TABLE conversation_files ADD caption VARCHAR(255) NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down20151006182838 is executed when this migration is rolled back
func Down20151006182838(txn *sql.Tx) {
	_, err := txn.Exec("ALTER TABLE conversation_files DROP COLUMN caption")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}
