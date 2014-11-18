package db

import (
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//AddContact records that adder has added addee as a contact.
func (db *DB) AddContact(adder gp.UserID, addee gp.UserID) (err error) {
	log.Println("DB hit: addContact")
	s, err := db.prepare("INSERT INTO contacts (adder, addee) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(adder, addee)
	return
}

//GetContacts retrieves all the contacts for user.
//TODO: This could return contacts which doesn't embed a user
func (db *DB) GetContacts(user gp.UserID) (contacts []gp.Contact, err error) {
	contacts = make([]gp.Contact, 0)
	s, err := db.prepare("SELECT adder, addee, confirmed FROM contacts WHERE adder = ? OR addee = ? ORDER BY time DESC")
	if err != nil {
		return
	}
	rows, err := s.Query(user, user)
	log.Println("DB hit: db.GetContacts")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var contact gp.Contact
		var adder, addee gp.UserID
		var confirmed bool
		err = rows.Scan(&adder, &addee, &confirmed)
		if err != nil {
			return
		}
		switch {
		case adder == user:
			contact.User, err = db.GetUser(addee)
			if err == nil {
				contact.YouConfirmed = true
				contact.TheyConfirmed = confirmed
				contacts = append(contacts, contact)
			} else {
				log.Println(err)
			}
		case addee == user:
			contact.User, err = db.GetUser(adder)
			if err == nil {
				contact.YouConfirmed = confirmed
				contact.TheyConfirmed = true
				contacts = append(contacts, contact)
			} else {
				log.Println(err)
			}
		}
	}
	return contacts, nil
}

//UpdateContact marks this adder/addee pair as "accepted"
func (db *DB) UpdateContact(user gp.UserID, contact gp.UserID) (err error) {
	s, err := db.prepare("UPDATE contacts SET confirmed = 1 WHERE addee = ? AND adder = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(user, contact)
	return
}

//ContactRequestExists returns true if this adder has already added addee.
func (db *DB) ContactRequestExists(adder gp.UserID, addee gp.UserID) (exists bool, err error) {
	s, err := db.prepare("SELECT COUNT(*) FROM contacts WHERE adder = ? AND addee = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(adder, addee).Scan(&exists)
	return
}
