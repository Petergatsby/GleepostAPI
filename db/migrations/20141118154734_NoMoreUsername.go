package main

import (
	"database/sql"
	"log"
)

//Up_20141118154734 is executed when this migration is applied
func Up_20141118154734(txn *sql.Tx) {
	_, err := txn.Query("UPDATE users SET firstname = name WHERE firstname IS NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	_, err = txn.Query("ALTER TABLE users DROP COLUMN name")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	_, err = txn.Query("ALTER TABLE users MODIFY COLUMN firstname VARCHAR(64) NOT NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	_, err = txn.Query("ALTER TABLE users ALTER COLUMN firstname DROP DEFAULT")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}

}

// Down is executed when this migration is rolled back
func Down_20141118154734(txn *sql.Tx) {
	_, err := txn.Query("ALTER TABLE users ADD name VARCHAR(255) NOT NULL DEFAULT 'unknown_user' BEFORE password")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("UPDATE users SET name = firstname")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("ALTER TABLE users ALTER COLUMN firstname VARCHAR NULL")
	if err != nil {
		txn.Rollback()
		return
	}
}
