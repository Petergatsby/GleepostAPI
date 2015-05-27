package lib

import (
	"database/sql"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/apns"
)

const (
	//For parsing
	mysqlTime = "2006-01-02 15:04:05"
)

//NoSuchLevelErr happens when you try to set an approval level outside the range [0..3].
var NoSuchLevelErr = gp.APIerror{Reason: "That's not a valid approval level"}

//NoSuchGroup is returned when a lookup of a group's master-group fails.
var NoSuchGroup = gp.APIerror{Reason: "No such group"}

//NotChanged indicates that the approve level was set equal to its existing value.
var NotChanged = gp.APIerror{Reason: "Level not changed"}

//ApproveAccess returns this user's access to review / change review level in this network.
func (api *API) ApproveAccess(userID gp.UserID) (access gp.ApprovePermission, err error) {
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	return api.approveAccess(userID, primary.ID)
}

func (api *API) approveAccess(userID gp.UserID, netID gp.NetworkID) (perm gp.ApprovePermission, err error) {
	q := "SELECT role_level FROM user_network JOIN network ON network.master_group = user_network.network_id WHERE network.id = ? AND user_network.user_id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	var level int
	err = s.QueryRow(netID, userID).Scan(&level)
	switch {
	case err != nil && err == sql.ErrNoRows:
		return perm, nil
	case err != nil:
		return perm, err
	default:
		if level > 0 {
			perm.ApproveAccess = true
		}
		if level > 1 {
			perm.LevelChange = true
		}
		return perm, nil
	}

}

//ApproveLevel returns this network's current approval level, or ENOTALLOWED if you aren't allowed to see it.
func (api *API) ApproveLevel(userID gp.UserID) (level gp.ApproveLevel, err error) {
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	return api.approveLevel(primary.ID)
}

//SetApproveLevel sets this network's approval level, or returns ENOTALLOWED if you can't.
func (api *API) SetApproveLevel(userID gp.UserID, level int) (err error) {
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	access, err := api.approveAccess(userID, primary.ID)
	switch {
	case err != nil:
		return err
	case access.LevelChange == false:
		return &ENOTALLOWED
	case level < 0 || level > 3:
		return NoSuchLevelErr
	default:
		err = api.setApproveLevel(primary.ID, level)
		if err == nil {
			go api.approvalChangePush(primary.ID, userID, level)
		}
		if err == NotChanged {
			err = nil
		}
		return
	}
}

func (api *API) approvalChangePush(netID gp.NetworkID, changer gp.UserID, level int) (err error) {
	badge := api.approvalBadgeCount(changer, netID)
	users, err := api.approveUsers(netID)
	if err != nil {
		log.Println(err)
		return
	}
	for _, u := range users {
		devices, err := getDevices(api.sc, u.ID, "approve")
		if err != nil {
			log.Println(err)
			continue
		}
		for _, d := range devices {
			switch {
			case d.Type == "ios":
				payload := apns.NewPayload()
				alert := apns.NewAlertDictionary()
				alert.ActionLocKey = "OK"
				alert.LocKey = "level_change"
				alert.LocArgs = []string{strconv.Itoa(level)}
				payload.Badge = badge
				payload.Alert = alert
				payload.Sound = "default"
				pn := apns.NewPushNotification()
				pn.DeviceToken = d.ID
				pn.AddPayload(payload)
				err := api.pushers["approve"].IOSPush(pn)
				if err != nil {
					log.Println(err)
				}
			default:
				//We only support iOS so far.
			}
		}
	}
	return nil
}

//UserGetPending returns all the posts pending review in this user's primary network.
func (api *API) UserGetPending(userID gp.UserID) (pending []gp.PendingPost, err error) {
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	return api.GetNetworkPending(userID, primary.ID)

}

//UserGetApproved returns posts that have been approved in this user's primary network.
func (api *API) UserGetApproved(userID gp.UserID, mode int, index int64, count int) (approved []gp.PendingPost, err error) {
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	return api.GetNetworkApproved(userID, primary.ID, mode, index, count)

}

//UserGetRejected returns posts that have been rejected in this user's primary network.
func (api *API) UserGetRejected(userID gp.UserID, mode int, index int64, count int) (rejected []gp.PendingPost, err error) {
	primary, err := api.getUserUniversity(userID)
	if err != nil {
		return
	}
	return api.GetNetworkRejected(userID, primary.ID, mode, index, count)

}

