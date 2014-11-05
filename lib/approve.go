package lib

import "github.com/draaglom/GleepostAPI/lib/gp"

//ApproveAccess returns this user's access to review / change review level in this network.
func (api *API) ApproveAccess(userID gp.UserID, netID gp.NetworkID) (access gp.ApprovePermission, err error) {
	return api.db.ApproveAccess(userID, netID)
}

//ApproveLevel returns this network's current approval level, or ENOTALLOWED if you aren't allowed to see it.
func (api *API) ApproveLevel(userID gp.UserID, netID gp.NetworkID) (level gp.ApproveLevel, err error) {
	return api.db.ApproveLevel(netID)
}
