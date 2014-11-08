package lib

import "github.com/draaglom/GleepostAPI/lib/gp"

//AddDevice records this user's device for the purpose of sending them push notifications.
func (api *API) AddDevice(user gp.UserID, deviceType string, deviceID string, application string) (device gp.Device, err error) {
	err = api.db.AddDevice(user, deviceType, deviceID, application)
	if err != nil {
		return
	}
	device.User = user
	device.Type = deviceType
	device.ID = deviceID
	return
}

//GetDevices returns all this user's associated devices.
func (api *API) GetDevices(user gp.UserID, application string) (devices []gp.Device, err error) {
	return api.db.GetDevices(user, application)
}

//DeleteDevice removes this user's device (they are no longer able to receive push notifications)
func (api *API) DeleteDevice(user gp.UserID, deviceID string) (err error) {
	return api.db.DeleteDevice(user, deviceID)
}
