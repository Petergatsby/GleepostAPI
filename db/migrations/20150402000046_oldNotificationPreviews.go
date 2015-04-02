package main

import (
	"database/sql"
	"log"
)

type postBy struct {
	post      int
	commentBy int
}

type notificationText struct {
	notification int
	text         string
}

// Up20150402000046 is executed when this migration is applied
func Up20150402000046(txn *sql.Tx) {
	//Find all the posts with comment-null notifications
	rows, err := txn.Query("SELECT id, `by`, post_id FROM notifications WHERE type = 'commented' AND preview_text IS NULL ORDER BY post_id ASC")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	//{post:user}:[notification,notification,....]
	commentBag := make(map[postBy][]int)
	for rows.Next() {
		pb := postBy{}
		var nID int
		var postID sql.NullInt64
		err := rows.Scan(&nID, &pb.commentBy, &postID)
		if err != nil {
			log.Println(err)
			txn.Rollback()
			return
		}
		if !postID.Valid {
			log.Println("A notification was missing a post ID")
			txn.Rollback()
			return
		}
		pb.post = int(postID.Int64)
		commentBag[pb] = append(commentBag[pb], nID)
	}
	rows.Close()
	s, err := txn.Prepare("SELECT text FROM post_comments WHERE post_id = ? AND `by` = ?")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	//Now we can iterate through the comments and notifications, and since they're in the same order and none of either are deleted (hopefully??)
	//they'll match up.
	var nts []notificationText
	for pb := range commentBag {
		log.Printf("Getting comments for post:user %d:%d\n", pb.post, pb.commentBy)
		log.Println("We have:", len(commentBag[pb]), "notifications to account for")
		rows, err := s.Query(pb.post, pb.commentBy)
		if err != nil {
			log.Println(err)
			txn.Rollback()
			return
		}
		i := 0
		for rows.Next() {
			log.Println("Getting comment", i)
			if i > len(commentBag[pb])-1 {
				log.Printf("Had more comments than notifications for post:user pair %d:%d\n", pb.post, pb.commentBy)
				continue
			}
			nt := notificationText{notification: commentBag[pb][i]}
			err = rows.Scan(&nt.text)
			if err != nil {
				log.Println(err)
				txn.Rollback()
				return
			}
			if len(nt.text) >= 100 {
				nt.text = nt.text[:97] + "..."
			}
			nts = append(nts, nt)
			i++
		}
		rows.Close()
	}
	log.Println("Finished getting comment texts")
	setPreview, err := txn.Prepare("UPDATE notifications SET preview_text = ? WHERE id = ?")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	for _, nt := range nts {
		log.Printf("Setting notification %d to %s\n", nt.notification, nt.text)
		_, err = setPreview.Exec(nt.text, nt.notification)
		if err != nil {
			log.Println(err)
			txn.Rollback()
			return
		}

	}
	//Finally, for the notifications we can't work out...
	_, err = txn.Exec("UPDATE notifications SET preview_text = '...' WHERE type = 'commented' AND preview_text IS NULL")
	if err != nil {
		log.Println(err)
		txn.Rollback()
	}
	return
}

// Down20150402000046 is executed when this migration is rolled back
func Down20150402000046(txn *sql.Tx) {
	//There actually isn't anything to do here:
	//There's no real way to know if the comment was previously NULL or not.

}
