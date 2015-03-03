package lib

import "fmt"

//ContactFormRequest records a request for contact and emails it out to someone.
func (api *API) ContactFormRequest(fullName, college, email, phoneNo string) (err error) {
	err = api.db.ContactFormRequest(fullName, college, email, phoneNo)
	if err != nil {
		return
	}
	body := fmt.Sprintf("Their email address is %s\nand their phone number is %s.", email, phoneNo)
	api.mail.SendPlaintext("tade@gleepost.com", fmt.Sprintf("%s from %s reached out for contact", fullName, college), body)
	return nil
}
