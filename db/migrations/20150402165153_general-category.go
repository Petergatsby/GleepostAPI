package main

import (
	"database/sql"
	"log"
)

// Up20150402165153 is executed when this migration is applied
func Up20150402165153(txn *sql.Tx) {
	_, err := txn.Query("INSERT INTO categories (id, tag, name) VALUES (?, ?, ?)", 1, "general", "General")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	//Existing category-1 posts are invalid
	_, err = txn.Exec("DELETE FROM post_categories WHERE category_id = 1")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	res, err := txn.Exec("INSERT INTO post_categories (post_id, category_id) SELECT id, 1 FROM wall_posts WHERE NOT EXISTS (SELECT * FROM post_categories WHERE post_id = wall_posts.id AND category_id >= 5 AND category_id <= 12)")
	if err != nil {
		log.Println(err)
		txn.Rollback()
		return
	}
	affected, err := res.RowsAffected()
	log.Println("Posts affected:", affected, "Error:", err)
}

// Down20150402165153 is executed when this migration is rolled back
func Down20150402165153(txn *sql.Tx) {
	_, err := txn.Query("DELETE FROM post_categories WHERE category_id = 1")
	if err != nil {
		txn.Rollback()
		return
	}
	_, err = txn.Query("DELETE FROM categories WHERE tag = 'general'")
	if err != nil {
		txn.Rollback()
		return
	}
}
