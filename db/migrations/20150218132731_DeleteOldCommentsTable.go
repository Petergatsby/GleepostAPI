package main

import (
	"database/sql"
	"log"
)

//Up20150218132731 is executed when this migration is applied
func Up20150218132731(txn *sql.Tx) {
	_, err := txn.Query("DROP TABLE comments")
	if err != nil {
		log.Println(err)
		return
	}

}

//Down20150218132731 is executed when this migration is rolled back
func Down20150218132731(txn *sql.Tx) {
	q := "CREATE TABLE `comments` ( " +
		"`id` int(10) unsigned NOT NULL AUTO_INCREMENT, " +
		"`user_id` int(10) unsigned NOT NULL, " +
		"`listing_id int(10) unsigned NOT NULL, " +
		"`text` varchar(512) NOT NULL, " +
		"`timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, " +
		"PRIMARY KEY (`id`) ) " +
		"ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;"
	_, err := txn.Query(q)
	if err != nil {
		log.Println(err)
		return
	}
}
