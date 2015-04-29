package main

import "database/sql"

// Up20150421165118 is executed when this migration is applied
func Up20150421165118(txn *sql.Tx) {
	q := "CREATE TABLE `video_jobs` ( "
	q += "`id` int(10) unsigned NOT NULL AUTO_INCREMENT, "
	q += "`parent_id` int(10) unsigned NOT NULL, "
	q += "`source` VARCHAR(255) NOT NULL, "
	q += "`target` VARCHAR(10) NOT NULL, "
	q += "`result` VARCHAR(255) NULL, "
	q += "`rotate` BOOLEAN NOT NULL DEFAULT false, "
	q += "`creation_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, "
	q += "`claim_time` DATETIME NULL, "
	q += "`completion_time` DATETIME NULL, "
	q += "PRIMARY KEY(`id`) ) "
	q += "ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	_, err := txn.Query(q)
	if err != nil {
		txn.Rollback()
		return
	}
}

// Down20150421165118 is executed when this migration is rolled back
func Down20150421165118(txn *sql.Tx) {
	_, err := txn.Exec("DROP TABLE `video_jobs`")
	if err != nil {
		txn.Rollback()
		return
	}
}
