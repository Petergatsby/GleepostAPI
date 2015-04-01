package main

import (
	"database/sql"
)

// Up20150401214324 is executed when this migration is applied
func Up20150401214324(txn *sql.Tx) {
	_, err := txn.Query("INSERT INTO categories (tag, name) VALUES (?, ?)", "food", "Free Food")
	if err != nil {
		txn.Rollback()
		return
	}
}

// Down20150401214324 is executed when this migration is rolled back
func Down20150401214324(txn *sql.Tx) {
	_, err := txn.Query("DELETE FROM categories WHERE tag = 'food'")
	if err != nil {
		txn.Rollback()
		return
	}
}
