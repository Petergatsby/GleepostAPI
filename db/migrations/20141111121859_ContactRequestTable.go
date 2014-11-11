package main

import (
	"database/sql"
)

// Up is executed when this migration is applied
func Up_20141111121859(txn *sql.Tx) {
	q := "CREATE TABLE `contact_requests` ( " +
		"`id` int(10) unsigned NOT NULL AUTO_INCREMENT, " +
		"`full_name` varchar(255) COLLATE utf8_bin NOT NULL, " +
		"`college` varchar(255) COLLATE utf8_bin NOT NULL, " +
		"`email` varchar(255) COLLATE utf8_bin NOT NULL, " +
		"`phone_no` varchar(255) COLLATE utf8_bin NOT NULL, " +
		"`timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, " +
		"PRIMARY KEY (`id`) ) " +
		"ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;"
	_, err := txn.Query(q)
	if err != nil {
		txn.Rollback()
	}
}

// Down is executed when this migration is rolled back
func Down_20141111121859(txn *sql.Tx) {
	_, err := txn.Query("DROP TABLE contact_requests")
	if err != nil {
		txn.Rollback()
	}

}
