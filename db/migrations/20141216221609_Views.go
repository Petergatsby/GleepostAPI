package main

import (
	"database/sql"
	"log"
)

//Up20141216221609 is executed when this migration is applied
func Up20141216221609(txn *sql.Tx) {
	q := "CREATE TABLE `post_views` ( "
	q += "`user_id` int(10) unsigned NOT NULL, "
	q += "`post_id` int(10) unsigned NOT NULL, "
	q += "`ts` datetime NOT NULL ) "
	q += "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;"
	_, err := txn.Query(q)
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	q = "ALTER TABLE  `gleepost`.`post_views` ADD INDEX  `p_t` (  `post_id` ,  `ts` )"
	_, err = txn.Query(q)
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

//Down20141216221609 is executed when this migration is rolled back
func Down20141216221609(txn *sql.Tx) {
	_, err := txn.Query("DROP TABLE post_views")
	if err != nil {
		txn.Rollback()
	}
}
