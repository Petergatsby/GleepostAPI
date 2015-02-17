package main

import (
	"database/sql"
	"log"
)

// Up is executed when this migration is applied
func Up_20150217123826(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE wall_posts charset=utf8mb4, MODIFY COLUMN `text` VARCHAR(1024) CHARACTER SET utf8mb4")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
	_, err = txn.Query("ALTER TABLE post_comments charset=utf8mb4, MODIFY COLUMN `text` VARCHAR(1024) CHARACTER SET utf8mb4")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down is executed when this migration is rolled back
func Down_20150217123826(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE post_comments charset=utf8, MODIFY COLUMN `text` VARCHAR(1024) CHARACTER SET utf8")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
	_, err = txn.Query("ALTER TABLE wall_posts charset=utf8, MODIFY COLUMN `text` VARCHAR(1024) CHARACTER SET utf8")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}
