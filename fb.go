package main

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

func ValidateToken(fbToken string) (token FacebookToken, err error) {
	conf := gp.GetConfig()
	res, err := facebook.Get("/debug_token", facebook.Params{
		"access_token": token,
	})
	if err != nil {
		return
	}
	var id string
	id = res["app_id"].(string)
	if id != conf.Facebook.AppID {
		return token, gp.APIerror{"Bad facebook token"}
	}
	var unix int64
	unix = res["expires_at"].(int64)
	if time.Unix(unix, 0).After(time.Now()) {
		return token, gp.APIerror{"Bad facebook token"}
	}
	var valid bool
	valid = res["is_valid"].(bool)
	if !valid {
		return token, gp.APIerror{"Bad facebook token"}
	}
	err = res.Decode(token)
	fmt.Printf("%v", token)
	return
}

func FacebookLogin(fbToken string) (token gp.Token, err error) {
	t, err := ValidateToken(fbToken)
	if err != nil {
		return
	}
	userId, err := FBGetGPUser(t.FBUser)
	if err != nil {
		token = createToken(userId)
		return
	}
	//TODO: Implement!
	return
}

func FBGetGPUser(fbid uint64) (id gp.UserId, err error) {
	return db.UserIdFromFB(fbid)
}
