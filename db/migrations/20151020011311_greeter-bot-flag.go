package main

import (
	"database/sql"
	"log"
)

// Up20151020011311 is executed when this migration is applied
func Up20151020011311(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ADD greeter BOOLEAN DEFAULT 0")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down20151020011311 is executed when this migration is rolled back
func Down20151020011311(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE network DROP COLUMN greeter")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}
