package main

import (
	"database/sql"
	"log"
)

// Up20150929172651 is executed when this migration is applied
func Up20150929172651(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ADD external_id VARCHAR(64) NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}

}

// Down20150929172651 is executed when this migration is rolled back
func Down20150929172651(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users DROP COLUMN external_id")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
}
