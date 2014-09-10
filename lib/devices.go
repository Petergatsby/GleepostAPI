package lib

import "github.com/draaglom/GleepostAPI/lib/gp"

//AddDevice records this user's device for the purpose of sending them push notifications.
func (api *API) AddDevice(user gp.UserID, deviceType string, deviceID string) (device gp.Device, err error) {
	err = api.db.AddDevice(user, deviceType, deviceID)
	if err != nil {
		return
	}
	device.User = user
	device.Type = deviceType
	device.ID = deviceID
	return
}

//GetDevices returns all this user's associated devices.
func (api *API) GetDevices(user gp.UserID) (devices []gp.Device, err error) {
	return api.db.GetDevices(user)
}

//DeleteDevice removes this user's device (they are no longer able to receive push notifications)
func (api *API) DeleteDevice(user gp.UserID, deviceID string) (err error) {
	return api.db.DeleteDevice(user, deviceID)
}
