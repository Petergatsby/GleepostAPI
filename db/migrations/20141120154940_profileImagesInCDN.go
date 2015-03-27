package main

import (
	"database/sql"
)

//Up_20141120154940 is executed when this migration is applied
func Up_20141120154940(txn *sql.Tx) {
	_, err := txn.Query("UPDATE users SET avatar = CONCAT('http://d2tc2ce3464r63.cloudfront.net', SUBSTR(avatar, 41)) WHERE avatar LIKE '%gpimg%'")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("UPDATE users SET avatar = CONCAT('http://d3itv2rmlfeij9.cloudfront.net', SUBSTR(avatar, 42)) WHERE avatar LIKE '%gpcali%'")
	if err != nil {
		txn.Rollback()
		return
	}
}

//Down_20141120154940 is executed when this migration is rolled back
func Down_20141120154940(txn *sql.Tx) {
	_, err := txn.Query("UPDATE users SET avatar = CONCAT('https://s3-eu-west-1.amazonaws.com/gpimg', SUBSTR(avatar, 37)) WHERE avatar LIKE '%d2tc2ce3464r63%'")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("UPDATE users SET avatar = CONCAT('https://s3-us-west-1.amazonaws.com/gpcali', SUBSTR(avatar, 37)) WHERE avatar LIKE '%d3itv2rmlfeij9%'")
	if err != nil {
		txn.Rollback()
		return
	}
}
