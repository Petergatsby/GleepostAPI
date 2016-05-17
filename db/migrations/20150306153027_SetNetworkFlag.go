package main

import (
	"database/sql"
	"log"

	"github.com/Petergatsby/GleepostAPI/lib/conf"
)

//Up20150306153027 is executed when this migration is applied
//Must be executed at commit bb57bd7aedf58c687d78318ab47b36a8d0a75bdf or before (before admins were deleted from conf)
func Up20150306153027(txn *sql.Tx) {
	conf := conf.GetConfig()
	s, err := txn.Prepare("UPDATE users SET is_admin = 1 WHERE id IN (SELECT user_id FROM user_network WHERE network_id = ?)")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	_, err = s.Exec(conf.Admins)
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
}

//Down20150306153027 is executed when this migration is rolled back
func Down20150306153027(txn *sql.Tx) {
	_, err := txn.Query("UPDATE users SET is_admin = 0")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
}
