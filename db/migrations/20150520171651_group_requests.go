package main

import (
	"database/sql"
)

// Up20150520171651 is executed when this migration is applied
func Up20150520171651(txn *sql.Tx) {
	q := "CREATE TABLE `network_requests` ( "
	q += "`user_id` int(10) unsigned NOT NULL, "
	q += "`network_id` int(10) unsigned NOT NULL, "
	q += "`request_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, "
	q += "`status` varchar(10) NOT NULL DEFAULT 'pending', "
	q += "`update_time` datetime NULL, "
	q += "`processed_by` int(10) unsigned NULL, "
	q += "CONSTRAINT user_net UNIQUE (user_id, network_id) ) "
	q += "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;"
	_, err := txn.Query(q)
	if err != nil {
		txn.Rollback()
	}

}

// Down20150520171651 is executed when this migration is rolled back
func Down20150520171651(txn *sql.Tx) {
	_, err := txn.Query("DROP TABLE network_requests")
	if err != nil {
		txn.Rollback()
	}
}
