package main

import (
	"database/sql"
	"log"
)

//Up_20150116222816 is executed when this migration is applied
func Up_20150116222816(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE chat_messages charset=utf8mb4, MODIFY COLUMN `text` VARCHAR(1024) CHARACTER SET utf8mb4")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

//Down_20150116222816 is executed when this migration is rolled back
func Down_20150116222816(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE chat_messages charset=utf8, MODIFY COLUMN `text` VARCHAR(1024) CHARACTER SET utf8")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}