//GetNetworkPending returns all the posts which are pending review in this network.
func (api *API) GetNetworkPending(userID gp.UserID, netID gp.NetworkID) (pending []gp.PendingPost, err error) {
	pending = make([]gp.PendingPost, 0)
	access, err := api.approveAccess(userID, netID)
	switch {
	case err != nil:
		return
	case !access.ApproveAccess:
		return pending, &ENOTALLOWED
	default:
		return api.getNetworkPending(userID, netID)
	}
}

func (api *API) getNetworkPending(userID gp.UserID, netID gp.NetworkID) (pending []gp.PendingPost, err error) {
	pending = make([]gp.PendingPost, 0)
	_pending, err := api.pendingPosts(netID)
	if err != nil {
		return pending, err
	}

	for i := range _pending {
		pending = append(pending, gp.PendingPost{PostSmall: _pending[i]})
		processed, err := api.postProcess(pending[i].PostSmall, userID)
		if err == nil {
			pending[i].PostSmall = processed
		}
		history, err := api.reviewHistory(pending[i].ID)
		if err == nil {
			pending[i].ReviewHistory = history
		}
	}
	return pending, nil
}

func (api *API) isPendingVisible(userID gp.UserID, postID gp.PostID) (visible bool, err error) {
	p, err := api.getPost(postID)
	if err != nil {
		//Not sure what kinds of errors GetPost will give me, so we'll just say you can't see the post.
		return false, nil
	}
	in, err := api.userInNetwork(userID, p.Network)
	switch {
	case err != nil:
		return
	case !in:
		return false, &ENOTALLOWED
	default:
		//Is the post still pending?
		pending, _ := api.pendingStatus(postID)
		if pending > 0 {
			return true, nil
		}
		return false, nil
	}
}

//ApprovePost will mark this post approved if you are allowed to do so, or return ENOTALLOWED otherwise.
func (api *API) ApprovePost(userID gp.UserID, postID gp.PostID, reason string) (err error) {
	visible, err := api.isPendingVisible(userID, postID)
	if !visible || err != nil {
		return &ENOTALLOWED
	}
	p, _ := api.getPost(postID)
	access, _ := api.approveAccess(userID, p.Network)
	if !access.ApproveAccess {
		return &ENOTALLOWED
	}
	err = api.approvePost(userID, postID, reason)
	if err == nil {
		//Notify user their post has been approved
		api.notifObserver.Notify(approvedEvent{userID: userID, recipientID: p.By.ID, postID: postID})
		//Silently reduce badge count for app users
		//nb: just using p.Network won't work if we eventually want to eg. approve posts in public groups
		api.silentSetApproveBadgeCount(p.Network, userID)
	}
	return
}

//GetNetworkApproved returns the list of approved posts in this network.
func (api *API) GetNetworkApproved(userID gp.UserID, netID gp.NetworkID, mode int, index int64, count int) (approved []gp.PendingPost, err error) {
	approved = make([]gp.PendingPost, 0)
	access, err := api.approveAccess(userID, netID)
	switch {
	case err != nil:
		return
	case !access.ApproveAccess:
		return approved, &ENOTALLOWED
	default:
		_approved, err := api.getNetworkApproved(netID, mode, index, count)
		if err != nil {
			return approved, err
		}
		for i := range _approved {
			approved = append(approved, gp.PendingPost{PostSmall: _approved[i]})
			processed, err := api.postProcess(approved[i].PostSmall, userID)
			if err == nil {
				approved[i].PostSmall = processed
			}
			history, err := api.reviewHistory(approved[i].ID)
			if err == nil {
				approved[i].ReviewHistory = history
			}
		}
		return approved, nil
	}
}

//RejectPost marks this post as rejected (if you're allowed) or ENOTALLOWED otherwise.
func (api *API) RejectPost(userID gp.UserID, postID gp.PostID, reason string) (err error) {
	visible, err := api.isPendingVisible(userID, postID)
	if !visible || err != nil {
		return &ENOTALLOWED
	}
	p, _ := api.getPost(postID)
	access, _ := api.approveAccess(userID, p.Network)
	if !access.ApproveAccess {
		return &ENOTALLOWED
	}
	err = api.rejectPost(userID, postID, reason)
	if err == nil {
		api.notifObserver.Notify(rejectedEvent{userID: userID, recipientID: p.By.ID, postID: postID})
		api.silentSetApproveBadgeCount(p.Network, userID)
	}
	return
}

