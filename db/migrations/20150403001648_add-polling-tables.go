package main

import (
	"database/sql"
	"log"
)

// Up20150403001648 is executed when this migration is applied
func Up20150403001648(txn *sql.Tx) {
	q := "CREATE TABLE `post_polls` ( "
	q += "`post_id` int(10) unsigned NOT NULL, "
	q += "`expiry_time` timestamp NOT NULL, "
	q += "PRIMARY KEY (`post_id`) ) "
	q += "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;"
	_, err := txn.Query(q)
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
	q = "CREATE TABLE `poll_options` ( "
	q += "`post_id` int(10) unsigned NOT NULL, "
	q += "`option_id` int(2) unsigned NOT NULL, "
	q += "`option` VARCHAR(50) NOT NULL, "
	q += "UNIQUE KEY `post_option` (`post_id`, `option_id`) ) "
	q += "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
	_, err = txn.Query(q)
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
	q = "CREATE TABLE `poll_votes` ( "
	q += "`post_id` int(10) unsigned NOT NULL, "
	q += "`user_id` int(10) unsigned NOT NULL, "
	q += "`option_id` int(2) unsigned NOT NULL, "
	q += "`vote_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, "
	q += "UNIQUE KEY `vote` (`post_id`, `user_id`) ) "
	q += "ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	_, err = txn.Query(q)
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down20150403001648 is executed when this migration is rolled back
func Down20150403001648(txn *sql.Tx) {
	_, err := txn.Query("DROP TABLE post_polls")
	if err != nil {
		txn.Rollback()
	}
	_, err = txn.Query("DROP TABLE poll_options")
	if err != nil {
		txn.Rollback()
	}
	_, err = txn.Query("DROP TABLE poll_votes")
	if err != nil {
		txn.Rollback()
	}
}
