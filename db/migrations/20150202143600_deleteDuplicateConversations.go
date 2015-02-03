package main

import (
	"database/sql"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

// Up is executed when this migration is applied
func Up_20150202143600(txn *sql.Tx) {
	//Merge all duplicate conversations between user pairs into one
	log.Println("Retrieving all 2-person conversations")
	rows, err := txn.Query("SELECT conversation_id, participant_id, conversations.last_mod FROM conversation_participants JOIN conversations ON conversation_id = conversations.id WHERE primary_conversation = 1 ORDER BY conversation_id ASC")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	//Build time-ordered list of conversations & participants
	var conversations []gp.Conversation
	for rows.Next() {
		var conversation gp.Conversation
		var participant gp.User
		var t string
		err = rows.Scan(&conversation.ID, &participant.ID, &t)
		if err != nil {
			log.Println(err)
			txn.Rollback()
			return
		}
		lastActivity, _ := time.Parse("2006-01-02 15:04:05", t)
		if len(conversations) > 0 && conversation.ID == conversations[len(conversations)-1].ID {
			//Add this participant to the last conversation (since it's the same one)
			conversations[len(conversations)-1].Participants = append(conversations[len(conversations)-1].Participants, participant)
		} else {
			//Add the new conversation to the list
			conversation.Participants = append(conversation.Participants, participant)
			conversation.LastActivity = lastActivity
			conversations = append(conversations, conversation)
		}
	}
	log.Println("Got", len(conversations), "conversations.")
	rows.Close()
	log.Println("Preparing: re-conversate messages")
	changeMessagesConvStmt, err := txn.Prepare("UPDATE chat_messages SET conversation_id = ? WHERE conversation_id = ?")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	log.Println("Preparing: update last_mod")
	updateLastModStmt, err := txn.Prepare("UPDATE conversations SET last_mod = ? WHERE id = ?")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	log.Println("Preparing: set merged")
	mergeConversationStmt, err := txn.Prepare("UPDATE conversations SET merged = ? WHERE id = ?")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	log.Println("Preparing: delete particip")
	deleteParticipantsStmt, err := txn.Prepare("DELETE FROM conversation_participants WHERE conversation_id = ?")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}

	merged := make(map[gp.ConversationID]bool)
	//Identify duplicates
	//Merge newer conversations into older
	count := 0
	for _, c := range conversations {
		_, ok := merged[c.ID]
		if !ok {
			for _, d := range conversations {
				switch {
				case c.Participants[0].ID == d.Participants[0].ID && c.Participants[1].ID == d.Participants[1].ID && c.ID < d.ID:
					fallthrough
				case c.Participants[0].ID == d.Participants[1].ID && c.Participants[1].ID == d.Participants[0].ID && c.ID < d.ID:
					log.Println("Found dupe convs:", c.ID, d.ID)

					merged[d.ID] = true
					//Take messages from d to c
					log.Println("Changing messages convs")
					_, err = changeMessagesConvStmt.Exec(c.ID, d.ID)
					if err != nil {
						log.Println(err)
						txn.Rollback()
						return
					}
					//Update to more recent activity
					if d.LastActivity.After(c.LastActivity) {
						log.Println("Updating lastActivity")
						_, err = updateLastModStmt.Exec(c.ID, d.LastActivity)
						if err != nil {
							log.Println(err)
							txn.Rollback()
							return
						}
					}
					//Delete d
					log.Println("Merging", d.ID)
					_, err = mergeConversationStmt.Exec(c.ID, d.ID)
					if err != nil {
						log.Println(err)
						txn.Rollback()
						return
					}
					log.Println("Deleting participants", d.ID)
					_, err = deleteParticipantsStmt.Exec(d.ID)
					if err != nil {
						log.Println(err)
						txn.Rollback()
						return
					}
					count++

				default:
				}
			}
		}
	}
	log.Println("Merged:", count, "of:", len(conversations))
}

// Down is executed when this migration is rolled back
func Down_20150202143600(txn *sql.Tx) {

}
