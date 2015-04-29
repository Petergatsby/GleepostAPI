package lib

import (
	"database/sql"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//AddDevice records this user's device for the purpose of sending them push notifications.
func (api *API) AddDevice(user gp.UserID, deviceType string, deviceID string, application string) (device gp.Device, err error) {
	err = api.addDevice(user, deviceType, deviceID, application)
	if err != nil {
		return
	}
	device.User = user
	device.Type = deviceType
	device.ID = deviceID
	return
}

//GetDevices returns all this user's associated devices.
func getDevices(db *sql.DB, user gp.UserID, application string) (devices []gp.Device, err error) {
	s, err := db.Prepare("SELECT user_id, device_type, device_id FROM devices WHERE user_id = ? AND application = ?")
	if err != nil {
		return
	}
	rows, err := s.Query(user, application)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		device := gp.Device{}
		if err = rows.Scan(&device.User, &device.Type, &device.ID); err != nil {
			return
		}
		devices = append(devices, device)
	}
	return
}

//DeleteDevice removes this user's device (they are no longer able to receive push notifications)
func (api *API) DeleteDevice(user gp.UserID, deviceID string) (err error) {
	return api.deleteDevice(user, deviceID)
}

//DeviceFeedback is called in response to APNS feedback; it records that a device token was no longer valid at this time and deletes it if it hasn't been re-registered since.
func (api *API) DeviceFeedback(deviceID string, timestamp uint32) (err error) {
	t := time.Unix(int64(timestamp), 0)
	return api.feedback(deviceID, t)
}

/********************************************************************
		Device
********************************************************************/

//AddDevice idempotently records user's ios or android device id for pushing notifications to.
func (api *API) addDevice(user gp.UserID, deviceType string, deviceID string, application string) (err error) {
	s, err := api.db.Prepare("REPLACE INTO devices (user_id, device_type, device_id, application) VALUES (?, ?, ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(user, deviceType, deviceID, application)
	return
}

//DeleteDevice deregisters this device (if it exists).
func (api *API) deleteDevice(user gp.UserID, device string) (err error) {
	log.Printf("Deleting %d's device: %s\n", user, device)
	s, err := api.db.Prepare("DELETE FROM devices WHERE user_id = ? AND device_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(user, device)
	return
}

//Feedback deletes the ios device with this ID unless it has been re-registered more recently than timestamp.
func (api *API) feedback(deviceID string, timestamp time.Time) (err error) {
	s, err := api.db.Prepare("DELETE FROM devices WHERE device_id = ? AND last_update < ? AND device_type = 'ios'")
	r, err := s.Exec(deviceID, timestamp)
	n, _ := r.RowsAffected()
	log.Printf("Feedback: %d devices deleted\n", n)
	return
}

//GetAllDevices returns all pushable devices on this platform. Use with caution!
func (api *API) getAllDevices(platform string) (devices []gp.Device, err error) {
	s, err := api.db.Prepare("SELECT user_id, device_type, device_id FROM devices WHERE device_type = ?")
	if err != nil {
		return
	}
	rows, err := s.Query(platform)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		device := gp.Device{}
		if err = rows.Scan(&device.User, &device.Type, &device.ID); err != nil {
			return
		}
		devices = append(devices, device)
	}
	return
}
