//Package gp contains the core datatypes in Gleepost.
package gp

import (
	"log"
	"time"
)

//UserID is self explanatory.
type UserID uint64

//NetworkID is the id of a network (which Groups are a subset of).
type NetworkID uint64

//MessageID uniquely identifies a chat message.
type MessageID uint64

//PostID uniquely identifies a post (which Events are a subset of).
type PostID uint64

//CommentID identifies a comment on a post.
type CommentID uint64

//ConversationID identifies a conversation.
type ConversationID uint64

//NotificationID identifies a gleepost notification, eg "John Smith commented on your post!"
type NotificationID uint64

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
	ID     UserID `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"profile_image"`
}

//Profile is the fuller representation of a user, containing their tagline, their primary network, their course and their full name (where available)
type Profile struct {
	User
	Desc      string `json:"tagline"`
	Network   Group  `json:"network"`
	Course    string `json:"course"`
	FullName  string `json:"full_name"`
	RSVPCount int    `json:"rsvp_count,omitempty"`
}

//Contact represents a contact relation from the perspective of a particular user, containing the other user and who has accepted the request so far.
type Contact struct {
	User
	YouConfirmed  bool `json:"you_confirmed"`
	TheyConfirmed bool `json:"they_confirmed"`
}

//Network is any grouping of users / posts - ie, a university or a user-created group.
type Network struct {
	ID   NetworkID `json:"id"`
	Name string    `json:"name"`
}

//Group is a user-group. It's a network with a cover image, a description and maybe a creator.
type Group struct {
	Network
	Image   string `json:"image,omitempty"`
	Desc    string `json:"description,omitempty"`
	Creator *User  `json:"creator,omitempty"`
	Privacy string `json:"privacy,omitempty"`
}

//Message is independent of a conversation. If you need that, see RedisMessage.
//TODO: Combine them?
type Message struct {
	ID   MessageID `json:"id"`
	By   User      `json:"by"`
	Text string    `json:"text"`
	Time time.Time `json:"timestamp"`
}

//Read represents the most recent message a user has seen in a particular conversation (it doesn't make much sense without that context).
type Read struct {
	UserID   UserID    `json:"user"`
	LastRead MessageID `json:"last_read"`
}

//RedisMessage is a message with a ConversationID so that someone on the other end of a queue can place it in the correct context.
type RedisMessage struct {
	Message
	Conversation ConversationID `json:"conversation_id"`
}

//Token is a gleepost access token.
//TODO: Add scopes?
//TODO: Deprecate in favour of OAuth?
type Token struct {
	UserID UserID    `json:"id"`
	Token  string    `json:"value"`
	Expiry time.Time `json:"expiry"`
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
}

//PostSmall enhances a Post with a comment count, a like count, and all the users who've liked the post.
type PostSmall struct {
	Post
	CommentCount int        `json:"comment_count"`
	LikeCount    int        `json:"like_count"`
	Likes        []LikeFull `json:"likes,omitempty"`
}

