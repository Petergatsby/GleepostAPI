package db

import (
	"database/sql"
	"log"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/go-sql-driver/mysql"
)

/********************************************************************
		User
********************************************************************/

//RegisterUser creates a user with a username (todo:remove), a password hash and an email address.
//They'll be created in an unverified state, and without a full name (which needs to be set separately).
func (db *DB) RegisterUser(user string, hash []byte, email string) (gp.UserID, error) {
	s, err := db.prepare("INSERT INTO users(name, password, email) VALUES (?,?,?)")
	if err != nil {
		return 0, err
	}
	res, err := s.Exec(user, hash, email)
	if err != nil {
		if err, ok := err.(*mysql.MySQLError); ok {
			if err.Number == 1062 {
				return 0, UserAlreadyExists
			}
		}
		return 0, err
	}
	id, _ := res.LastInsertId()
	return gp.UserID(id), nil
}

//SetUserName sets a user's real name.
func (db *DB) SetUserName(id gp.UserID, firstName, lastName string) (err error) {
	s, err := db.prepare("UPDATE users SET firstname = ?, lastname = ? where id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(firstName, lastName, id)
	return
}

//UserChangeTagline sets this user's tagline (obviously enough)
func (db *DB) UserChangeTagline(userID gp.UserID, tagline string) (err error) {
	s, err := db.prepare("UPDATE users SET `desc` = ? WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(tagline, userID)
	return
}

//GetUser returns a gp.User with User.Name set to their firstname if available (username if not).
func (db *DB) GetUser(id gp.UserID) (user gp.User, err error) {
	var av, firstName sql.NullString
	s, err := db.prepare("SELECT id, name, avatar, firstname FROM users WHERE id=?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&user.ID, &user.Name, &av, &firstName)
	log.Println("DB hit: db.GetUser id(user.Name, user.Id, user.Avatar)")
	if err != nil {
		if err == sql.ErrNoRows {
			err = &gp.ENOSUCHUSER
		}
		return
	}
	if av.Valid {
		user.Avatar = av.String
	}
	if firstName.Valid {
		user.Name = firstName.String
	}
	return
}

//GetProfile fetches a user but DOES NOT GET THEIR NETWORK.
func (db *DB) GetProfile(id gp.UserID) (user gp.Profile, err error) {
	var av, desc, firstName, lastName sql.NullString
	s, err := db.prepare("SELECT name, `desc`, avatar, firstname, lastname FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&user.Name, &desc, &av, &firstName, &lastName)
	log.Println("DB hit: getProfile id(user.Name, user.Desc)")
	if err != nil {
		if err == sql.ErrNoRows {
			return user, &gp.ENOSUCHUSER
		}
		return
	}
	if av.Valid {
		user.Avatar = av.String
	}
	if desc.Valid {
		user.Desc = desc.String
	}
	if firstName.Valid {
		user.Name = firstName.String
	}
	if lastName.Valid {
		user.FullName = firstName.String + " " + lastName.String
	}
	user.ID = id
	return
}

//SetProfileImage updates this user's avatar to url.
func (db *DB) SetProfileImage(id gp.UserID, url string) (err error) {
	s, err := db.prepare("UPDATE users SET avatar = ? WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(url, id)
	return
}

//SetBusyStatus records whether this user is busy or not.
func (db *DB) SetBusyStatus(id gp.UserID, busy bool) (err error) {
	s, err := db.prepare("UPDATE users SET busy = ? WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(busy, id)
	return
}

//BusyStatus returns this user's busy status.
func (db *DB) BusyStatus(id gp.UserID) (busy bool, err error) {
	s, err := db.prepare("SELECT busy FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&busy)
	return
}

//UserIDFromFB gets the gleepost user who has fbid associated, or an error if there is none.
func (db *DB) UserIDFromFB(fbid uint64) (id gp.UserID, err error) {
	s, err := db.prepare("SELECT user_id FROM facebook WHERE fb_id = ? AND user_id IS NOT NULL")
	if err != nil {
		return
	}
	err = s.QueryRow(fbid).Scan(&id)
	return
	//TODO: return ENOSUCHUSER instead.
}

//GetEmail returns this user's email address.
func (db *DB) GetEmail(id gp.UserID) (email string, err error) {
	s, err := db.prepare("SELECT email FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&email)
	return
}

//UserWithEmail returns the user whose email this is, or an error if they don't exist.
func (db *DB) UserWithEmail(email string) (id gp.UserID, err error) {
	s, err := db.prepare("SELECT id FROM users WHERE email = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(email).Scan(&id)
	return
}
