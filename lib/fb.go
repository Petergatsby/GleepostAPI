package lib

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/huandu/facebook"
)

//FacebookToken contains the parsed expiry, user and permission scopes of a facebook authentication token.
type FacebookToken struct {
	Expiry time.Time `facebook:"expires_at"`
	FBUser uint64    `facebook:"user_id"`
	Scopes []string  `facebook:"scopes"`
}

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

//FB contains the configuration specific to this facebook app.
type FB struct {
	config gp.FacebookConfig
}

//FBAPIError is a catchall error for anything that went wrong with a facebook reqest.
var FBAPIError = gp.APIerror{Reason: "Something went wrong with a facebook API call."}

//FBValidateToken takes a client-supplied facebook access token and returns a FacebookToken, or an error if the token is invalid in some way
//ie, expired or for another app.
func (api *API) FBValidateToken(fbToken string) (token FacebookToken, err error) {
	app := facebook.New(api.fb.config.AppID, api.fb.config.AppSecret)
	appToken := app.AppAccessToken()
	res, err := facebook.Get("/debug_token", facebook.Params{
		"access_token": appToken,
		"input_token":  fbToken,
	})
	if err != nil {
		return
	}
	data := res["data"].(map[string]interface{})
	fmt.Printf("%v\n", data)
	tokenappid := uint64(data["app_id"].(float64))
	appid, err := strconv.ParseUint(api.fb.config.AppID, 10, 64)
	if err != nil {
		return
	}
	if appid != tokenappid {
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
	token.FBUser = uint64(data["user_id"].(float64))
	scopes := data["scopes"].([]interface{})
	for _, scope := range scopes {
		token.Scopes = append(token.Scopes, scope.(string))
	}
	return
}

//FacebookLogin takes a facebook access token supplied by a user and tries to issue a gleepost session token,
// or an error if there isn't an associated gleepost user for this facebook account.
func (api *API) FacebookLogin(fbToken string) (token gp.Token, err error) {
	t, err := api.FBValidateToken(fbToken)
	if err != nil {
		return
	}
	userID, err := api.FBGetGPUser(t.FBUser)
	if err != nil {
		return
	}
	err = api.UpdateFBData(fbToken)
	if err != nil {
		log.Println("Error pulling in profile changes from facebook:", err)
	}
	token, err = api.CreateAndStoreToken(userID)
	return
}

//UpdateFBData is a placeholder for the time being. In the future, place anything which needs to be regularly checked from facebook here.
func (api *API) UpdateFBData(fbToken string) (err error) {
	return nil
}

//FBGetGPUser returns the associated gleepost user for a given facebook id, or sql.ErrNoRows if that user doesn't exist.
//TODO: Change to ENOSUCHUSER
func (api *API) FBGetGPUser(fbid uint64) (id gp.UserId, err error) {
	return api.db.UserIdFromFB(fbid)
}

//FacebookRegister takes a facebook access token, an email and an (optional) invite key.
//If the email/invite pair is valid, it will associate this facebook account with the owner of this
// email address, or create a gleepost account as appropriate.
//If the invite is invalid or nonexistent, it issues a verification email
//(the rest of the association will be handled upon verification in FBVerify.
func (api *API) FacebookRegister(fbToken string, email string, invite string) (id gp.UserId, err error) {
	t, err := api.FBValidateToken(fbToken)
	if err != nil {
		return
	}
	err = api.db.CreateFBUser(t.FBUser, email)
	exists, _ := api.InviteExists(email, invite)
	if exists {
		id, err = api.FBSetVerified(email, t.FBUser)
		return
	}
	if err == nil {
		err = api.FBissueVerification(t.FBUser)
	}
	return
}

//FBSetVerified creates a gleepost user for this fbuser, or associates with an existing one as appropriate.
func (api *API) FBSetVerified(email string, fbuser uint64) (id gp.UserId, err error) {
	id, err = api.UserWithEmail(email)
	if err != nil {
		log.Println("There isn't a user with this facebook email")
		id, err = api.CreateUserFromFB(fbuser, email)
		return
	}
	err = api.UserSetFB(id, fbuser)
	if err == nil {
		err = api.db.Verify(id)
		if err == nil {
			log.Println("Verifying worked. Now setting networks from invites...")
			err = api.AssignNetworksFromInvites(id, email)
			if err != nil {
				log.Println("Something went wrong while setting networks from invites:", err)
				return
			}
			err = api.AcceptAllInvites(email)
		}
	}
	return
}

//FBissueVerification creates and sends a verification email for this facebook user, or returns an error if we haven't seen them before (ie, we don't have their email address on file)
//TODO: Think about decoupling this from the email check
func (api *API) FBissueVerification(fbid uint64) (err error) {
	email, err := api.FBGetEmail(fbid)
	if err != nil {
		return
	}
	random, err := RandomString()
	if err != nil {
		return
	}
	err = api.db.CreateFBVerification(fbid, random)
	if err != nil {
		return
	}
	firstName, _, _, err := FBName(fbid)
	if err != nil {
		return
	}
	err = api.issueVerificationEmail(email, firstName, random)
	return
}

//FBName retrieves the first-, last-, and username of facebook id fbid.
func FBName(fbid uint64) (firstName, lastName, username string, err error) {
	res, err := facebook.Get(fmt.Sprintf("/%d", fbid), nil)
	log.Println(res)
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
func FBAvatar(username string) (avatar string) {
	return fmt.Sprintf("https://graph.facebook.com/%s/picture?type=large", username)
}

//FBVerify takes a verification token (previously sent to the user's email) and returns the facebook id of the (to-be-verified) facebook user or an error if the invite isn't valid.
func (api *API) FBVerify(token string) (fbid uint64, err error) {
	return api.db.FBVerificationExists(token)
}

//FBGetEmail returns the email address we have on file for this facebook id, or an error if we don't have one.
func (api *API) FBGetEmail(fbid uint64) (email string, err error) {
	return api.db.FBUserEmail(fbid)
}

//UserSetFB sets the associated facebook account for the gleepost user userID.
func (api *API) UserSetFB(userID gp.UserId, fbid uint64) (err error) {
	return api.db.FBSetGPUser(fbid, userID)
}

//FBUserWithEmail returns the facebook ID for the user who owns email, or an error if we don't know about that email.
func (api *API) FBUserWithEmail(email string) (fbid uint64, err error) {
	return api.db.FBUserWithEmail(email)
}

//UserAddFBUsersToGroup takes a list of facebook users and records that they've been invited to the group netID by userID
func (api *API) UserAddFBUsersToGroup(userID gp.UserId, fbusers []uint64, netID gp.NetworkId) (count int, err error) {
	for _, u := range fbusers {
		err = api.db.UserAddFBUserToGroup(userID, u, netID)
		if err == nil {
			count++
		} else {
			return
		}
	}
	return
}

//CreateUserFromFB takes a facebook id and an email address and creates a gleepost user, returning their newly created id.
func (api *API) CreateUserFromFB(fbid uint64, email string) (userID gp.UserId, err error) {
	firstName, lastName, username, err := FBName(fbid)
	if err != nil {
		log.Println("Couldn't get name info from facebook:", err)
		return
	}
	random, err := RandomString()
	if err != nil {
		return
	}
	//TODO: Do something different with names, two john smiths are
	userID, err = api.createUser(username, random, email)
	if err != nil {
		log.Println("Something went wrong while creating the user from facebook:", err)
		return
	}
	_, err = api.assignNetworks(userID, email)
	if err != nil {
		return
	}
	err = api.SetUserName(userID, firstName, lastName)
	if err != nil {
		log.Println("Problem setting name:", err)
		return
	}
	err = api.SetProfileImage(userID, FBAvatar(username))
	if err != nil {
		log.Println("Problem setting avatar:", err)
	}
	err = api.db.Verify(userID)
	if err != nil {
		log.Println("Verifying failed in the db:", err)
		return
	}
	err = api.UserSetFB(userID, fbid)
	if err != nil {
		log.Println("associating facebook account with user account failed:", err)
		return
	}
	err = api.AssignNetworksFromInvites(userID, email)
	if err != nil {
		log.Println("Something went wrong while setting networks from invites:", err)
		return
	}
	err = api.AcceptAllInvites(email)
	if err != nil {
		log.Println("Something went wrong while accepting invites:", err)
		return
	}
	err = api.AssignNetworksFromFBInvites(userID, fbid)
	if err != nil {
		log.Println("Something went wrong while setting networks from fb invites:", err)
		return
	}
	err = api.AcceptAllFBInvites(fbid)
	return

}
