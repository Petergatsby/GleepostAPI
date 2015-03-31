package main

import (
	"database/sql"
)

//Up20150302154058 is executed when this migration is applied
func Up20150302154058(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ALTER new_message_threshold SET DEFAULT 0")
	if err != nil {
		txn.Rollback()
	}

}

//Down20150302154058 is executed when this migration is rolled back
func Down20150302154058(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ALTER new_message_threshold DROP DEFAULT")
	if err != nil {
		txn.Rollback()
	}
}
