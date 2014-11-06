package main

import (
	"database/sql"
)

// Up is executed when this migration is applied
func Up_20141106120249(txn *sql.Tx) {
	q := "CREATE TABLE `post_reviews` ( "
	q += "`id` int(10) unsigned NOT NULL AUTO_INCREMENT, "
	q += "`post_id` int(10) unsigned NOT NULL, "
	q += "`action` varchar(255) COLLATE utf8_bin NOT NULL, "
	q += "`by` int(10) unsigned NOT NULL, "
	q += "`reason` varchar(1024) NULL, "
	q += "`timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, "
	q += "PRIMARY KEY (`id`) ) "
	q += "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;"
	_, err := txn.Query(q)
	if err != nil {
		txn.Rollback()
	}
}

// Down is executed when this migration is rolled back
func Down_20141106120249(txn *sql.Tx) {
	_, err := txn.Query("DROP TABLE post_reviews")
	if err != nil {
		txn.Rollback()
	}
}
