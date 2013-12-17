package lib

import (
	"fmt"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/huandu/facebook"
	"strconv"
	"time"
)

type FacebookToken struct {
	Expiry time.Time `facebook:"expires_at"`
	FBUser uint64    `facebook:"user_id"`
	Scopes []string  `facebook:"scopes"`
}

func DebugToken(token string) {
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
	token, err = api.CreateAndStoreToken(userId)
	return
}

func (api *API) FBGetGPUser(fbid uint64) (id gp.UserId, err error) {
	return api.db.UserIdFromFB(fbid)
}

func (api *API) FacebookRegister(fbToken string, email string) (err error) {
	t, err := api.FBValidateToken(fbToken)
	if err != nil {
		return
	}
	err = api.db.CreateFBUser(t.FBUser, email)
	if err == nil {
		err = api.FBissueVerification(t.FBUser)
	}
	return
}

func (api *API) FBissueVerification(fbid uint64) (err error) {
	email, err := api.FBGetEmail(fbid)
	if err != nil {
		return
	}
	random, err := randomString()
	if err != nil {
		return
	}
	err = api.db.CreateFBVerification(fbid, random)
	if err != nil {
		return
	}
	name, err := FBName(fbid)
	if err != nil {
		return
	}
	err = api.issueVerificationEmail(email, name, random)
	return
}

//TODO: get name from fb api
func FBName(fbid uint64) (name string, err error) {
	res, err := facebook.Get(fmt.Sprintf("/%d", fbid), nil)
	return res["name"].(string), err
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
