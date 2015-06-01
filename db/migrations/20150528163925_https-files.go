package main

import (
	"database/sql"
	"log"
)

// Up20150528163925 is executed when this migration is applied
func Up20150528163925(txn *sql.Tx) {
	_, err := txn.Exec("UPDATE uploads SET `url` = CONCAT('https:', SUBSTR(`url`, 6)) WHERE `url` LIKE 'http:%'")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
	_, err = txn.Exec("UPDATE uploads SET `webm_url` = CONCAT('https:', SUBSTR(`webm_url`, 6)) WHERE `webm_url` LIKE 'http:%'")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
	_, err = txn.Exec("UPDATE uploads SET `mp4_url` = CONCAT('https:', SUBSTR(`mp4_url`, 6)) WHERE `mp4_url` LIKE 'http:%'")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
	_, err = txn.Exec("UPDATE users SET `avatar` = CONCAT('https:', SUBSTR(`avatar`, 6)) WHERE `avatar` LIKE 'http:%'")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
	_, err = txn.Exec("UPDATE network SET `cover_img` = CONCAT('https:', SUBSTR(`cover_img`, 6)) WHERE `cover_img` LIKE 'http:%'")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
	_, err = txn.Exec("UPDATE post_images SET `url` = CONCAT('https:', SUBSTR(`url`, 6)) WHERE `url` LIKE 'http:%'")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}

// Down20150528163925 is executed when this migration is rolled back
func Down20150528163925(txn *sql.Tx) {

}