//PostFull enhances a Post with comments and likes.
type PostFull struct {
	Post
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

//Rule represents a condition that makes a user part of a particular Network. At the moment the only possible Rule.Type is "email";
//Rule.Value must then be a domain (eg "gleepost.com") - verified owners of emails in this domain will get added to this network.
type Rule struct {
	NetworkID NetworkID
	Type      string
	Value     string
}

//Conversation is a container for a bunch of messages.
type Conversation struct {
	ID           ConversationID `json:"id"`
	LastActivity time.Time      `json:"lastActivity"`
	Participants []User         `json:"participants"`     //Participants can send messages to and read from this conversation.
	Read         []Read         `json:"read,omitempty"`   //Read represents the most recent message each user has seen.
	Expiry       *Expiry        `json:"expiry,omitempty"` //Expiry is optional; if a conversation does expire, it's no longer accessible.
}

//ConversationSmall only contains the last message in a conversation - for things like displaying an inbox view.
type ConversationSmall struct {
	Conversation
	LastMessage *Message `json:"mostRecentMessage,omitempty"`
}

//ConversationAndMessages contains the messages in this conversation.
type ConversationAndMessages struct {
	Conversation
	Messages []Message `json:"messages"`
}

//MysqlConfig represents the database configuration.
type MysqlConfig struct {
	MaxConns int
	User     string
	Pass     string
	Host     string
	Port     string
}

//RedisConfig represents the cache configuration.
type RedisConfig struct {
	Proto        string
	Address      string
	MessageCache int //Max number of messages per conversation to cache
	PostCache    int //Max number of posts per network to cache
	CommentCache int //Max number of comments per post to cache
}

//AWSConfig contains AWS credentials.
type AWSConfig struct {
	KeyID     string
	SecretKey string
}

//APNSConfig contains Apple push credentials
type APNSConfig struct {
	CertFile   string
	KeyFile    string
	Production bool //Targeting real servers or sandbox?
}

//GCMConfig contains GCM credentials.
type GCMConfig struct {
	APIKey string
}

//EmailConfig contains SMTP credentials.
type EmailConfig struct {
	User       string
	Pass       string
	Server     string
	Port       int
	From       string
	FromHeader string
}

//FacebookConfig contains facebook credentials.
type FacebookConfig struct {
	AppID     string
	AppSecret string
}

//Config defines all the available configuration for the API.
type Config struct {
	DevelopmentMode      bool
	Port                 string
	LoginOverride        bool
	RegisterOverride     bool
	MessagePageSize      int
	PostPageSize         int
	CommentPageSize      int
	ConversationPageSize int
	OnlineTimeout        int
	Expiry               int
	NewPushEnabled       bool
	Admins               int
	Mysql                MysqlConfig
	Redis                RedisConfig
	AWS                  AWSConfig
	APNS                 APNSConfig
	GCM                  GCMConfig
	Email                EmailConfig
	Facebook             FacebookConfig
	Futures              []ConfigFuture
}

//Device is a particular (iOS|Android) device owned by a particular user.
type Device struct {
	User UserID `json:"user"`
	Type string `json:"type"`
	ID   string `json:"id"`
}

//Notification is a gleepost notification which a user may receive based on other users' actions.
type Notification struct {
	ID   NotificationID `json:"id"`
	Type string         `json:"type"`
	Time time.Time      `json:"time"`
	By   User           `json:"user"`
	Seen bool           `json:"seen"`
}

//PostNotification is a Notification that has "happened" at a particular post.
type PostNotification struct {
	Notification
	Post PostID `json:"post"`
}

//GroupNotification is a Notification that has "happened" in a particular group.
type GroupNotification struct {
	Notification
	Group NetworkID `json:"network"`
}

//ConnectionString returns the db/sql string for connecting to MySQL based on this config.
func (c *MysqlConfig) ConnectionString() string {
	return c.User + ":" + c.Pass + "@tcp(" + c.Host + ":" + c.Port + ")/gleepost?charset=utf8"
}

//APIerror is a JSON-ified error.
type APIerror struct {
	Reason string `json:"error"`
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

//Expiry indicates when a conversation is due to expire / whether it has ended yet.
type Expiry struct {
	Time  time.Time `json:"time"`
	Ended bool      `json:"ended"`
}

//NewExpiry creates an expiry d into the future.
func NewExpiry(d time.Duration) *Expiry {
	return &Expiry{Time: time.Now().Add(d), Ended: false}
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
	Value   string
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
	Status string `json:"status"`
	Video
}

//PostFuture represents a commitment to keeping an event's event-time in the future by a specified duration.
type PostFuture struct {
	Post   PostID        `json:"id"`
	Future time.Duration `json:"future"`
}

//ConfigFuture is PostFuture but without the duration because json can't unmarshal it apparently.
type ConfigFuture struct {
	Post   PostID `json:"id"`
	Future string `json:"future"`
}

//ParseDuration converts a ConfigFuture into a PostFuture
func (c ConfigFuture) ParseDuration() (pf PostFuture) {
	pf.Post = c.Post
	duration, err := time.ParseDuration(c.Future)
	if err != nil {
		log.Println("Error parsing duration:", err)
		return
	}
	pf.Future = duration
	return pf
}
