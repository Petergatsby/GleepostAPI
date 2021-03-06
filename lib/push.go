package lib

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/Petergatsby/GleepostAPI/lib/gp"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/draaglom/apns"
)

var normRegex = regexp.MustCompile(`<@[\w:]+\|(@\w+)>`)

func normalizeMessage(message string) (textified string) {
	return normRegex.ReplaceAllString(message, "$1")
}

type Push struct {
	Alert   *string     `json:"alert,omitempty"`
	Sound   *string     `json:"sound,omitempty"`
	Message *string     `json:"message,omitempty"`
	Data    interface{} `json:"custom_data,omitempty"`
	Badge   *int        `json:"badge,omitempty"`
}

type wrapper struct {
	APNS        string `json:"APNS,omitempty"`
	APNSSandbox string `json:"APNS_SANDBOX,omitempty"`
	GCM         string `json:"GCM,omitempty"`
	Default     string `json:"default,omitempty"`
}

type pushContainer struct {
	APS Push `json:"aps,omitempty"`
}

func publishToEndpoint(device gp.Device, data *Push) (err error) {
	svc := sns.New(session.New(), &aws.Config{Region: aws.String("us-west-2")})

	// arn is arn:aws:sns:us-west-2:807138844328:endpoint/{service_type}/CampusWire-GP-API-Prod/{uuid} where {service_type} is either APNS or GCM (depending on the type of token received) and {uuid} is the uuid that aws returned in the createplatformendpoint call

	log.Println("Device", device)

	arn := device.ARN

	msg := wrapper{}
	ios := pushContainer{
		APS: *data,
	}
	b, err := json.Marshal(ios)
	if err != nil {
		log.Println("Publishing error", err)
		return err
	}
	msg.APNS = string(b[:])
	// msg.APNSSandbox = string(b[:])
	// msg.Default = msg.Default
	pushData, err := json.Marshal(msg)
	if err != nil {
		log.Println("Publishing marshal error", err)
		return err
	}
	m := string(pushData[:])

	params := &sns.PublishInput{
		Message:          aws.String(m),
		MessageStructure: aws.String("json"),
		TargetArn:        aws.String(arn),
	}
	log.Println("Params", params)
	_, err = svc.Publish(params)
	return
}
func (api *API) messagePush(message gp.Message, convID gp.ConversationID) {
	devices, err := api.pushableDevices(convID)
	if err != nil {
		log.Println("Get pushable devices error", err)
		return
	}
	for _, device := range devices {
		if device.ARN == "" {
			log.Println("No Arn exists for the device")
			continue
		}
		// mentioned := false
		// presence, err := api.Presences.getPresence(device.User)
		// if err != nil && err != noPresence {
		// 	log.Println("Error getting user presence:", err)
		// }
		// if presence.Form == "desktop" && presence.At.Add(30*time.Second).After(time.Now()) {
		// 	log.Println("Not pushing to this user (they're active on the desktop in the last 30s) (push.go)")
		// 	continue
		// }
		// muted, err := api.conversationMuted(device.User, convID)
		// if err != nil {
		// 	log.Println(err)
		// 	continue
		// }
		// mentions := api.spotMentions(message.Text, convID)
		// if mentions.Contains(message.By.ID) {
		// 	mentioned = true
		// }
		log.Println("Device", device)
		log.Println("Message", message)
		// if device.User != message.By.ID && (!muted || mentioned) {
		if device.User != message.By.ID {
			// TODO: Send push notifications here
			// At this point you have a bunch of gp.Device, which have a device.ARN on each of them.
			// You probably want to use (a) normalizeMessage to make our attachment/embed syntax nicer for the notification and
			// (b) use api.badgeCount() to work out what your badge should be
			// You have a profile image available in message.By.Avatar
			// You may have a group ID at message.Group, that's probably important to send if it's there
			normalized := normalizeMessage(message.Text)
			badgeCount, badgeErr := api.badgeCount(device.User)
			if badgeErr != nil {
				log.Println("Error when getting users badge count", badgeErr)
			}
			alertString := fmt.Sprintf("%s: %s", message.By.Name, normalized)
			sound := "default"
			pushObj := &Push{
				Alert: &alertString,
				Badge: &badgeCount,
				Sound: &sound,
			}
			snsErr := publishToEndpoint(device, pushObj)
			if snsErr != nil {
				log.Println("Error when pushing to SNS", snsErr)
			}
			log.Println("Pushed to SNS!!!")
		}
	}
}

