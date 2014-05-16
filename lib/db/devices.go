package db

import (
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	//mysql provides the mysql driver as an import side-effect. Not sure if it's actually necessary to import everywhere though.
	_ "github.com/go-sql-driver/mysql"
)

/********************************************************************
		Device
********************************************************************/

//AddDevice idempotently records user's ios or android device id for pushing notifications to.
func (db *DB) AddDevice(user gp.UserID, deviceType string, deviceID string) (err error) {
	s, err := db.prepare("REPLACE INTO devices (user_id, device_type, device_id) VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(user, deviceType, deviceID)
	return
}

//GetDevices returns all a user's devices.
func (db *DB) GetDevices(user gp.UserID) (devices []gp.Device, err error) {
	s, err := db.prepare("SELECT user_id, device_type, device_id FROM devices WHERE user_id = ?")
	if err != nil {
		return
	}
	rows, err := s.Query(user)
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

//DeleteDevice deregisters this device (if it exists).
func (db *DB) DeleteDevice(user gp.UserID, device string) (err error) {
	log.Printf("Deleting %d's device: %s\n", user, device)
	s, err := db.prepare("DELETE FROM devices WHERE user_id = ? AND device_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(user, device)
	return
}

//Feedback deletes the ios device with this ID unless it has been re-registered more recently than timestamp.
func (db *DB) Feedback(deviceID string, timestamp time.Time) (err error) {
	s, err := db.prepare("DELETE FROM devices WHERE device_id = ? AND last_update < ? AND device_type = 'ios'")
	r, err := s.Exec(deviceID, timestamp)
	n, _ := r.RowsAffected()
	log.Printf("Feedback: %d devices deleted\n", n)
	return
}

//GetAllDevices returns all pushable devices on this platform. Use with caution!
func (db *DB) GetAllDevices(platform string) (devices []gp.Device, err error) {
	s, err := db.prepare("SELECT user_id, device_type, device_id FROM devices WHERE device_type = ?")
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
