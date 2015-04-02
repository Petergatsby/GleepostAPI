package main

import (
	"database/sql"
	"log"
)

// Up20150402174034 is executed when this migration is applied
func Up20150402174034(txn *sql.Tx) {
	_, err := txn.Query("DROP TABLE IF EXISTS user_at")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down20150402174034 is executed when this migration is rolled back
func Down20150402174034(txn *sql.Tx) {
	q := "CREATE TABLE `user_at` ( "
	q += "`user_id` int(10) unsigned NOT NULL, "
	q += "`address_id` int(10) unsigned NOT NULL, "
	q += "UNIQUE KEY `user_address` (`user_id`,`address_id`) "
	q += ") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;"
	_, err := txn.Exec(q)
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}

}
