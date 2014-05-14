package db

//AllEmails returns all registered emails.
func (db *DB) AllEmails() (emails []string, err error) {
	s, err := db.prepare("SELECT email FROM users")
	if err != nil {
		return
	}
	rows, err := s.Query()
	if err != nil {
		return
	}
	for rows.Next() {
		var email string
		err = rows.Scan(&email)
		if err != nil {
			return
		}
		emails = append(emails, email)
	}
	return
}
