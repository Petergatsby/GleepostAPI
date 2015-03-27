package main

import (
	"database/sql"
	"log"
)

//Up20150327155038 is executed when this migration is applied
func Up20150327155038(txn *sql.Tx) {
	q := "CREATE TABLE `post_templates` ( "
	q += "`id` int(10) unsigned NOT NULL AUTO_INCREMENT, "
	q += "`set` int(10) unsigned NOT NULL, "
	q += "`template` TEXT(4096) NOT NULL, "
	q += "PRIMARY KEY (`id`) ) "
	q += "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;"
	_, err := txn.Query(q)
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

//Down20150327155038 is executed when this migration is rolled back
func Down20150327155038(txn *sql.Tx) {
	_, err := txn.Query("DROP TABLE post_templates")
	if err != nil {
		txn.Rollback()
	}
}
