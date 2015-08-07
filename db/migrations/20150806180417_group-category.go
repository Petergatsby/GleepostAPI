package main

import (
	"database/sql"
	"log"
)

// Up20150806180417 is executed when this migration is applied
func Up20150806180417(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE network ADD category VARCHAR(25) NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}

}

// Down20150806180417 is executed when this migration is rolled back
func Down20150806180417(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE network DROP COLUMN category")
	if err != nil {
		txn.Rollback()
	}

}
