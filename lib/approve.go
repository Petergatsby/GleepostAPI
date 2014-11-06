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