//GetNetworkRejected returns the list of rejected posts in this network.
func (api *API) GetNetworkRejected(userID gp.UserID, netID gp.NetworkID, mode int, index int64, count int) (rejected []gp.PendingPost, err error) {
	rejected = make([]gp.PendingPost, 0)
	access, err := api.approveAccess(userID, netID)
	switch {
	case err != nil:
		return
	case !access.ApproveAccess:
		return rejected, &ENOTALLOWED
	default:
		_rejected, err := api.getNetworkRejected(netID, mode, index, count)
		if err != nil {
			return rejected, err
		}
		for i := range _rejected {
			rejected = append(rejected, gp.PendingPost{PostSmall: _rejected[i]})
			processed, err := api.postProcess(rejected[i].PostSmall, userID)
			if err == nil {
				rejected[i].PostSmall = processed
			}
			history, err := api.reviewHistory(rejected[i].ID)
			if err == nil {
				rejected[i].ReviewHistory = history
			}
		}
		return rejected, nil
	}
}

//PendingPosts returns this user's pending posts.
func (api *API) PendingPosts(userID gp.UserID) (pending []gp.PendingPost, err error) {
	pending = make([]gp.PendingPost, 0)
	_pending, err := api.userPendingPosts(userID)
	if err != nil {
		return pending, err
	}
	for i := range _pending {
		pending = append(pending, gp.PendingPost{PostSmall: _pending[i]})
		processed, err := api.postProcess(pending[i].PostSmall, userID)
		if err == nil {
			pending[i].PostSmall = processed
		}
		history, err := api.reviewHistory(pending[i].ID)
		if err == nil {
			pending[i].ReviewHistory = history
		}
	}
	return
}

func (api *API) silentSetApproveBadgeCount(netID gp.NetworkID, userID gp.UserID) {
	posts, err := api.getNetworkPending(userID, netID)
	if err != nil {
		log.Println(err)
		return
	}
	badge := len(posts)
	users, err := api.approveUsers(netID)
	if err != nil {
		log.Println(err)
		return
	}
	for _, u := range users {
		devices, err := getDevices(api.sc, u.ID, "approve")
		if err != nil {
			log.Println(err)
			continue
		}
		for _, d := range devices {
			switch {
			case d.Type == "ios":
				payload := apns.NewPayload()
				payload.Badge = badge
				pn := apns.NewPushNotification()
				pn.DeviceToken = d.ID
				pn.AddPayload(payload)
				err := api.pushers["approve"].IOSPush(pn)
				if err != nil {
					log.Println(err)
				}
			default:
				//We only support iOS so far.
			}
		}
	}
}

//approveUsers returns all the users who have Approve access in this network.
func (api *API) approveUsers(netID gp.NetworkID) (users []gp.UserRole, err error) {
	master, err := api.masterGroup(netID)
	if err != nil {
		return
	}
	return getNetworkUsers(api.sc, master)
}

func (api *API) approvalBadgeCount(userID gp.UserID, netID gp.NetworkID) (badge int) {
	posts, err := api.getNetworkPending(userID, netID)
	if err != nil {
		log.Println(err)
		return
	}
	badge = len(posts)
	return
}

func (api *API) postsToApproveNotification(userID gp.UserID, netID gp.NetworkID) {
	badge := api.approvalBadgeCount(userID, netID)
	if badge == 0 {
		return
	}
	users, err := api.approveUsers(netID)
	if err != nil {
		log.Println(err)
		return
	}
	for _, u := range users {
		devices, err := getDevices(api.sc, u.ID, "approve")
		if err != nil {
			log.Println(err)
			continue
		}
		for _, d := range devices {
			switch {
			case d.Type == "ios":
				payload := apns.NewPayload()
				alert := apns.NewAlertDictionary()
				alert.ActionLocKey = "Review"
				alert.LocKey = "to_review"
				alert.LocArgs = []string{strconv.Itoa(badge)}
				payload.Badge = badge
				payload.Alert = alert
				payload.Sound = "default"
				pn := apns.NewPushNotification()
				pn.DeviceToken = d.ID
				pn.AddPayload(payload)
				err := api.pushers["approve"].IOSPush(pn)
				if err != nil {
					log.Println(err)
				}
			default:
				//We only support iOS so far.
			}
		}
	}
}

func (api *API) maybeResubmitPost(userID gp.UserID, postID gp.PostID, netID gp.NetworkID, reason string) (err error) {
	pending, err := api.pendingStatus(postID)
	if err != nil {
		return
	}
	//if !pending, do nothing
	if pending == 0 {
		return
	}
	return api.ResubmitPost(userID, postID, netID, reason)
}

