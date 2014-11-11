package db

//ContactFormRequest records a request for contact in the db.
func (db *DB) ContactFormRequest(fullName, college, email, phoneNo string) (err error) {
	q := "INSERT INTO contact_requests(full_name, college, email, phone_no) VALUES (?, ?, ?, ?)"
	s, err := db.prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(fullName, college, email, phoneNo)
	return
}
