package main

import (
	"database/sql"
)

//Up20150128152252 is executed when this migration is applied
func Up20150128152252(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE chat_messages ADD system BOOLEAN NOT NULL DEFAULT 0")
	if err != nil {
		txn.Rollback()
		return
	}
}

//Down20150128152252 is executed when this migration is rolled back
func Down20150128152252(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE chat_messages DROP COLUMN system")
	if err != nil {
		txn.Rollback()
		return
	}
}
