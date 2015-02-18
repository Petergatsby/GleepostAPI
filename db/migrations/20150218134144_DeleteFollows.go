package main

import (
	"database/sql"
	"log"
)

// Up is executed when this migration is applied
func Up_20150218134144(txn *sql.Tx) {
	_, err := txn.Query("DROP TABLE follows")
	if err != nil {
		log.Println(err)
		return
	}
}

// Down is executed when this migration is rolled back
func Down_20150218134144(txn *sql.Tx) {
	q := "CREATE TABLE `follows` ( " +
		"`leader` int(10) unsigned NOT NULL, " +
		"`follower` int(10) unsigned NOT NULL, " +
		") " +
		"ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;"
	_, err := txn.Query(q)
	if err != nil {
		log.Println(err)
		return
	}
}