//ResubmitPost puts the post back in the approval queue to be reviewed again.
func (api *API) ResubmitPost(userID gp.UserID, postID gp.PostID, netID gp.NetworkID, reason string) (err error) {
	err = api.resubmitPost(userID, postID, reason)
	if err == nil {
		api.postsToApproveNotification(userID, netID)
	}
	return
}

//ApproveLevel returns this network's current approval level.
func (api *API) approveLevel(netID gp.NetworkID) (level gp.ApproveLevel, err error) {
	q := "SELECT approval_level, approved_categories FROM network WHERE id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	var approvedCategories sql.NullString
	err = s.QueryRow(netID).Scan(&level.Level, &approvedCategories)
	if err != nil {
		return
	}
	cats := []string{}
	if approvedCategories.Valid {
		cats = strings.Split(approvedCategories.String, ",")
	}
	level.Categories = cats
	return level, nil
}

//SetApproveLevel updates this network's approval level.
func (api *API) setApproveLevel(netID gp.NetworkID, level int) (err error) {
	q := "UPDATE network SET approval_level = ?, approved_categories = ? WHERE id = ?"
	var categories string
	switch {
	case level == 0:
		categories = ""
	case level == 1:
		categories = "party"
	case level == 2:
		categories = "event"
	case level == 3:
		categories = "all"
	default:
		return gp.APIerror{Reason: "That's not a valid approve level"}
	}
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	res, err := s.Exec(level, categories, netID)
	if err != nil {
		return
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return
	}
	if affected == 0 {
		return NotChanged
	}
	return
}

//PendingPosts returns all the posts in this network which are awaiting review.
func (api *API) pendingPosts(netID gp.NetworkID) (pending []gp.PostSmall, err error) {
	pending = make([]gp.PostSmall, 0)
	//This query assumes pending = 1 and rejected = 2
	q := "SELECT wall_posts.id, wall_posts.`by`, time, text, network_id " +
		"FROM wall_posts " +
		"WHERE deleted = 0 AND pending = 1 AND network_id = ? " +
		"ORDER BY time DESC "
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(netID)
	if err != nil {
		return
	}
	defer rows.Close()
	return api.scanPostRows(rows, false)
}

//ReviewHistory returns all the review events on this post
func (api *API) reviewHistory(postID gp.PostID) (history []gp.ReviewEvent, err error) {
	history = make([]gp.ReviewEvent, 0)
	q := "SELECT action, `by`, reason, `timestamp` FROM post_reviews WHERE post_id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(postID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		event := gp.ReviewEvent{}
		var by gp.UserID
		var reason sql.NullString
		var t string
		err = rows.Scan(&event.Action, &by, &reason, &t)
		if err != nil {
			return
		}
		if reason.Valid {
			event.Reason = reason.String
		}
		user, UsrErr := api.users.byID(by)
		if UsrErr != nil {
			return history, UsrErr
		}
		event.By = user
		time, TimeErr := time.Parse(mysqlTime, t)
		if TimeErr != nil {
			return history, TimeErr
		}
		event.At = time
		history = append(history, event)
	}
	return
}

//PendingStatus returns the current approval status of this post. 0 = approved, 1 = pending, 2 = rejected.
func (api *API) pendingStatus(postID gp.PostID) (pending int, err error) {
	q := "SELECT pending FROM wall_posts WHERE id = ?"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	err = s.QueryRow(postID).Scan(&pending)
	return
}

//ApprovePost marks this post as approved by this user.
func (api *API) approvePost(userID gp.UserID, postID gp.PostID, reason string) (err error) {
	//Should be one transaction...
	q := "INSERT INTO post_reviews (post_id, action, `by`, reason) VALUES (?, 'approved', ?, ?)"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(postID, userID, reason)
	if err != nil {
		return
	}
	q2 := "UPDATE wall_posts SET pending = 0 WHERE id = ?"
	s, err = api.sc.Prepare(q2)
	if err != nil {
		return
	}
	_, err = s.Exec(postID)
	if err != nil {
		return
	}
	q3 := "UPDATE wall_posts SET time = NOW() WHERE id = ?"
	s, err = api.sc.Prepare(q3)
	if err != nil {
		return
	}
	_, err = s.Exec(postID)
	return
}

