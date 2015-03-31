package main

import (
	"database/sql"
	"log"

	"github.com/draaglom/GleepostAPI/lib/conf"
)

//Up20150325171408 is executed when this migration is applied
func Up20150325171408(txn *sql.Tx) {
	config := conf.GetConfig()
	s, err := txn.Prepare("INSERT INTO post_attribs (post_id, attrib, value) VALUES (?, ?, ?)")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	for _, f := range config.Futures {
		_, err = s.Exec(f.Post, "meta-future", f.Future)
		if err != nil {
			log.Println(err)
			txn.Rollback()
			return
		}
	}
}

//Down20150325171408 is executed when this migration is rolled back
func Down20150325171408(txn *sql.Tx) {
	_, err := txn.Query("DELETE FROM post_attribs WHERE attrib = 'meta-future'")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
}
