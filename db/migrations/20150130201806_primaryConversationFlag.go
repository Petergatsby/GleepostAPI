package main

import (
	"database/sql"
	"log"
)

//Up_20150130201806 is executed when this migration is applied
func Up_20150130201806(txn *sql.Tx) {
	log.Println("Adding primary_conversation flag")
	_, err := txn.Query("ALTER TABLE conversations ADD primary_conversation BOOLEAN NOT NULL DEFAULT 0")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	log.Println("Setting primary_conversation flag for all 2-person conversations")
	_, err = txn.Query("UPDATE conversations SET primary_conversation = 1 WHERE id IN (SELECT conversation_id FROM conversation_participants GROUP BY conversation_id HAVING COUNT(*) = 2)")
	if err != nil {
		txn.Rollback()
		return
	}
	log.Println("Adding merged_into reference on conversations")
	_, err = txn.Query("ALTER TABLE conversations ADD merged INT(10) UNSIGNED NULL")
	if err != nil {
		log.Println(err)
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
