package main

import (
	"database/sql"
)

// Up20150406171351 is executed when this migration is applied
func Up20150406171351(txn *sql.Tx) {
	_, err := txn.Exec("INSERT INTO categories (tag, name) VALUES ('poll', 'Poll')")
	if err != nil {
		txn.Rollback()
		return
	}
}

// Down20150406171351 is executed when this migration is rolled back
func Down20150406171351(txn *sql.Tx) {
	_, err := txn.Exec("DELETE FROM categories WHERE tag = 'poll'")
	if err != nil {
		txn.Rollback()
		return
	}

}
