package lib

import (
	"fmt"
	"github.com/draaglom/GleepostAPI/db"
	"github.com/draaglom/GleepostAPI/gp"
	"github.com/huandu/facebook"
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

func FBValidateToken(fbToken string) (token FacebookToken, err error) {
	conf := gp.GetConfig()
	app := facebook.New(conf.Facebook.AppID, conf.Facebook.AppSecret)
	appToken := app.AppAccessToken()
	res, err := facebook.Get("/debug_token", facebook.Params{
		"access_token": appToken,
		"input_token": fbToken,
	})
	if err != nil {
		return
	}
	fmt.Printf("Result: %v\n", res)
	fmt.Println(res.Get("data.app_id"))
	fmt.Println(res.Get("app_id"))
	fmt.Println(res.Get("data.0.app_id"))
	fmt.Println(res.Get("data"))
	fmt.Println(res["data"])
	fmt.Println(res["data"]["app_id"])
	var id string
	id = res.Get("data.app_id").(string)
	if id != conf.Facebook.AppID {
		return token, gp.APIerror{"Bad facebook token"}
	}
	var unix int64
	unix = res.Get("data.expires_at").(int64)
	if time.Unix(unix, 0).After(time.Now()) {
		return token, gp.APIerror{"Bad facebook token"}
	}
	var valid bool
	valid = res.Get("data.is_valid").(bool)
	if !valid {
		return token, gp.APIerror{"Bad facebook token"}
	}
	err = res.Decode(token)
	fmt.Printf("%v", token)
	return
}

func FacebookLogin(fbToken string) (token gp.Token, err error) {
	t, err := FBValidateToken(fbToken)
	if err != nil {
		return
	}
	userId, err := FBGetGPUser(t.FBUser)
	if err != nil {
		token = createToken(userId)
		return
	}
	return
}

func FBGetGPUser(fbid uint64) (id gp.UserId, err error) {
	return db.UserIdFromFB(fbid)
}

func FacebookRegister(fbToken string, email string) (err error) {
	t, err := FBValidateToken(fbToken)
	if err != nil {
		return
	}
	err = db.CreateFBUser(t.FBUser, email)
	if err == nil {
		err = FBissueVerification(t.FBUser)
	}
	return
}

func FBissueVerification(fbid uint64) (err error) {
	email, err := FBGetEmail(fbid)
	if err != nil {
		return
	}
	random, err := randomString()
	if err != nil {
		return
	}
	err = db.CreateFBVerification(fbid, random)
	if err != nil {
		return
	}
	name, err := FBName(fbid)
	if err != nil {
		return
	}
	err = issueVerificationEmail(email, name, random)
	return
}

//TODO: get name from fb api
func FBName(fbid uint64) (name string, err error) {
	res, err := facebook.Get(fmt.Sprintf("/%d", fbid), nil)
	return res["name"].(string), err
}

func FBVerify(token string) (fbid uint64, err error) {
	return db.FBVerificationExists(token)
}

func FBGetEmail(fbid uint64) (email string, err error) {
	return db.FBUserEmail(fbid)
}

func UserSetFB(userId gp.UserId, fbid uint64) (err error) {
	return db.FBSetGPUser(fbid, userId)
}
