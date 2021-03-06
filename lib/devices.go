package lib

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Petergatsby/GleepostAPI/lib/gp"
	"github.com/Petergatsby/GleepostAPI/lib/psc"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

func platformFor(deviceType string) (platform string, err error) {
	switch {
	case deviceType == "ios":
		platform = "APNS"
	case deviceType == "android":
		platform = "GCM"
	default:
		err = errors.New("what?")
	}
	return
}

//AddDevice records this user's device for the purpose of sending them push notifications.
func (api *API) AddDevice(user gp.UserID, deviceType, deviceID, application string) (device gp.Device, err error) {
	device, err = getDevice(api.sc, user, deviceID)
	if err != nil {
		log.Println("Error when getting device when trying to add device", err)
		// return
	}
	if device.ARN == "" {
		platform := ""
		platform, err = platformFor(deviceType)
		if err != nil {
			log.Println("Error when getting platform for device type when trying to add device", err)
			return
		}
		device.ARN, err = api.createEndpoint(deviceID, platform, user)
		if err != nil {
			log.Println("Error when creating SNS endpoint when trying to add device", err)
			return
		}
		api.setDevice(user, deviceType, deviceID, application, device.ARN)
	}
	return
}

func getDevice(sc *psc.StatementCache, user gp.UserID, deviceID string) (device gp.Device, err error) {
	s, err := sc.Prepare("SELECT user_id, device_type, device_id, arn FROM devices WHERE user_id = ? AND device_id = ? LIMIT 1")
	if err != nil {
		return
	}
	var arn sql.NullString
	err = s.QueryRow(user, deviceID).Scan(&device.User, &device.Type, &device.ID, &arn)
	if err != nil {
		return
	}
	log.Println("ARN", arn)
	if arn.Valid {
		device.ARN = arn.String
	}
	return
}

//GetDevices returns all this user's associated devices.
func getDevices(sc *psc.StatementCache, user gp.UserID, application string) (devices []gp.Device, err error) {
	s, err := sc.Prepare("SELECT user_id, device_type, device_id, arn FROM devices WHERE user_id = ? AND application = ?")
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
		var arn *sql.NullString
		if err = rows.Scan(&device.User, &device.Type, &device.ID, &arn); err != nil {
			return
		}
		if arn.Valid {
			device.ARN = arn.String
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

//AddDevice idempotently records user's ios or android device id for pushing notifications to.
func (api *API) setDevice(user gp.UserID, deviceType, deviceID, application, arn string) (err error) {
	s, err := api.sc.Prepare("REPLACE INTO devices (user_id, device_type, device_id, application, arn) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return
	}
	_, err = s.Exec(user, deviceType, deviceID, application, arn)
	return
}

func (api *API) createEndpoint(token, platform string, user gp.UserID) (arn string, err error) {
	svc := sns.New(session.New(), &aws.Config{Region: aws.String("us-west-2")})

	// token is the push token we got off the device
	// application arn is arn:aws:sns:us-west-2:807138844328:app/{service_type}/CampusWire-GP-API-Prod where {service_type} is either APNS or GCM (depending on the type of token received)

	applicationARN := fmt.Sprintf("arn:aws:sns:us-west-2:807138844328:app/%s/CampusWire-GP-API-Prod", platform)

	params := &sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: aws.String(applicationARN),
		Token: aws.String(token),
		Attributes: map[string]*string{
			"Token":          aws.String(token),
			"CustomUserData": aws.String(fmt.Sprintf("%d", user)),
			"Enabled":        aws.String("true"),
		},
		CustomUserData: aws.String(fmt.Sprintf("%d", user)),
	}
	resp, err := svc.CreatePlatformEndpoint(params)
	log.Println("Resp from AWS", resp)
	if err != nil {
		return "", err
	} else {
		return *resp.EndpointArn, nil
	}
}

//DeleteDevice deregisters this device (if it exists).
func (api *API) deleteDevice(user gp.UserID, device string) (err error) {
	log.Printf("Deleting %d's device: %s\n", user, device)
	s, err := api.sc.Prepare("DELETE FROM devices WHERE user_id = ? AND device_id = ?")
	if err != nil {
		return
	}
	_, err = s.Exec(user, device)
	return
}

//Feedback deletes the ios device with this ID unless it has been re-registered more recently than timestamp.
func (api *API) feedback(deviceID string, timestamp time.Time) (err error) {
	s, err := api.sc.Prepare("DELETE FROM devices WHERE device_id = ? AND last_update < ? AND device_type = 'ios'")
	r, err := s.Exec(deviceID, timestamp)
	n, _ := r.RowsAffected()
	log.Printf("Feedback: %d devices deleted\n", n)
	return
}

//GetAllDevices returns all pushable devices on this platform. Use with caution!
func (api *API) getAllDevices(platform string) (devices []gp.Device, err error) {
	s, err := api.sc.Prepare("SELECT user_id, device_type, device_id FROM devices WHERE device_type = ?")
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
