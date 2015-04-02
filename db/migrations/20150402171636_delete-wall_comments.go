package main

import (
	"database/sql"
	"log"
)

// Up20150402171636 is executed when this migration is applied
func Up20150402171636(txn *sql.Tx) {
	_, err := txn.Exec("DROP TABLE wall_comments")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down20150402171636 is executed when this migration is rolled back
func Down20150402171636(txn *sql.Tx) {
	q := "CREATE TABLE `wall_comments` ( "
	q += "`id` int(10) unsigned NOT NULL AUTO_INCREMENT, "
	q += "`post_id` int(10) unsigned NOT NULL, "
	q += "`by` int(10) unsigned NOT NULL, "
	q += "`time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, "
	q += "`text` varchar(1024) COLLATE utf8_bin NOT NULL, "
	q += "PRIMARY KEY (`id`) "
	q += ") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;"
	_, err := txn.Exec(q)
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}