//GetNetworkApproved returns the 20 most recent approved posts in this network.
func (api *API) getNetworkApproved(netID gp.NetworkID, mode int, index int64, count int) (approved []gp.PostSmall, err error) {
	approved = make([]gp.PostSmall, 0)
	q := "SELECT wall_posts.id, wall_posts.`by`, time, text, network_id " +
		"FROM wall_posts JOIN post_reviews ON post_reviews.post_id = wall_posts.id " +
		"WHERE wall_posts.deleted = 0 AND pending = 0 AND post_reviews.action = 'approved' " +
		"AND network_id = ? "
	switch {
	case mode == ByOffsetDescending:
		q += "ORDER BY post_reviews.timestamp DESC LIMIT ?, ?"
	case mode == ChronologicallyAfterID:
		q += "AND wall_posts.time > (SELECT time FROM wall_posts WHERE post_id = ?) " +
			"ORDER BY post_reviews.timestamp DESC LIMIT 0, ?"
	case mode == ChronologicallyBeforeID:
		q += "AND wall_posts.time < (SELECT time FROM wall_posts WHERE post_id = ?) " +
			"ORDER BY post_reviews.timestamp DESC LIMIT 0, ?"
	}
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(netID, index, count)
	if err != nil {
		return
	}
	defer rows.Close()
	return api.scanPostRows(rows, false)
}

//RejectPost marks this post as 'rejected'.
func (api *API) rejectPost(userID gp.UserID, postID gp.PostID, reason string) (err error) {
	q := "INSERT INTO post_reviews (post_id, action, `by`, reason) VALUES (?, 'rejected', ?, ?)"
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(postID, userID, reason)
	if err != nil {
		return
	}
	q = "UPDATE wall_posts SET pending = 2 WHERE id = ?"
	s, err = api.sc.Prepare(q)
	if err != nil {
		return
	}
	_, err = s.Exec(postID)
	return
}

//ResubmitPost marks this post as 'pending' again.
func (api *API) resubmitPost(userID gp.UserID, postID gp.PostID, reason string) (err error) {
	s, err := api.sc.Prepare("INSERT INTO post_reviews (post_id, action, `by`, reason) VALUES (?, 'edited', ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(postID, userID, reason)
	if err != nil {
		return
	}
	s, err = api.sc.Prepare("UPDATE wall_posts SET pending = 1 WHERE id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(postID)
	return
}

//GetNetworkRejected returns the posts in this network which have been rejected.
func (api *API) getNetworkRejected(netID gp.NetworkID, mode int, index int64, count int) (rejected []gp.PostSmall, err error) {
	rejected = make([]gp.PostSmall, 0)
	q := "SELECT wall_posts.id, wall_posts.`by`, time, text, network_id " +
		"FROM wall_posts JOIN post_reviews ON post_reviews.post_id = wall_posts.id " +
		"WHERE wall_posts.deleted = 0 AND pending = 2 AND post_reviews.action = 'rejected' " +
		"AND network_id = ? "
	switch {
	case mode == ByOffsetDescending:
		q += "ORDER BY post_reviews.timestamp DESC LIMIT ?, ?"
	case mode == ChronologicallyAfterID:
		q += "AND wall_posts.time > (SELECT time FROM wall_posts WHERE post_id = ?) " +
			"ORDER BY post_reviews.timestamp DESC LIMIT 0, ?"
	case mode == ChronologicallyBeforeID:
		q += "AND wall_posts.time < (SELECT time FROM wall_posts WHERE post_id = ?) " +
			"ORDER BY post_reviews.timestamp DESC LIMIT 0, ?"
	}
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(netID, index, count)
	if err != nil {
		return
	}
	defer rows.Close()
	return api.scanPostRows(rows, false)
}

//UserPendingPosts returns all this user's pending posts.
func (api *API) userPendingPosts(userID gp.UserID) (pending []gp.PostSmall, err error) {
	pending = make([]gp.PostSmall, 0)
	//This query assumes pending = 1 and rejected = 2
	q := "SELECT DISTINCT wall_posts.id, wall_posts.`by`, time, text, network_id " +
		"FROM wall_posts " +
		"LEFT JOIN post_reviews ON wall_posts.id = post_reviews.post_id " +
		"WHERE deleted = 0 AND pending > 0 AND wall_posts.`by` = ? " +
		"GROUP BY wall_posts.id " +
		"ORDER BY CASE WHEN MAX(post_reviews.timestamp) IS NULL THEN wall_posts.time ELSE MAX(post_reviews.timestamp) END DESC "
	s, err := api.sc.Prepare(q)
	if err != nil {
		return
	}
	rows, err := s.Query(userID)
	if err != nil {
		return
	}
	defer rows.Close()
	return api.scanPostRows(rows, false)
}
