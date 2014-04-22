package lib

import (
	"fmt"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/huandu/facebook"
	"strconv"
	"time"
	"log"
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

type FB struct {
	config gp.FacebookConfig
}

var FBAPIError = gp.APIerror{"Something went wrong with a facebook API call."}

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
		return token, gp.APIerror{"Bad facebook token"}
	}
	expiry := time.Unix(int64(data["expires_at"].(float64)), 0)
	if !expiry.After(time.Now()) {
		fmt.Println("Token expired already")
		return token, gp.APIerror{"Bad facebook token"}
	}
	var valid bool
	valid = data["is_valid"].(bool)
	if !valid {
		fmt.Println("Token isn't valid")
		return token, gp.APIerror{"Bad facebook token"}
	}
	token.Expiry = expiry
	token.FBUser = uint64(data["user_id"].(float64))
	scopes := data["scopes"].([]interface{})
	for _, scope := range scopes {
		token.Scopes = append(token.Scopes, scope.(string))
	}
	return
}

func (api *API) FacebookLogin(fbToken string) (token gp.Token, err error) {
	t, err := api.FBValidateToken(fbToken)
	if err != nil {
		return
	}
	userId, err := api.FBGetGPUser(t.FBUser)
	if err != nil {
		return
	}
	err = api.UpdateFBData(fbToken)
	if err != nil {
		log.Println("Error pulling in profile changes from facebook:", err)
	}
	token, err = api.CreateAndStoreToken(userId)
	return
}

//UpdateFBData is a placeholder for the time being. In the future, place anything which needs to be regularly checked from facebook here.
func (api *API) UpdateFBData(fbToken string) (err error) {
	return nil
}

func (api *API) FBGetGPUser(fbid uint64) (id gp.UserId, err error) {
	return api.db.UserIdFromFB(fbid)
}

func (api *API) FacebookRegister(fbToken string, email string, invite string) (id gp.UserId, err error) {
	t, err := api.FBValidateToken(fbToken)
	if err != nil {
		return
	}
	err = api.db.CreateFBUser(t.FBUser, email)
	exists, _ := api.InviteExists(email, invite)
	if exists {
		id, err = api.UserWithEmail(email)
		if err != nil {
			log.Println("There isn't a user with this facebook email")
			id, err = api.CreateUserFromFB(t.FBUser, email)
		} else {
			err = api.UserSetFB(id, t.FBUser)
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
		}
	} else {
		if err == nil {
			err = api.FBissueVerification(t.FBUser)
		}
	}
	return
}

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

func FBAvatar(username string) (avatar string) {
	return fmt.Sprintf("https://graph.facebook.com/%s/picture?type=large", username)
}

func (api *API) FBVerify(token string) (fbid uint64, err error) {
	return api.db.FBVerificationExists(token)
}

func (api *API) FBGetEmail(fbid uint64) (email string, err error) {
	return api.db.FBUserEmail(fbid)
}

func (api *API) UserSetFB(userId gp.UserId, fbid uint64) (err error) {
	return api.db.FBSetGPUser(fbid, userId)
}

func (api *API) FBUserWithEmail(email string) (fbid uint64, err error) {
	return api.db.FBUserWithEmail(email)
}

func (api *API) UserAddFBUsersToGroup(user gp.UserId, fbusers []uint64, netId gp.NetworkId) (count int, err error) {
	for _, u := range fbusers {
		err = api.db.UserAddFBUserToGroup(user, u, netId)
		if err == nil {
			count++
		} else {
			return
		}
	}
	return
}

func (api *API) CreateUserFromFB(fbid uint64, email string) (id gp.UserId, err error) {
	//TODO: Deduplicate with lib/Verify
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
	id, err = api.createUser(username, random, email)
	if err != nil {
		log.Println("Something went wrong while creating the user from facebook:", err)
		return
	}
	_, err = api.assignNetworks(id, email)
	if err != nil {
		return
	}
	err = api.SetUserName(id, firstName, lastName)
	if err != nil {
		log.Println("Problem setting name:", err)
		return
	}
	err = api.SetProfileImage(id, FBAvatar(username))
	if err != nil {
		log.Println("Problem setting avatar:", err)
	}
	err = api.db.Verify(id)
	if err != nil {
		log.Println("Verifying failed in the db:", err)
		return
	}
	err = api.UserSetFB(id, fbid)
	if err != nil {
		log.Println("associating facebook account with user account failed:", err)
		return
	}
	err = api.AssignNetworksFromInvites(id, email)
	if err != nil {
		log.Println("Something went wrong while setting networks from invites:", err)
		return
	}
	err = api.AcceptAllInvites(email)
	if err != nil {
		log.Println("Something went wrong while accepting invites:", err)
		return
	}
	err = api.AssignNetworksFromFBInvites(id, fbid)
	if err !=  nil {
		log.Println("Something went wrong while setting networks from fb invites:", err)
		return
	}
	err = api.AcceptAllFBInvites(fbid)
	return

}
