package main

import (
	"database/sql"
)

//Up20141119144338 is executed when this migration is applied
func Up20141119144338(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ADD official BOOLEAN NOT NULL DEFAULT 0")
	if err != nil {
		txn.Rollback()
		return
	}
}

//Down20141119144338 is executed when this migration is rolled back
func Down20141119144338(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users DROP COLUMN official")
	if err != nil {
		txn.Rollback()
		return
	}

}
