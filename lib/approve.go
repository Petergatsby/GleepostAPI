package lib

import (
	"log"
	"strconv"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/apns"
)

//NoSuchLevelErr happens when you try to set an approval level outside the range [0..3].
var NoSuchLevelErr = gp.APIerror{Reason: "That's not a valid approval level"}

//ApproveAccess returns this user's access to review / change review level in this network.
func (api *API) ApproveAccess(userID gp.UserID, netID gp.NetworkID) (access gp.ApprovePermission, err error) {
	return api.db.ApproveAccess(userID, netID)
}

//ApproveLevel returns this network's current approval level, or ENOTALLOWED if you aren't allowed to see it.
func (api *API) ApproveLevel(userID gp.UserID, netID gp.NetworkID) (level gp.ApproveLevel, err error) {
	return api.db.ApproveLevel(netID)
}

//SetApproveLevel sets this network's approval level, or returns ENOTALLOWED if you can't.
func (api *API) SetApproveLevel(userID gp.UserID, netID gp.NetworkID, level int) (err error) {
	access, err := api.db.ApproveAccess(userID, netID)
	switch {
	case err != nil:
		return err
	case access.LevelChange == false:
		return &ENOTALLOWED
	case level < 0 || level > 3:
		return NoSuchLevelErr
	default:
		current, e := api.db.ApproveLevel(netID)
		switch {
		case e != nil:
			return e
		case current.Level == level:
			//noop
		default:
			err = api.db.SetApproveLevel(netID, level)
			if err == nil {
				//Notifications, etc.
			}
		}
		return
	}

}

//GetNetworkPending returns all the posts which are pending review in this network.
func (api *API) GetNetworkPending(userID gp.UserID, netID gp.NetworkID) (pending []gp.PendingPost, err error) {
	pending = make([]gp.PendingPost, 0)
	access, err := api.db.ApproveAccess(userID, netID)
	switch {
	case err != nil:
		return
	case !access.ApproveAccess:
		return pending, &ENOTALLOWED
	default:
		return api.getNetworkPending(netID)
	}
}

func (api *API) getNetworkPending(netID gp.NetworkID) (pending []gp.PendingPost, err error) {
	_pending, err := api.db.PendingPosts(netID)
	if err != nil {
		return pending, err
	}

	for i := range _pending {
		pending = append(pending, gp.PendingPost{PostSmall: _pending[i]})
		processed, err := api.PostProcess(pending[i].PostSmall)
		if err == nil {
			pending[i].PostSmall = processed
		}
		history, err := api.db.ReviewHistory(pending[i].ID)
		if err == nil {
			pending[i].ReviewHistory = history
		}
	}
	return pending, nil
}

func (api *API) isPendingVisible(userID gp.UserID, postID gp.PostID) (visible bool, err error) {
	p, err := api.db.GetPost(postID)
	if err != nil {
		//Not sure what kinds of errors GetPost will give me, so we'll just say you can't see the post.
		return false, nil
	}
	in, err := api.UserInNetwork(userID, p.Network)
	switch {
	case err != nil:
		return
	case !in:
		return false, &ENOTALLOWED
	default:
		//Is the post still pending?
		pending, _ := api.db.PendingStatus(postID)
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
	p, _ := api.db.GetPost(postID)
	access, _ := api.ApproveAccess(userID, p.Network)
	if !access.ApproveAccess {
		return &ENOTALLOWED
	}
	err = api.db.ApprovePost(userID, postID, reason)
	if err == nil {
		//Notify user their post has been approved
		api.createNotification("approved_post", userID, p.By.ID, uint64(postID))
		//Silently reduce badge count for app users
		//nb: just using p.Network won't work if we eventually want to eg. approve posts in public groups
		api.silentSetApproveBadgeCount(p.Network)
	}
	return
}

//GetNetworkApproved returns the list of approved posts in this network.
func (api *API) GetNetworkApproved(userID gp.UserID, netID gp.NetworkID) (approved []gp.PendingPost, err error) {
	approved = make([]gp.PendingPost, 0)
	access, err := api.ApproveAccess(userID, netID)
	switch {
	case err != nil:
		return
	case !access.ApproveAccess:
		return approved, &ENOTALLOWED
	default:
		_approved, err := api.db.GetNetworkApproved(netID)
		if err != nil {
			return approved, err
		}
		for i := range _approved {
			approved = append(approved, gp.PendingPost{PostSmall: _approved[i]})
			processed, err := api.PostProcess(approved[i].PostSmall)
			if err == nil {
				approved[i].PostSmall = processed
			}
			history, err := api.db.ReviewHistory(approved[i].ID)
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
	p, _ := api.db.GetPost(postID)
	access, _ := api.ApproveAccess(userID, p.Network)
	if !access.ApproveAccess {
		return &ENOTALLOWED
	}
	err = api.db.RejectPost(userID, postID, reason)
	if err == nil {
		api.silentSetApproveBadgeCount(p.Network)
	}
	return
}

//GetNetworkRejected returns the list of rejected posts in this network.
func (api *API) GetNetworkRejected(userID gp.UserID, netID gp.NetworkID) (rejected []gp.PendingPost, err error) {
	rejected = make([]gp.PendingPost, 0)
	access, err := api.ApproveAccess(userID, netID)
	switch {
	case err != nil:
		return
	case !access.ApproveAccess:
		return rejected, &ENOTALLOWED
	default:
		_rejected, err := api.db.GetNetworkRejected(netID)
		if err != nil {
			return rejected, err
		}
		for i := range _rejected {
			rejected = append(rejected, gp.PendingPost{PostSmall: _rejected[i]})
			processed, err := api.PostProcess(rejected[i].PostSmall)
			if err == nil {
				rejected[i].PostSmall = processed
			}
			history, err := api.db.ReviewHistory(rejected[i].ID)
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
	_pending, err := api.db.UserPendingPosts(userID)
	if err != nil {
		return pending, err
	}
	for i := range _pending {
		pending = append(pending, gp.PendingPost{PostSmall: _pending[i]})
		processed, err := api.PostProcess(pending[i].PostSmall)
		if err == nil {
			pending[i].PostSmall = processed
		}
		history, err := api.db.ReviewHistory(pending[i].ID)
		if err == nil {
			pending[i].ReviewHistory = history
		}
	}
	return
}

func (api *API) silentSetApproveBadgeCount(netID gp.NetworkID) {
	posts, err := api.getNetworkPending(netID)
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
		devices, err := api.GetDevices(u.ID, "approve")
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

func (api *API) approveUsers(netID gp.NetworkID) (users []gp.UserRole, err error) {
	master, err := api.db.MasterGroup(netID)
	if err != nil {
		return
	}
	return api.db.GetNetworkUsers(master)
}

func (api *API) postsToApproveNotification(netID gp.NetworkID) {
	posts, err := api.getNetworkPending(netID)
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
		devices, err := api.GetDevices(u.ID, "approve")
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
