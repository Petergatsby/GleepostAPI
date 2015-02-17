//Package gp contains the core datatypes in Gleepost.
package gp

import "time"

//UserID is self explanatory.
type UserID uint64

//PostID uniquely identifies a post (which Events are a subset of).
type PostID uint64

//NoSuchPost is returned when trying to get a post that doesn't exist (from your perspective)
var NoSuchPost = APIerror{Reason: "No such post"}

//CommentID identifies a comment on a post.
type CommentID uint64

const (
	//OSTART - This resource will be retreived starting at an index position ("posts starting from the n-th")
	OSTART = iota
	//OBEFORE - This resource will be retreived starting from the entries which happened chronologically right before the index.
	OBEFORE
	//OAFTER - Opposite of OBEFORE.
	OAFTER
)

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
	Desc       string          `json:"tagline"`
	Network    GroupMembership `json:"network"`
	Course     string          `json:"course"`
	FullName   string          `json:"full_name"`
	RSVPCount  int             `json:"rsvp_count,omitempty"`
	GroupCount int             `json:"group_count,omitempty"`
	PostCount  int             `json:"post_count,omitempty"`
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

//PostCore is the minimal representation of a post.
type PostCore struct {
	ID   PostID    `json:"id"`
	By   User      `json:"by"`
	Time time.Time `json:"timestamp"`
	Text string    `json:"text"`
}

//Post represents a slightlly fuller representation of a post, containing everything about a post but its potentially limitless number of comments / likes.
type Post struct {
	Network    NetworkID              `json:"-"`
	ID         PostID                 `json:"id"`
	By         User                   `json:"by"`
	Time       time.Time              `json:"timestamp"`
	Text       string                 `json:"text"`
	Images     []string               `json:"images"`
	Videos     []Video                `json:"videos,omitempty"`
	Categories []PostCategory         `json:"categories,omitempty"`
	Attribs    map[string]interface{} `json:"attribs,omitempty"`
	Popularity int                    `json:"popularity,omitempty"`
	Attendees  int                    `json:"attendee_count,omitempty"`
	Group      *Group                 `json:"network,omitempty"`
	Views      int                    `json:"views,omitempty"`
	Attending  bool                   `json:"attending,omitempty"`
}

//PostSmall enhances a Post with a comment count, a like count, and all the users who've liked the post.
type PostSmall struct {
	Post
	CommentCount int        `json:"comment_count"`
	LikeCount    int        `json:"like_count"`
	Likes        []LikeFull `json:"likes,omitempty"`
}

//PendingPost adds review data to a PostSmall
type PendingPost struct {
	PostSmall
	ReviewHistory []ReviewEvent `json:"review_history,omitempty"`
}

//ReviewEvent records something that has happened to a post in review.
type ReviewEvent struct {
	PostID `json:"-"`
	Action string    `json:"action"`
	By     User      `json:"by"`
	Reason string    `json:"reason,omitempty"`
	At     time.Time `json:"at"`
}

//PostFull enhances a Post with comments and likes.
type PostFull struct {
	PendingPost
	CommentCount int        `json:"comment_count"`
	LikeCount    int        `json:"like_count"`
	Comments     []Comment  `json:"comments"`
	Likes        []LikeFull `json:"likes"`
}

//Comment is a comment on a Post.
type Comment struct {
	ID   CommentID `json:"id"`
	Post PostID    `json:"-"`
	By   User      `json:"by"`
	Time time.Time `json:"timestamp"`
	Text string    `json:"text"`
}

//Like represents a user who has liked a post at a particular time.
type Like struct {
	UserID UserID
	Time   time.Time
}

//LikeFull is the same as a like but contains a whole user object rather than an ID.
type LikeFull struct {
	User User      `json:"by"`
	Time time.Time `json:"timestamp"`
}

//Device is a particular (iOS|Android) device owned by a particular user.
type Device struct {
	User UserID `json:"user"`
	Type string `json:"type"`
	ID   string `json:"id"`
}

//APIerror is a JSON-ified error.
type APIerror struct {
	Reason string `json:"error"`
}

//Created is a convenience structure for when you just want to indicate the id of some created resource.
type Created struct {
	ID uint64 `json:"id"`
}

//CreatedPost indicates the ID of a post that's been created, and optionally if it is pending or not.
type CreatedPost struct {
	ID      PostID `json:"id"`
	Pending bool   `json:"pending,omitempty"`
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

//Liked represents a particular post and whether you've liked it.
type Liked struct {
	Post  PostID `json:"post"`
	Liked bool   `json:"liked"`
}

//CategoryID identifies a particular post category/tag.
type CategoryID uint64

//PostCategory represents a particular post category.
type PostCategory struct {
	ID   CategoryID `json:"id"`
	Tag  string     `json:"tag"`
	Name string     `json:"name"`
}

//Error - implements the error interface.
func (e APIerror) Error() string {
	return e.Reason
}

//ENOSUCHUSER is the error that should be returned when performing some action against a non-existent user.
var ENOSUCHUSER = APIerror{"No such user."}

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

type AttendeeSummary struct {
	Popularity    int    `json:"popularity"`
	AttendeeCount int    `json:"attendee_count"`
	Attendees     []User `json:"attendees,omitempty"`
}
