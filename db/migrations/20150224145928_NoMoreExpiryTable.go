package main

import (
	"database/sql"
	"log"
)

//Up_20150224145928 is executed when this migration is applied
func Up_20150224145928(txn *sql.Tx) {
	_, err := txn.Query("DELETE FROM chat_messages WHERE conversation_id IN (SELECT conversation_id FROM conversation_expirations)")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	_, err = txn.Query("DELETE FROM conversations WHERE id IN (SELECT conversation_id FROM conversation_expirations)")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	_, err = txn.Query("DROP TABLE conversation_expirations")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
}

// Down is executed when this migration is rolled back
func Down_20150224145928(txn *sql.Tx) {
	q := "CREATE TABLE IF NOT EXISTS `conversation_expirations` ( " +
		"`conversation_id` int(10) unsigned NOT NULL, " +
		"`expiry` datetime NOT NULL, " +
		"`ended` tinyint(1) NOT NULL DEFAULT '0', " +
		"UNIQUE KEY `c_id` (`conversation_id`) " +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	_, err := txn.Query(q)
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
}
