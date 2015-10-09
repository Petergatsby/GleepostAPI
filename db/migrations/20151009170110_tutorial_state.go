package main

import (
	"database/sql"
	"log"
)

// Up20151009170110 is executed when this migration is applied
func Up20151009170110(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ADD tutorial_state VARCHAR(1024) NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down20151009170110 is executed when this migration is rolled back
func Down20151009170110(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users DROP COLUMN tutorial_state")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}

}
