//Package gp contains the core datatypes in Gleepost.
package gp

import "time"

//UserID is self explanatory.
type UserID uint64

//User is the basic user representation, containing their unique ID, their name and their profile image.
type User struct {
	ID       UserID `json:"id"`
	Name     string `json:"name"`
	Avatar   string `json:"profile_image"`
	Official bool   `json:"official,omitempty"`
}

//Profile is the fuller representation of a user, containing their tagline, their primary network, their course and their full name (where available)
type Profile struct {
	User
	Desc          string          `json:"tagline"`
	Network       GroupSubjective `json:"network"`
	Course        string          `json:"course"`
	FullName      string          `json:"full_name"`
	RSVPCount     int             `json:"rsvp_count,omitempty"`
	GroupCount    int             `json:"group_count,omitempty"`
	PostCount     int             `json:"post_count,omitempty"`
	Type          string          `json:"type,omitempty"`
	Notifications int             `json:"notification_count,omitempty"`
	Unread        int             `json:"unread,omitempty"`
	GroupsBadge   int             `json:"group_badge,omitempty"`
	FBID          uint64          `json:"fb_id,omitempty"`
	InstitutionID string          `json:"institution_id,omitempty"` //nb. institutionID is their university-given ID, not the ID of their university.
	TutorialState []string        `json:"tutorial_state,omitempty"`
}

//PublicProfile contains all the fields about a user which are visible to all users (and are not subjective, like post_count)
type PublicProfile struct {
	User
	Desc          string          `json:"tagline"`
	Network       GroupSubjective `json:"network"` //This isn't actually subjective in this context, but that's dependent on the behaviour of getUserUniversity(). Should probably change types here.
	Course        string          `json:"course"`
	FullName      string          `json:"full_name"`
	Type          string          `json:"type,omitempty"`
	FBID          uint64          `json:"fb_id,omitempty"`
	InstitutionID string          `json:"institution_id,omitempty"` //nb. institutionID is their university-given ID, not the ID of their university.
}

//UserRole represents a user and their role within a particular network
type UserRole struct {
	User
	Role `json:"role"`
}

//ApprovePermission represents a user's ability to access the Approve app and
type ApprovePermission struct {
	ApproveAccess bool `json:"access"`
	LevelChange   bool `json:"settings"`
}

//ApproveLevel indicates the current approval level of this network.
type ApproveLevel struct {
	Level      int      `json:"level"`
	Categories []string `json:"categories"`
}

//ReviewEvent records something that has happened to a post in review.
type ReviewEvent struct {
	PostID `json:"-"`
	Action string    `json:"action"`
	By     User      `json:"by"`
	Reason string    `json:"reason,omitempty"`
	At     time.Time `json:"at"`
}

//APIerror is a JSON-ified error.
type APIerror struct {
	Reason     string `json:"error"`
	StatusCode int    `json:"-"`
}

//Created is a convenience structure for when you just want to indicate the id of some created resource.
type Created struct {
	ID uint64 `json:"id"`
}

//NewUser represents the status of a user as part of the registration process.
type NewUser struct {
	ID     UserID `json:"id"`
	Status string `json:"status"`
}

//URLCreated represents a url you've uploaded.
type URLCreated struct {
	URL string `json:"url"`
}

//BusyStatus is an indication of whether the user is Busy (accepting random chats) or not.
type BusyStatus struct {
	Busy bool `json:"busy"`
}

//Error - implements the error interface.
func (e APIerror) Error() string {
	return e.Reason
}

//ENOSUCHUSER is the error that should be returned when performing some action against a non-existent user.
var ENOSUCHUSER = APIerror{Reason: "No such user."}

//MsgQueue will deliver you a bunch of json-encoded things (internal events or messages sent to the user) through MsgQueue.Messages.
//You can stop listening by sending QueueCommand{"UNSUBSCRIBE", ""} and after a little while the Messages chan should close.
type MsgQueue struct {
	Commands chan QueueCommand
	Messages chan []byte
}

//QueueCommand represents a command to be sent to the message queue. It's sent as is, so never expose this to API clients!
type QueueCommand struct {
	Command string
	Value   []string
}

//WrappedEvent adds some stuff around an Event so that certain shitty javascript websocket can consume it OK.
type WrappedEvent struct {
	Event string `json:"event"`
	Data  Event  `json:"data"`
}

//Event represents something that happened which a consumer of a MsgQueue wants to hear about in real time.
//It has a type, a location (typically a resource) and a json payload.
type Event struct {
	Type     string      `json:"type"`
	Location string      `json:"location,omitempty"`
	Data     interface{} `json:"data"`
}

//Video contains a URL for an .mp4 and .webm encode of the same video, as well as thumbnails where available.
type Video struct {
	//uploaded marks whether this is just a local copy or refers to properly hosted files
	Uploaded bool     `json:"-"`
	ID       VideoID  `json:"id,omitempty"`
	MP4      string   `json:"mp4,omitempty"`
	WebM     string   `json:"webm,omitempty"`
	Thumbs   []string `json:"thumbnails,omitempty"`
	Owner    UserID   `json:"-"`
}

//VideoID is a reference to an uploaded video.
type VideoID uint64

//UploadStatus represents the status of an uploaded video.
type UploadStatus struct {
	ShouldRotate bool   `json:"-"`
	Status       string `json:"status"`
	Video
}
