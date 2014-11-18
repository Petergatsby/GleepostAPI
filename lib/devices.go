package lib

import (
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

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

//DeviceFeedback is called in response to APNS feedback; it records that a device token was no longer valid at this time and deletes it if it hasn't been re-registered since.
func (api *API) DeviceFeedback(deviceID string, timestamp uint32) (err error) {
	t := time.Unix(int64(timestamp), 0)
	return api.db.Feedback(deviceID, t)
}
