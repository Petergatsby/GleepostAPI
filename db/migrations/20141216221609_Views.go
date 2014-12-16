package main

import (
	"database/sql"
	"log"
)

// Up is executed when this migration is applied
func Up_20141216221609(txn *sql.Tx) {
	q := "CREATE TABLE `post_views` ( "
	q += "`user_id` int(10) unsigned NOT NULL, "
	q += "`post_id` int(10) unsigned NOT NULL, "
	q += "`ts` datetime NOT NULL ) "
	q += "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;"
	_, err := txn.Query(q)
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down is executed when this migration is rolled back
func Down_20141216221609(txn *sql.Tx) {
	_, err := txn.Query("DROP TABLE post_views")
	if err != nil {
		txn.Rollback()
	}
}
