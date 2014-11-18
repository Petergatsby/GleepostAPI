package lib

import "github.com/draaglom/GleepostAPI/lib/gp"

//GetContacts returns all contacts (incl. those who have not yet accepted) for this user.
func (api *API) GetContacts(user gp.UserID) (contacts []gp.Contact, err error) {
	return api.db.GetContacts(user)
}

//AreContacts returns true if a and b are (confirmed) contacts.
//TODO: Implement a proper db-level version
func (api *API) AreContacts(a, b gp.UserID) (areContacts bool, err error) {
	contacts, err := api.GetContacts(a)
	if err != nil {
		return
	}
	for _, c := range contacts {
		if c.ID == b && c.YouConfirmed && c.TheyConfirmed {
			return true, nil
		}
	}
	return false, nil
}

//AddContact sends a contact request from adder to addee.
func (api *API) AddContact(adder gp.UserID, addee gp.UserID) (contact gp.Contact, err error) {
	user, err := api.GetUser(addee)
	if err != nil {
		return
	}
	exists, err := api.ContactRequestExists(addee, adder)
	if err != nil {
		return
	}
	if exists {
		return api.AcceptContact(adder, addee)
	}
	err = api.db.AddContact(adder, addee)
	if err == nil {
		go api.createNotification("added_you", adder, addee, 0)
	}
	contact.User = user
	contact.YouConfirmed = true
	contact.TheyConfirmed = false
	return
}

//ContactRequestExists returns true if adder has previously added addee (whether they have accepted or not).
func (api *API) ContactRequestExists(adder gp.UserID, addee gp.UserID) (exists bool, err error) {
	return api.db.ContactRequestExists(adder, addee)
}

//AcceptContact marks this request as accepted - these users are now contacts.
func (api *API) AcceptContact(user gp.UserID, toAccept gp.UserID) (contact gp.Contact, err error) {
	err = api.db.UpdateContact(user, toAccept)
	if err != nil {
		return
	}
	contact.User, err = api.GetUser(toAccept)
	if err != nil {
		return
	}
	contact.YouConfirmed = true
	contact.TheyConfirmed = true
	go api.createNotification("accepted_you", user, toAccept, 0)
	go api.UnExpireBetween([]gp.UserID{user, toAccept})
	return
}
