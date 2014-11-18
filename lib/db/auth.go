package db

import (
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//GetHash returns this user's password hash (by username).
func (db *DB) GetHash(user string) (hash []byte, id gp.UserID, err error) {
	s, err := db.prepare("SELECT id, password FROM users WHERE email = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(user).Scan(&id, &hash)
	return
}

//GetHashByID returns this user's password hash.
func (db *DB) GetHashByID(id gp.UserID) (hash []byte, err error) {
	s, err := db.prepare("SELECT password FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&hash)
	return
}

//PassUpdate replaces this user's password hash with a new one.
func (db *DB) PassUpdate(id gp.UserID, newHash []byte) (err error) {
	s, err := db.prepare("UPDATE users SET password = ? WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(newHash, id)
	return
}

//SetVerificationToken records a (hopefully random!) verification token for this user.
func (db *DB) SetVerificationToken(id gp.UserID, token string) (err error) {
	s, err := db.prepare("REPLACE INTO `verification` (user_id, token) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(id, token)
	return
}

//VerificationTokenExists returns the user who this verification token belongs to, or an error if there isn't one.
func (db *DB) VerificationTokenExists(token string) (id gp.UserID, err error) {
	s, err := db.prepare("SELECT user_id FROM verification WHERE token = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(token).Scan(&id)
	return
}

//Verify marks a user as verified.
func (db *DB) Verify(id gp.UserID) (err error) {
	s, err := db.prepare("UPDATE users SET verified = 1 WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(id)
	return
}

//IsVerified returns true if this user is verified.
func (db *DB) IsVerified(user gp.UserID) (verified bool, err error) {
	s, err := db.prepare("SELECT verified FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(user).Scan(&verified)
	return
}

//AddPasswordRecovery records a password recovery token for this user.
func (db *DB) AddPasswordRecovery(userID gp.UserID, token string) (err error) {
	s, err := db.prepare("REPLACE INTO password_recovery (token, user) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(token, userID)
	return
}

//CheckPasswordRecovery returns true if this password recovery user:token pair exists.
func (db *DB) CheckPasswordRecovery(userID gp.UserID, token string) (exists bool, err error) {
	s, err := db.prepare("SELECT count(*) FROM password_recovery WHERE user = ? and token = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(userID, token).Scan(&exists)
	return
}

//DeletePasswordRecovery removes this password recovery token so it can't be used again.
func (db *DB) DeletePasswordRecovery(userID gp.UserID, token string) (err error) {
	s, err := db.prepare("DELETE FROM password_recovery WHERE user = ? and token = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(userID, token)
	return
}

/********************************************************************
		Token
********************************************************************/

//TokenExists returns true if this user:token pair exists, false otherwise (or in the case of error)
func (db *DB) TokenExists(id gp.UserID, token string) bool {
	var expiry string
	s, err := db.prepare("SELECT expiry FROM tokens WHERE user_id = ? AND token = ?")
	if err != nil {
		return false
	}
	err = s.QueryRow(id, token).Scan(&expiry)
	if err != nil {
		return false
	}
	t, _ := time.Parse(mysqlTime, expiry)
	if t.After(time.Now()) {
		return (true)
	}
	return (false)
}

//AddToken records this session token in the database.
func (db *DB) AddToken(token gp.Token) (err error) {
	s, err := db.prepare("INSERT INTO tokens (user_id, token, expiry) VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(token.UserID, token.Token, token.Expiry)
	return
}
