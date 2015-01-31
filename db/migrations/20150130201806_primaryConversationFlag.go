package main

import (
	"database/sql"
)

// Up is executed when this migration is applied
func Up_20150130201806(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE conversations ADD primary_conversation BOOLEAN NOT NULL DEFAULT 0")
	if err != nil {
		txn.Rollback()
	}
	_, err = txn.Query("UPDATE conversations SET primary_conversation = 1 WHERE id IN (SELECT conversation_id FROM conversation_participants GROUP BY conversation_id HAVING COUNT(*) = 2)")
	if err != nil {
		txn.Rollback()
	}
}

// Down is executed when this migration is rolled back
func Down_20150130201806(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE conversations DROP COLUMN primary_conversation")
	if err != nil {
		txn.Rollback()
	}
}
