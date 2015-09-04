package main

import (
	"database/sql"
	"log"
)

// Up20150904155652 is executed when this migration is applied
func Up20150904155652(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE network ADD COLUMN shortname VARCHAR(100) NULL, ADD COLUMN appname VARCHAR(100) NULL, ADD COLUMN tagline VARCHAR(255) NULL, ADD COLUMN ios_url VARCHAR(255) NULL, ADD COLUMN android_url VARCHAR(255) NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down20150904155652 is executed when this migration is rolled back
func Down20150904155652(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE network DROP COLUMN shortname, appname, tagline, ios_url, android_url")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}

}
