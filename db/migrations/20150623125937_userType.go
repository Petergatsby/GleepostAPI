package main

import (
	"database/sql"
	"log"
)

// Up20150623125937 is executed when this migration is applied
func Up20150623125937(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ADD type VARCHAR(10) NOT NULL DEFAULT 'student'")
	if err != nil {
		txn.Rollback()
		log.Println(err)
	}
}

// Down20150623125937 is executed when this migration is rolled back
func Down20150623125937(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users DROP COLUMN type")
	if err != nil {
		txn.Rollback()
		log.Println(err)
	}
}
