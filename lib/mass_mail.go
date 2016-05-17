package lib

import (
	"log"

	"github.com/Petergatsby/GleepostAPI/lib/gp"
)

//Massmail sends a standard email to all users. Probably just use MailChimp instead, though.
func (api *API) Massmail(userID gp.UserID) (err error) {
	if !api.isAdmin(userID) {
		return ENOTALLOWED
	}
	subject := "FREE REDBULL STUDYGRAMS AT TRESIDDER AND GREEN LIBRARY!"
	body := `<html><body>Hey guys!<br><br>

It’s dead week and we all know that we’re all stressing over exams! You've helped us move toward our goals, so we want to give you the energy to reach yours (or maybe just get through the week). In less than 3 weeks, almost 1500 Stanford students have downloaded and started using Gleepost! To show our gratitude, we’re giving away FREE REDBULL. That’s right! Absolutely Free Red Bull! Just like the Free Red Bull Giveaway event on Gleepost <a href="https://gleepost.com/studygram?r=ec1">here</a>, so we can be sure to have enough, and come and collect your free Red Bull at Tresidder, or outside Green Library tonight or tomorrow night, from 8:30pm to 10pm!<br><br>

See you there and Good luck with exams!<br><br> 

Curing FOMO one day at a time, The Gleepost Team.</body></html>`
	emails, err := api.allEmails()
	if err != nil {
		log.Println(err)
		return
	}
	count := 0
	for _, email := range emails {
		err = api.Mail.SendHTML(email, subject, body)
		if err != nil {
			log.Println(err)
		} else {
			count++
			log.Println("Sent mails:", count)
		}
	}
	return
}

//AllEmails returns all registered emails.
func (api *API) allEmails() (emails []string, err error) {
	s, err := api.sc.Prepare("SELECT email FROM users")
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
