package lib

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/Petergatsby/GleepostAPI/lib/conf"
	"github.com/Petergatsby/GleepostAPI/lib/gp"
	"github.com/draaglom/facebook"
)

//FacebookToken contains the parsed expiry, user and permission scopes of a facebook authentication token.
type FacebookToken struct {
	Expiry time.Time `facebook:"expires_at"`
	FBUser uint64    `facebook:"user_id"`
	Scopes []string  `facebook:"scopes"`
}

//FB contains the configuration specific to this facebook app.
type FB struct {
	config conf.FacebookConfig
}

var (
	//FBAPIError is a catchall error for anything that went wrong with a facebook reqest.
	FBAPIError = gp.APIerror{Reason: "Something went wrong with a facebook API call."}
	//AlreadyAssociated means you're trying to connect a facebook account to a second gleepost account.
	AlreadyAssociated = gp.APIerror{Reason: "Facebook account already associated with another gleepost account..."}
	//BadFBToken means the facebook token provided is invalid in some way.
	BadFBToken = gp.APIerror{Reason: "Bad token"}
	//FBNoEmail = the email supplied by the user is too short.
	FBNoEmail = gp.APIerror{Reason: "Email required"}
)

//debugToken logs the response from facebook's /debug_token.
func debugToken(token string) {
	res, err := facebook.Get("/debug_token", facebook.Params{
		"access_token": token,
	})
	fmt.Println(res["app_id"])
	fmt.Println(res["expires_at"])
	fmt.Println(res["is_valid"])
	fmt.Printf("%v", res["scopes"])
	fmt.Printf("%v", res)
	fmt.Printf("%v", err)
}

//fBValidateToken takes a client-supplied facebook access token and returns a FacebookToken, or an error if the token is invalid in some way
//ie, expired or for another app.
func (api *API) fBValidateToken(fbToken string, retries int) (token FacebookToken, err error) {
	app := facebook.New(api.fb.config.AppID, api.fb.config.AppSecret)
	appToken := app.AppAccessToken()
	res, err := facebook.Get("/debug_token", facebook.Params{
		"access_token": appToken,
		"input_token":  fbToken,
	})
	if err != nil {
		if _, ok := err.(net.Error); ok && retries > 0 {
			//Probably a transient connection error, we can go again.
			<-time.After(3 * time.Second)
			token, err = api.fBValidateToken(fbToken, retries-1)
		} else {
			log.Println("Couldn't retry:", err)
		}
		return
	}
	data := res["data"].(map[string]interface{})
	tokenappid, ok := data["app_id"].(string)
	if !ok {
		log.Println("Couldn't cast data[\"app_id\"] as string")
	}
	if api.fb.config.AppID != tokenappid {
		fmt.Println("App id doesn't match")
		return token, gp.APIerror{Reason: "Bad facebook token"}
	}
	expiry := time.Unix(int64(data["expires_at"].(float64)), 0)
	if !expiry.After(time.Now()) {
		fmt.Println("Token expired already")
		return token, gp.APIerror{Reason: "Bad facebook token"}
	}
	var valid bool
	valid = data["is_valid"].(bool)
	if !valid {
		fmt.Println("Token isn't valid")
		return token, gp.APIerror{Reason: "Bad facebook token"}
	}
	token.Expiry = expiry
	token.FBUser, err = strconv.ParseUint(data["user_id"].(string), 10, 64)
	if err != nil {
		return token, err
	}
	scopes := data["scopes"].([]interface{})
	for _, scope := range scopes {
		token.Scopes = append(token.Scopes, scope.(string))
	}
	return
}

