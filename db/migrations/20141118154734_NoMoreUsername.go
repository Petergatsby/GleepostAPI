package main

import (
	"database/sql"
)

// Up is executed when this migration is applied
func Up_20141118154734(txn *sql.Tx) {
	_, err := txn.Query("UPDATE users SET firstname = name WHERE firstname IS NULL")
	if err != nil {
		txn.Rollback()
	}
	_, err = txn.Query("ALTER TABLE users DROP COLUMN name")
	if err != nil {
		txn.Rollback()
	}
	_, err = txn.Query("ALTER TABLE users ALTER COLUMN firstname VARCHAR NOT NULL")
	if err != nil {
		txn.Rollback()
	}

}

// Down is executed when this migration is rolled back
func Down_20141118154734(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ADD name VARCHAR(255) NOT NULL BEFORE password")
	if err != nil {
		return
	}
	_, err = txn.Query("UPDATE users SET name = firstname")
	if err != nil {
		return
	}
	_, err = txn.Query("ALTER TABLE users ALTER COLUMN firstname VARCHAR NULL")
	if err != nil {
		return
	}
}
