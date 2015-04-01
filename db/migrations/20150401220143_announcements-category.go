package main

import (
	"database/sql"
)

// Up20150401220143 is executed when this migration is applied
func Up20150401220143(txn *sql.Tx) {
	_, err := txn.Query("INSERT INTO categories (tag, name) VALUES (?, ?)", "announcement", "Announcements")
	if err != nil {
		txn.Rollback()
		return
	}

}

// Down20150401220143 is executed when this migration is rolled back
func Down20150401220143(txn *sql.Tx) {
	_, err := txn.Query("DELETE FROM categories WHERE tag = 'announcement'")
	if err != nil {
		txn.Rollback()
		return
	}

}
