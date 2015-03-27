package main

import (
	"database/sql"
	"log"
)

//Up20150225162015 is executed when this migration is applied
func Up20150225162015(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE conversations DROP COLUMN last_mod")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

//Down20150225162015 is executed when this migration is rolled back
func Down20150225162015(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE conversations ADD last_mod TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}
