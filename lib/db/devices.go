package db

import (
	"github.com/draaglom/GleepostAPI/lib/gp"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"time"
)

/********************************************************************
		Device
********************************************************************/

func (db *DB) AddDevice(user gp.UserId, deviceType string, deviceId string) (err error) {
	s, err := db.prepare("REPLACE INTO devices (user_id, device_type, device_id) VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(user, deviceType, deviceId)
	return
}

func (db *DB) GetDevices(user gp.UserId) (devices []gp.Device, err error) {
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
		if err = rows.Scan(&device.User, &device.Type, &device.Id); err != nil {
			return
		}
		devices = append(devices, device)
	}
	return
}

func (db *DB) DeleteDevice(user gp.UserId, device string) (err error) {
	log.Printf("Deleting %d's device: %s\n", user, device)
	s, err := db.prepare("DELETE FROM devices WHERE user_id = ? AND device_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(user, device)
	return
}

func (db *DB) Feedback(deviceId string, timestamp time.Time) (err error) {
	s, err := db.prepare("DELETE FROM devices WHERE device_id = ? AND last_update < ? AND device_type = 'ios'")
	r, err := s.Exec(deviceId, timestamp)
	n, _ := r.RowsAffected()
	log.Printf("Feedback: %d devices deleted\n", n)
	return
}

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
		if err = rows.Scan(&device.User, &device.Type, &device.Id); err != nil {
			return
		}
		devices = append(devices, device)
	}
	return
}
