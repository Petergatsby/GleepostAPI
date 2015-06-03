package main

import (
	"database/sql"
	"log"
)

// Up20150602171801 is executed when this migration is applied
func Up20150602171801(txn *sql.Tx) {
	_, err := txn.Exec("ALTER TABLE conversation_participants ADD read_at DATETIME NULL")
	if err != nil {
		log.Println(err)
		return
	}

}

// Down20150602171801 is executed when this migration is rolled back
func Down20150602171801(txn *sql.Tx) {
	_, err := txn.Exec("ALTER TABLE conversation_participants DROP COLUMN read_at")
	if err != nil {
		log.Println(err)
		return
	}
}
