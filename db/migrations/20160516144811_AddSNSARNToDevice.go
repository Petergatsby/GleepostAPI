package main

import (
	"database/sql"
	"log"
)

// Up is executed when this migration is applied
func Up20160516144811(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE devices ADD arn VARCHAR(300) NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down is executed when this migration is rolled back
func Down20160516144811(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE devices DROP COLUMN arn")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}
