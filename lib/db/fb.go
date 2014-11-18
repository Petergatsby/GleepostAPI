package db

import (
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//CreateFBUser records the existence of this (fbid:email) pair; when the user is verified it will be converted to a full gleepost user.
func (db *DB) CreateFBUser(fbID uint64, email string) (err error) {
	s, err := db.prepare("INSERT INTO facebook (fb_id, email) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(fbID, email)
	return
}

//FBUserEmail returns this facebook user's email address.
func (db *DB) FBUserEmail(fbid uint64) (email string, err error) {
	s, err := db.prepare("SELECT email FROM facebook WHERE fb_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(fbid).Scan(&email)
	return
}

//FBUserWithEmail returns the facebook id we've seen associated with this email, or error if none exists.
func (db *DB) FBUserWithEmail(email string) (fbid uint64, err error) {
	s, err := db.prepare("SELECT fb_id FROM facebook WHERE email = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(email).Scan(&fbid)
	return
}

//CreateFBVerification records a (hopefully random!) verification token for this facebook user.
func (db *DB) CreateFBVerification(fbid uint64, token string) (err error) {
	s, err := db.prepare("REPLACE INTO facebook_verification (fb_id, token) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(fbid, token)
	return
}

//FBVerificationExists returns the user this verification token is for, or an error if there is none.
func (db *DB) FBVerificationExists(token string) (fbid uint64, err error) {
	s, err := db.prepare("SELECT fb_id FROM facebook_verification WHERE token = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(token).Scan(&fbid)
	return
}

//FBSetGPUser records the association of this facebook user with this gleepost user.
//After this, the user should be able to log in with this facebook account.
func (db *DB) FBSetGPUser(fbid uint64, userID gp.UserID) (err error) {
	fbSetGPUser := "REPLACE INTO facebook (user_id, fb_id) VALUES (?, ?)"
	stmt, err := db.prepare(fbSetGPUser)
	if err != nil {
		return
	}
	res, err := stmt.Exec(userID, fbid)
	log.Println(res.RowsAffected())
	return
}