func (api *API) pushableDevices(convID gp.ConversationID) (devices []gp.Device, err error) {
	defer api.Statsd.Time(time.Now(), "gleepost.pushable_devices.byConversationID.db")
	s, err := api.sc.Prepare("SELECT participant_id, device_type, device_id, arn FROM conversation_participants JOIN users ON conversation_participants.participant_id = users.id JOIN devices ON participant_id = devices.user_id WHERE conversation_id = ? AND deleted = 0 AND application = 'gleepost'")
	if err != nil {
		return
	}
	rows, err := s.Query(convID)
	if err != nil {
		log.Println("Error getting participant device:", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		device := gp.Device{}
		var arn *sql.NullString
		if err = rows.Scan(&device.User, &device.Type, &device.ID, &arn); err != nil {
			log.Println("Pushable devices scan error", err)
			return
		}
		if arn != nil && arn.Valid {
			device.ARN = arn.String
		}
		devices = append(devices, device)
	}
	return
}

func (api *API) iosPushMessage(device string, message gp.Message, convID gp.ConversationID, user gp.UserID, mentioned bool) (err error) {
	payload := apns.NewPayload()
	d := apns.NewAlertDictionary()
	if mentioned {
		d.LocKey = "mentioned"
	} else {
		d.LocKey = "MSG"
	}
	d.LocArgs = []string{message.By.Name}
	message.Text = normalizeMessage(message.Text)
	if len(message.Text) > 64 {
		d.LocArgs = append(d.LocArgs, message.Text[:64]+"...")
	} else {
		d.LocArgs = append(d.LocArgs, message.Text)
	}
	payload.Alert = d
	payload.Sound = "default"
	payload.Badge, err = api.badgeCount(user)
	pn := apns.NewPushNotification()
	pn.DeviceToken = device
	pn.AddPayload(payload)
	pn.Set("conv", convID)
	if message.Group > 0 {
		pn.Set("group", message.Group)
	}
	pn.Set("profile_image", message.By.Avatar)
	return
}

//FeedbackDaemon checks the APNS feedback service every frequency seconds.
func (api *API) FeedbackDaemon(frequency int) {
	duration := time.Duration(frequency) * time.Second
	c := time.Tick(duration)
	for {
		<-c
		for _, psh := range api.pushers {
			go psh.CheckFeedbackService(api.DeviceFeedback)
		}
	}
}

//SendUpdateNotification sends an update notification to all devices which, when pressed, prompts the user to update if version > installed version.
func (api *API) SendUpdateNotification(userID gp.UserID, message, version, platform string) (count int, err error) {
	if !api.isAdmin(userID) {
		err = ENOTALLOWED
		return
	}
	return api.massNotification(message, version, platform)
}

//MassNotification sends an update notification to all devices which, when pressed, prompts the user to update if version > installed version.
func (api *API) massNotification(message string, version string, platform string) (count int, err error) {
	devices, err := api.getAllDevices(platform)
	if err != nil {
		return
	}
	if len(devices) == 0 {
		return 0, errors.New("no devices on that platform")
	}
	for _, device := range devices {
		switch {
		case device.Type == "ios":
			err = api.iOSUpdateNotification(device, message, version)
			if err == nil {
				count++
			} else {
				log.Println(err)
			}
		default:
		}
	}
	return
}

func (api *API) iOSUpdateNotification(device gp.Device, message string, version string) (err error) {
	payload := apns.NewPayload()
	payload.Alert = message
	payload.Sound = "default"
	payload.Badge, err = api.badgeCount(device.User)
	if err != nil {
		log.Println(err)
	}
	pn := apns.NewPushNotification()
	pn.DeviceToken = device.ID
	pn.AddPayload(payload)
	pn.Set("version", version)
	pusher, ok := api.pushers["gleepost"]
	if ok {
		pusher.IOSPush(pn)
	}
	return
}

func (api *API) badgeCount(user gp.UserID) (count int, err error) {
	count, err = api.userUnreadNotifications(user)
	if err != nil {
		log.Println(err)
		return
	}
	unread, err := api.UnreadMessageCount(user)
	if err == nil {
		count += unread
	} else {
		log.Println(err)
	}
	newPosts, err := api.totalGroupsNewPosts(user)
	if err == nil {
		count += newPosts
	} else {
		log.Println(err)
	}
	err = nil
	return
}

var captureIDRegex = regexp.MustCompile(`<@(\w+)\|@?\w+>`)

func (api *API) spotMentions(message string, convID gp.ConversationID) (mentioned mentioned) {
	m := make(map[gp.UserID]bool)
	participants, err := api.getParticipants(convID, false)
	if err != nil {
		log.Println(err)
	}
	ids := captureIDRegex.FindAllStringSubmatch(message, -1)
	if len(ids) == 0 {
		return
	}
	for _, stringids := range ids {
		for _, stringid := range stringids {
			if stringid == "all" {
				for _, p := range participants {
					m[p.ID] = true
				}
			}
			_id, err := strconv.ParseUint(stringid, 10, 64)
			if err != nil {
				continue
			}
			id := gp.UserID(_id)
			for _, p := range participants {
				if id == p.ID {
					m[id] = true
				}
			}
		}
	}
	for u := range m {
		mentioned = append(mentioned, u)
	}
	return mentioned
}

type mentioned []gp.UserID

func (m mentioned) Contains(u gp.UserID) bool {
	for _, user := range m {
		if u == user {
			return true
		}
	}
	return false
}
