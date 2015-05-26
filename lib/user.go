package lib

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/psc"
	"github.com/go-sql-driver/mysql"
)

type Users struct {
	sc      *psc.StatementCache
	statter PrefixStatter
}

//returns ENOSUCHUSER if this user doesn't exist
func (u Users) byID(id gp.UserID) (user gp.User, err error) {
	defer u.statter.Time(time.Now(), "users.byID.db")
	var av sql.NullString
	s, err := u.sc.Prepare("SELECT id, avatar, firstname, official FROM users WHERE id=?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&user.ID, &av, &user.Name, &user.Official)
	if err != nil {
		if err == sql.ErrNoRows {
			err = &gp.ENOSUCHUSER
		}
		return
	}
	if av.Valid {
		user.Avatar = av.String
	}
	return
}

//UserGetProfile returns the Profile (extended info) for the user with this ID.
func (api *API) UserGetProfile(userID, otherID gp.UserID) (user gp.Profile, err error) {
	if userID == otherID {
		return api.getProfile(userID, otherID)
	}
	shared, e := api.sameUniversity(userID, otherID)
	switch {
	case e != nil:
		fallthrough
	case !shared:
		err = &ENOTALLOWED
	default:
		return api.getProfile(userID, otherID)
	}
	return
}

//getProfile returns the Profile (extended info) for the user with this ID.
func (api *API) getProfile(perspective, otherID gp.UserID) (user gp.Profile, err error) {
	user, err = api._getProfile(otherID)
	if err != nil {
		return
	}
	user.Network, err = api.getUserUniversity(user.ID)
	if err != nil {
		return
	}
	rsvps, err := api.subjectiveRSVPCount(perspective, otherID)
	if err != nil {
		return
	}
	user.RSVPCount = rsvps
	groupCount, err := api.subjectiveMembershipCount(perspective, otherID)
	if err != nil {
		return
	}
	user.GroupCount = groupCount
	postCount, err := api.userPostCount(perspective, otherID)
	if err != nil {
		return
	}
	user.PostCount = postCount
	go api.esIndexUser(otherID)
	return
}

//IsAdmin returns true if this user member has their Admin flag set.
func (api *API) isAdmin(user gp.UserID) (admin bool) {
	s, err := api.sc.Prepare("SELECT is_admin FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(user).Scan(&admin)
	if err == nil && admin {
		return true
	}
	return false
}

//UserCreateUserSpecial manually creates a user with these details, bypassing validation etc
func (api *API) UserCreateUserSpecial(creator gp.UserID, first, last, email, pass string, verified bool, primaryNetwork gp.NetworkID) (userID gp.UserID, err error) {
	if !api.isAdmin(creator) {
		err = ENOTALLOWED
		return
	}
	return api.createUserSpecial(first, last, email, pass, verified, primaryNetwork)
}

func (api *API) createUserSpecial(first, last, email, pass string, verified bool, primaryNetwork gp.NetworkID) (userID gp.UserID, err error) {
	userID, err = api.createUser(first, last, pass, email)
	if err != nil {
		return
	}
	if verified {
		err = api.verify(userID)
		if err != nil {
			return
		}
	}
	err = api.setNetwork(userID, primaryNetwork)
	return
}

func (api *API) inviteURL(token, email string) string {
	if api.Config.DevelopmentMode {
		return fmt.Sprintf("https://dev.gleepost.com/?invite=%s&email=%s", token, email)
	}
	return fmt.Sprintf("https://gleepost.com/?invite=%s&email=%s", token, email)
}

func (api *API) issueInviteEmail(email string, userID gp.UserID, netID gp.NetworkID, token string) (err error) {
	var from gp.User
	from, err = api.users.byID(userID)
	if err != nil {
		return
	}
	var group gp.Group
	group, err = api.getNetwork(netID)
	if err != nil {
		return
	}
	url := api.inviteURL(token, email)
	subject := fmt.Sprintf("%s has invited you to the private group \"%s\" on Gleepost.", from.Name, group.Name)
	html := "<html><body>" +
		"Don't miss out on their events - <a href=" + url + ">Click here to accept the invitation.</a><br>" +
		"On your phone? <a href=\"" + inviteCampaignIOS + "\">install the app on your iPhone here</a>" +
		" or <a href=\"" + inviteCampaignAndroid + "\">click here to get the Android app.</a>" +
		"</body></html>"
	err = api.Mail.SendHTML(email, subject, html)
	return
}

//GetEmail returns this user's email address.
func (api *API) getEmail(id gp.UserID) (email string, err error) {
	s, err := api.sc.Prepare("SELECT email FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&email)
	return
}

//UserSetName updates this user's name.
func (api *API) UserSetName(id gp.UserID, firstName, lastName string) (err error) {
	s, err := api.sc.Prepare("UPDATE users SET firstname = ?, lastname = ? where id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(firstName, lastName, id)
	if err != nil {
		return
	}
	api.esIndexUser(id)
	return
}

//UserSetProfileImage updates this user's profile image to the new url
func (api *API) UserSetProfileImage(id gp.UserID, url string) (err error) {
	exists, err := api.userUploadExists(id, url)
	if err != nil {
		return
	}
	if !exists {
		return NoSuchUpload
	}
	err = api.setProfileImage(id, url)
	if err != nil {
		return
	}
	api.esIndexUser(id)
	return
}

func (api *API) setProfileImage(id gp.UserID, url string) (err error) {
	s, err := api.sc.Prepare("UPDATE users SET avatar = ? WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(url, id)
	return
}

func (api *API) userPing(id gp.UserID) {
	api.cache.UserPing(id, api.Config.OnlineTimeout)
}

func (api *API) userIsOnline(id gp.UserID) bool {
	return api.cache.UserIsOnline(id)
}

//UserHasPosted returns true if user has ever created a post from the perspective of perspective.
//TODO: Implement a direct version
func (api *API) userHasPosted(user gp.UserID, perspective gp.UserID) (posted bool, err error) {
	posts, err := api.GetUserPosts(user, perspective, OSTART, 0, 1, "")
	if err != nil {
		return
	}
	if len(posts) > 0 {
		return true, nil
	}
	return false, nil
}

/********************************************************************
		User
********************************************************************/

//RegisterUser creates a user with a name a password hash and an email address.
//They'll be created in an unverified state.
func (api *API) _registerUser(first, last string, hash []byte, email string) (gp.UserID, error) {
	s, err := api.sc.Prepare("INSERT INTO users(firstname, lastname, password, email) VALUES (?,?,?,?)")
	if err != nil {
		return 0, err
	}
	res, err := s.Exec(first, last, hash, email)
	if err != nil {
		if err, ok := err.(*mysql.MySQLError); ok {
			if err.Number == 1062 {
				return 0, UserAlreadyExists
			}
		}
		return 0, err
	}
	id, _ := res.LastInsertId()
	return gp.UserID(id), nil
}

//UserChangeTagline sets this user's tagline (obviously enough)
func (api *API) UserChangeTagline(userID gp.UserID, tagline string) (err error) {
	s, err := api.sc.Prepare("UPDATE users SET `desc` = ? WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(tagline, userID)
	return
}

//GetProfile fetches a user but DOES NOT GET THEIR NETWORK.
func (api *API) _getProfile(id gp.UserID) (user gp.Profile, err error) {
	var av, desc, lastName sql.NullString
	s, err := api.sc.Prepare("SELECT `desc`, avatar, firstname, lastname, official FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&desc, &av, &user.Name, &lastName, &user.Official)
	log.Printf("db.GetProfile(%d)\n", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return user, &gp.ENOSUCHUSER
		}
		return
	}
	if av.Valid {
		user.Avatar = av.String
	}
	if desc.Valid {
		user.Desc = desc.String
	}
	if lastName.Valid {
		user.FullName = user.Name + " " + lastName.String
	}
	user.ID = id
	return
}

//SetBusyStatus records whether this user is busy or not.
func (api *API) SetBusyStatus(id gp.UserID, busy bool) (err error) {
	s, err := api.sc.Prepare("UPDATE users SET busy = ? WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(busy, id)
	return
}

//BusyStatus returns this user's busy status.
func (api *API) BusyStatus(id gp.UserID) (busy bool, err error) {
	s, err := api.sc.Prepare("SELECT busy FROM users WHERE id = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(id).Scan(&busy)
	return
}

//UserIDFromFB gets the gleepost user who has fbid associated, or an error if there is none.
func (api *API) userIDFromFB(fbid uint64) (id gp.UserID, err error) {
	s, err := api.sc.Prepare("SELECT user_id FROM facebook WHERE fb_id = ? AND user_id IS NOT NULL")
	if err != nil {
		return
	}
	err = s.QueryRow(fbid).Scan(&id)
	if err == sql.ErrNoRows {
		err = NoSuchUser
	}
	return
}

//UserWithEmail returns the user whose email this is, or an error if they don't exist.
func (api *API) userWithEmail(email string) (id gp.UserID, err error) {
	s, err := api.sc.Prepare("SELECT id FROM users WHERE email = ?")
	if err != nil {
		return
	}
	err = s.QueryRow(email).Scan(&id)
	if err == sql.ErrNoRows {
		err = NoSuchUser
	}
	return
}

//GetGlobalAdmins returns all users who are gleepost company admins.
func (api *API) getGlobalAdmins() (users []gp.User, err error) {
	users = make([]gp.User, 0)
	s, err := api.sc.Prepare("SELECT id, firstname, avatar, official FROM users WHERE is_admin = 1")
	if err != nil {
		return
	}
	rows, err := s.Query()
	if err != nil {
		return
	}
	for rows.Next() {
		var u gp.User
		var av sql.NullString
		err = rows.Scan(&u.ID, &u.Name, &av, &u.Official)
		if err != nil {
			log.Println("GetGlobalAdmins: Problem scanning:", err)
			continue
		}
		users = append(users, u)
	}
	return users, nil
}
