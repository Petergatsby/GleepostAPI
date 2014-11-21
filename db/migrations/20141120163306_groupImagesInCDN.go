package main

import (
	"database/sql"
)

// Up is executed when this migration is applied
func Up_20141120163306(txn *sql.Tx) {
	_, err := txn.Query("UPDATE network SET cover_img = CONCAT('http://d2tc2ce3464r63.cloudfront.net', SUBSTR(cover_img, 41)) WHERE cover_img LIKE '%gpimg%'")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("UPDATE network SET cover_img = CONCAT('http://d3itv2rmlfeij9.cloudfront.net', SUBSTR(cover_img, 42)) WHERE cover_img LIKE '%gpcali%'")
	if err != nil {
		txn.Rollback()
		return
	}
}

// Down is executed when this migration is rolled back
func Down_20141120163306(txn *sql.Tx) {
	_, err := txn.Query("UPDATE network SET cover_img = CONCAT('https://s3-eu-west-1.amazonaws.com/gpimg', SUBSTR(cover_img, 37)) WHERE cover_img LIKE '%d2tc2ce3464r63%'")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("UPDATE network SET cover_img = CONCAT('https://s3-us-west-1.amazonaws.com/gpcali', SUBSTR(cover_img, 37)) WHERE cover_img LIKE '%d3itv2rmlfeij9%'")
	if err != nil {
		txn.Rollback()
		return
	}
}
