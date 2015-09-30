package main

import (
	"database/sql"
	"log"
)

// Up20150921182913 is executed when this migration is applied
func Up20150921182913(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE network ADD COLUMN covervid_mp4 VARCHAR(255) NULL, ADD COLUMN covervid_webm VARCHAR(255) NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}

}

// Down20150921182913 is executed when this migration is rolled back
func Down20150921182913(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE network DROP COLUMN covervid_mp4, DROP COLUMN covervid_webm")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}