//FacebookLogin takes a facebook access token supplied by a user and tries to issue a gleepost session token,
// or an error if there isn't an associated gleepost user for this facebook account.
//As long as err != BadToken, the user's fbid is returned.
func (api *API) FacebookLogin(fbToken, email, invite string) (token gp.Token, FBUser uint64, status gp.Status, err error) {
	t, err := api.fBValidateToken(fbToken, 2)
	if err != nil {
		err = BadFBToken
		return
	}
	FBUser = t.FBUser
	userID, err := api.fBGetGPUser(t.FBUser)
	switch {
	case err == nil:
		err = api.updateFBData(fbToken)
		if err != nil {
			log.Println("Error pulling in profile changes from facebook:", err)
		}
		token, err = api.Auth.createAndStoreToken(userID)
		return
	case err == NoSuchUser: //No gleepost user already associated with this fb user.
		//If we have an error here, that means that there is no associated gleepost user account.
		log.Println("Error logging in with facebook, probably means there's no associated gleepost account:", err)
		//Did the user provide an email (takes precedence over stored email, because they might have typo'd the first time)
		var storedEmail string
		storedEmail, err = api.fBGetEmail(FBUser)
		switch {
		//Has this email been seen before for this user?
		case len(email) > 3 && (err != nil || storedEmail != email):
			//Either we don't have a stored email for this user, or at least it wasn't this one.
			//(So we should check if there's an existing signed up / verified user)
			//(and if not, issue a verification email)
			//(since this is the first time they've signed up with this email)
			token, status, err = api.fBFirstTimeWithEmail(email, fbToken, invite, FBUser)
			return
		case len(email) > 3 && (err == nil && (storedEmail == email)):
			//We already saw this user, so we don't need to re-send verification
			fallthrough
		case len(email) < 3 && (err == nil):
			//We already saw this user, so we don't need to re-send verification
			//So it should be "unverified" or "registered" as appropriate
			_, err = api.userWithEmail(storedEmail)
			if err != nil {
				log.Println("Should be unverified response")
				status = gp.NewStatus("unverified", storedEmail)
				return token, FBUser, status, nil
			}
			status = gp.NewStatus("registered", storedEmail)
			return
		case len(email) < 3 && (err != nil):
			err = FBNoEmail
			return
		}
		return //Don't think this branch is reachable.
	default: //Server error
		return
	}
}

//UpdateFBData is a placeholder for the time being. In the future, place anything which needs to be regularly checked from facebook here.
func (api *API) updateFBData(fbToken string) (err error) {
	return nil
}

//FBGetGPUser returns the associated gleepost user for a given facebook id, or sql.ErrNoRows if that user doesn't exist.
//TODO: Change to ENOSUCHUSER
func (api *API) fBGetGPUser(fbid uint64) (id gp.UserID, err error) {
	id, err = api.userIDFromFB(fbid)
	return
}

//FacebookRegister takes a facebook access token, an email and an (optional) invite key.
//If the email/invite pair is valid, it will associate this facebook account with the owner of this
// email address, or create a gleepost account as appropriate.
//If the invite is invalid or nonexistent, it issues a verification email
//(the rest of the association will be handled upon verification in FBVerify.
//It will either return a token (meaning that the user has logged in successfully) or a verification status (meaning the user should verify their email).
func (api *API) FacebookRegister(fbToken string, email string, invite string) (token gp.Token, verification gp.Status, err error) {
	t, err := api.fBValidateToken(fbToken, 3)
	if err != nil {
		return
	}
	err = api.createFBUser(t.FBUser, email, fbToken)
	exists, _ := api.inviteExists(email, invite)
	if exists {
		id, e := api.fBSetVerified(email, t.FBUser)
		if e != nil {
			err = e
			return
		}
		token, err = api.Auth.createAndStoreToken(id)
		return
	}
	if err == nil {
		err = api.FBissueVerification(t.FBUser)
	}
	verification = gp.NewStatus("unverified", email)
	return
}

//FBSetVerified creates a gleepost user for this fbuser, or associates with an existing one as appropriate.
func (api *API) fBSetVerified(email string, fbuser uint64) (id gp.UserID, err error) {
	id, err = api.userWithEmail(email)
	if err != nil {
		log.Println("There isn't a user with this facebook email")
		id, err = api.createUserFromFB(fbuser, email)
		return
	}
	err = api.userSetFB(id, fbuser)
	if err == nil {
		err = api.verify(id)
		if err == nil {
			log.Println("Verifying worked. Now setting networks from invites...")
			err = api.acceptAllInvites(id, email)
		}
	}
	return
}

//FBissueVerification creates and sends a verification email for this facebook user, or returns an error if we haven't seen them before (ie, we don't have their email address on file)
//TODO: Think about decoupling this from the email check
func (api *API) FBissueVerification(fbid uint64) (err error) {
	email, err := api.fBGetEmail(fbid)
	if err != nil {
		return
	}
	random, err := randomString()
	if err != nil {
		return
	}
	err = api.createFBVerification(fbid, random)
	if err != nil {
		return
	}
	fbtoken, err := api.fbToken(fbid)
	if err != nil {
		return
	}
	firstName, _, _, err := fBName(fbid, fbtoken)
	if err != nil {
		return
	}
	err = api.issueVerificationEmail(email, firstName, random)
	return
}

