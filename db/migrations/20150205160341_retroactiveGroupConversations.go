package main

import (
	"database/sql"
	"log"
)

//Up_20150205160341 is executed when this migration is applied
func Up_20150205160341(txn *sql.Tx) {
	_, err := txn.Query("INSERT INTO conversations (initiator, last_mod, primary_conversation, group_id) SELECT creator, NOW(), false, id FROM network WHERE creator IS NOT NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
	_, err = txn.Query("INSERT INTO conversation_participants (conversation_id, participant_id) SELECT conversations.id, user_network.user_id FROM conversations JOIN user_network ON conversations.group_id = user_network.network_id")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}

}

//Down_20150205160341 is executed when this migration is rolled back
func Down_20150205160341(txn *sql.Tx) {
	_, err := txn.Query("DELETE FROM conversation_participants WHERE conversation_id IN (SELECT id FROM conversations WHERE group_id IS NOT NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
	_, err = txn.Query("DELETE FROM conversations WHERE group_id IS NOT NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
}
