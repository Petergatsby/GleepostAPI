package lib

import (
	"errors"
	"fmt"
	"log"
)

//ContactFormRequest records a request for contact and emails it out to someone.
func (api *API) ContactFormRequest(fullName, college, email, phoneNo, ip string) (err error) {
	log.Println("Contact form request from:", ip, "email:", email)
	if len(fullName) < 3 || len(college) < 3 || len(phoneNo) < 6 {
		return errors.New("Invalid input")
	}
	if !looksLikeEmail(email) {
		return InvalidEmail
	}
	err = api.contactFormRequest(fullName, college, email, phoneNo)
	if err != nil {
		return
	}
	body := fmt.Sprintf("Their email address is %s\nand their phone number is %s.", email, phoneNo)
	api.Mail.SendPlaintext("tade@gleepost.com", fmt.Sprintf("%s from %s reached out for contact", fullName, college), body)
	return nil
}

//ChasenRequest emails Tade with the where&when for the michael chasen meeting.
func (api *API) ChasenRequest(where, when string) (err error) {
	log.Println("Chasen request")
	body := fmt.Sprintf("Location: %s\nTime: %s\n", where, when)
	api.Mail.SendPlaintext("tade@gleepost.com", "Michael Chasen has requested a meeting.", body)
	return nil
}

//ContactFormRequest records a request for contact in the db.
func (api *API) contactFormRequest(fullName, college, email, phoneNo string) (err error) {
	q := "INSERT INTO contact_requests(full_name, college, email, phone_no) VALUES (?, ?, ?, ?)"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(fullName, college, email, phoneNo)
	return
}