func (api *API) fbToken(fbid uint64) (fbtoken string, err error) {
	s, err := api.sc.Prepare("SELECT fb_token FROM facebook WHERE fb_id = ?")
	if err != nil {
		return
	}
	var token sql.NullString
	err = s.QueryRow(fbid).Scan(&token)
	if token.Valid {
		fbtoken = token.String
	} else {
		err = errors.New("No stored fb token")
	}
	return

}

//FBName retrieves the first-, last-, and username of facebook id fbid.
func fBName(fbid uint64, token string) (firstName, lastName, username string, err error) {
	res, err := facebook.Get(fmt.Sprintf("/%d", fbid), facebook.Params{"access_token": token})
	var ok bool
	firstName, ok = res["first_name"].(string)
	if !ok {
		err = &FBAPIError
	}
	lastName, ok = res["last_name"].(string)
	if !ok {
		err = &FBAPIError
	}
	username, ok = res["username"].(string)
	if !ok {
		username = ""
	}
	return firstName, lastName, username, err
}

//FBAvatar constructs the facebook graph url for the profile picture of a given facebook username/id
func fBAvatar(username string) (avatar string) {
	return fmt.Sprintf("https://graph.facebook.com/%s/picture?type=large", username)
}

//FBGetEmail returns the email address we have on file for this facebook id, or an error if we don't have one.
func (api *API) fBGetEmail(fbid uint64) (email string, err error) {
	s, err := api.sc.Prepare("SELECT email FROM facebook WHERE fb_id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(fbid).Scan(&email)
	return
}

//UserSetFB sets the associated facebook account for the gleepost user userID.
func (api *API) userSetFB(userID gp.UserID, fbid uint64) (err error) {
	fbSetGPUser := "REPLACE INTO facebook (user_id, fb_id) VALUES (?, ?)"
	stmt, err := api.sc.Prepare(fbSetGPUser)
	if err != nil {
		return
	}
	res, err := stmt.Exec(userID, fbid)
	log.Println(res.RowsAffected())
	return
}

//FBUserWithEmail returns the facebook ID for the user who owns email, or an error if we don't know about that email.
func (api *API) fBUserWithEmail(email string) (fbid uint64, err error) {
	return api.fBUserWithEmail(email)
}

//UserAddFBUsersToGroup takes a list of facebook users and records that they've been invited to the group netID by userID
func (api *API) UserAddFBUsersToGroup(userID gp.UserID, fbusers []uint64, netID gp.NetworkID) (count int, err error) {
	for _, u := range fbusers {
		err = api.userAddFBUserToGroup(userID, u, netID)
		if err == nil {
			count++
		} else {
			return
		}
	}
	return
}

//CreateUserFromFB takes a facebook id and an email address and creates a gleepost user, returning their newly created id.
func (api *API) createUserFromFB(fbid uint64, email string) (userID gp.UserID, err error) {
	fbtoken, err := api.fbToken(fbid)
	if err != nil {
		log.Println("Couldn't get stored facebook token", err)
		return
	}
	firstName, lastName, username, err := fBName(fbid, fbtoken)
	if err != nil {
		log.Println("Couldn't get name info from facebook:", err)
		return
	}
	random, err := randomString()
	if err != nil {
		return
	}
	userID, err = api.createUser(firstName, lastName, random, email)
	if err != nil {
		log.Println("Something went wrong while creating the user from facebook:", err)
		return
	}
	_, err = api.assignNetworks(userID, email)
	if err != nil {
		return
	}
	err = api.setProfileImage(userID, fBAvatar(username))
	if err != nil {
		log.Println("Problem setting avatar:", err)
	}
	err = api.verify(userID)
	if err != nil {
		log.Println("Verifying failed in the db:", err)
		return
	}
	err = api.userSetFB(userID, fbid)
	if err != nil {
		log.Println("associating facebook account with user account failed:", err)
		return
	}
	err = api.acceptAllInvites(userID, email)
	if err != nil {
		log.Println("Something went wrong while accepting invites:", err)
		return
	}
	err = api.assignNetworksFromFBInvites(userID, fbid)
	if err != nil {
		log.Println("Something went wrong while setting networks from fb invites:", err)
		return
	}
	err = api.acceptAllFBInvites(fbid)
	return

}

//AttemptLoginWithInvite tries to perform a facebook login with an invite code sent over email.
//This will implicitly verify an account (because they have to have access to that email) and issue a session token if the invite is valid.
//If the invite is not valid, returns status - registered.
//(why?? I can't remember.)
func (api *API) AttemptLoginWithInvite(email, invite string, FBUser uint64) (token gp.Token, status gp.Status, err error) {
	exists, _ := api.inviteExists(email, invite)
	if exists {
		//Verify
		id, e := api.fBSetVerified(email, FBUser)
		if e != nil {
			err = e
			return
		}
		//Login
		token, err = api.Auth.createAndStoreToken(id)
		if err != nil {
			return
		}
	}
	status = gp.NewStatus("registered", email)

	return
}

//AttemptAssociationWithCredentials tries to connect a particular facebook account to a particular user account.
func (api *API) AttemptAssociationWithCredentials(email, pass, fbToken string) (err error) {
	id, err := api.Auth.validatePass(email, pass)
	if err != nil {
		log.Println(err)
		err = BadLogin
		return
	}
	err = api.AssociateFB(id, fbToken)
	return
}

//AssociateFB tries to connect the facebook account encoded in this facebook token to this gleepost account.
//It will return BadFBToken if the token doesn't validate; and AlreadyAssociated if this facebook account is already associated with a different gleepost account.
func (api *API) AssociateFB(id gp.UserID, fbToken string) (err error) {
	//Ignore status for now - TODO(patrick): what does this imply
	token, fbuser, _, err := api.FacebookLogin(fbToken, "", "")
	switch {
	case err == BadFBToken:
		return
	case err != nil:
		//This isn't associated with a gleepost account
		err = api.userSetFB(id, fbuser)
		return
	case token.UserID == id:
		//The facebook account is already associated with this gleepost account
		return nil
	default:
		return AlreadyAssociated
	}
}

//FBFirstTimeWithEmail will create a fresh association with this fb:email pair. If there is no existing gleepost user signed up with this email, it will record this fb user and issue a verification email.
//If there's already a gleepost user, it will associate the two accounts if the invite is valid (proving that this fb user has access to that email; otherwise it will return status:registered.
func (api *API) fBFirstTimeWithEmail(email, fbToken, invite string, fbUser uint64) (token gp.Token, verification gp.Status, err error) {
	_, err = api.userWithEmail(email)
	if err != nil {
		//There isn't already a user with this email address.
		validates, e := api.validateEmail(email)
		if !validates {
			err = InvalidEmail
			return
		}
		if e != nil {
			err = e
			return
		}
		token, verification, err = api.FacebookRegister(fbToken, email, invite)
		return
	}
	//User has signed up already with a username+pass
	//If invite is valid, we can log in immediately
	token, verification, err = api.AttemptLoginWithInvite(email, invite, fbUser)
	return
}

//CreateFBUser records the existence of this (fbid:email) pair; when the user is verified it will be converted to a full gleepost user.
func (api *API) createFBUser(fbID uint64, email string, fbtoken string) (err error) {
	s, err := api.sc.Prepare("INSERT INTO facebook (fb_id, email, fb_token) VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(fbID, email, fbtoken)
	return
}

//FBUserWithEmail returns the facebook id we've seen associated with this email, or error if none exists.
func (api *API) FBUserWithEmail(email string) (fbid uint64, err error) {
	s, err := api.sc.Prepare("SELECT fb_id FROM facebook WHERE email = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(email).Scan(&fbid)
	if err == sql.ErrNoRows {
		err = NoSuchUser
	}
	return
}

//CreateFBVerification records a (hopefully random!) verification token for this facebook user.
func (api *API) createFBVerification(fbid uint64, token string) (err error) {
	s, err := api.sc.Prepare("REPLACE INTO facebook_verification (fb_id, token) VALUES (?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(fbid, token)
	return
}

//NoSuchVerificationToken is returned when the user provides a nonexistent verification token.
var NoSuchVerificationToken = gp.APIerror{Reason: "No such verification token"}

//FBVerificationExists returns the user this verification token is for, or an error if there is none.
func (api *API) fBVerificationExists(token string) (fbid uint64, err error) {
	s, err := api.sc.Prepare("SELECT fb_id FROM facebook_verification WHERE token = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(token).Scan(&fbid)
	if err == sql.ErrNoRows {
		err = NoSuchVerificationToken
	}
	return
}

func (api *API) fbUser(userID gp.UserID) (fbid uint64, err error) {
	s, err := api.sc.Prepare("SELECT fb_id FROM facebook WHERE user_id = ? AND fb_id != 0")
	if err != nil {
		return
	}
	err = s.QueryRow(userID).Scan(&fbid)
	if err == sql.ErrNoRows {
		err = NoSuchUser
	}
	return
}
