package main

import (
	"database/sql"
	"log"
)

// Up20150921163008 is executed when this migration is applied
func Up20150921163008(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE facebook ADD COLUMN fb_token VARCHAR(1024) NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down20150921163008 is executed when this migration is rolled back
func Down20150921163008(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE facebook DROP COLUMN fb_token")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}
