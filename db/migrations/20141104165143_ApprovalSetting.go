package main

import (
	"database/sql"
)

//Up20141104165143 is executed when this migration is applied
func Up20141104165143(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE network ADD approval_level INT(5) UNSIGNED NOT NULL DEFAULT 0")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("ALTER TABLE network ADD approved_categories VARCHAR(255) NULL")
	if err != nil {
		txn.Rollback()
	}
}

//Down20141104165143 is executed when this migration is rolled back
func Down20141104165143(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE network DROP COLUMN approval_level")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("ALTER TABLE network DROP COLUMN approved_categories")
	if err != nil {
		txn.Rollback()
	}
}
