package lib

import "github.com/draaglom/GleepostAPI/lib/gp"

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
		pending, err = api.db.PendingPosts(netID)
		if err != nil {
			return
		}
		for i := range pending {
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
	return api.db.ApprovePost(userID, postID, reason)
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
		approved, err = api.db.GetNetworkApproved(netID)
		if err != nil {
			return
		}
		for i := range approved {
			processed, err := api.PostProcess(approved[i].PostSmall)
			if err == nil {
				approved[i].PostSmall = processed
			}
			history, err := api.db.ReviewHistory(approved[i].ID)
			if err == nil {
				approved[i].ReviewHistory = history
			}
		}
		return
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
	return api.db.RejectPost(userID, postID, reason)
}
