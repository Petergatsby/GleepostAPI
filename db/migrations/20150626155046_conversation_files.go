package main

import (
	"database/sql"
	"log"
)

// Up20150626155046 is executed when this migration is applied
func Up20150626155046(txn *sql.Tx) {
	q := "CREATE TABLE conversation_files ( "
	q += "message_id INT(10) UNSIGNED NOT NULL, "
	q += "url varchar(255) NOT NULL, "
	q += "type varchar(15) NOT NULL, "
	q += "INDEX msg (`message_id`) "
	q += ") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
	_, err := txn.Exec(q)
	if err != nil {
		txn.Rollback()
		log.Println(err)
	}
}

// Down20150626155046 is executed when this migration is rolled back
func Down20150626155046(txn *sql.Tx) {
	_, err := txn.Exec("DROP TABLE conversation_files")
	if err != nil {
		txn.Rollback()
		log.Println(err)
	}
}
