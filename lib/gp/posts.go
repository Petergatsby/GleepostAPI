package gp

import "time"

//PostID uniquely identifies a post (which Events are a subset of).
type PostID uint64

//NoSuchPost is returned when trying to get a post that doesn't exist (from your perspective)
var NoSuchPost = APIerror{Reason: "No such post"}

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
	Poll       *SubjectivePoll        `json:"poll,omitempty"`
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

//PostFull enhances a Post with comments and likes.
type PostFull struct {
	PendingPost
	CommentCount int        `json:"comment_count"`
	LikeCount    int        `json:"like_count"`
	Comments     []Comment  `json:"comments"`
	Likes        []LikeFull `json:"likes"`
}

//CreatedPost indicates the ID of a post that's been created, and optionally if it is pending or not.
type CreatedPost struct {
	ID      PostID `json:"id"`
	Pending bool   `json:"pending,omitempty"`
}

//CategoryID identifies a particular post category/tag.
type CategoryID uint64

//PostCategory represents a particular post category.
type PostCategory struct {
	ID   CategoryID `json:"id"`
	Tag  string     `json:"tag"`
	Name string     `json:"name"`
}

//CommentID identifies a comment on a post.
type CommentID uint64

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

//Liked represents a particular post and whether you've liked it.
type Liked struct {
	Post  PostID `json:"post"`
	Liked bool   `json:"liked"`
}

//AttendeeSummary comprises a list of attending users, a total attendee count (which may not be len(attendees)) and an arbitrary "popularity" score
type AttendeeSummary struct {
	Popularity    int    `json:"popularity"`
	AttendeeCount int    `json:"attendee_count"`
	Attendees     []User `json:"attendees,omitempty"`
}

//Poll contains all the visible information about a poll.
type Poll struct {
	Options []string       `json:"options"`
	Votes   map[string]int `json:"votes"`
	Expiry  time.Time      `json:"expires-at"`
}

//SubjectivePoll is a poll which also contains your vote.
type SubjectivePoll struct {
	Poll
	YourVote string `json:"your-vote,omitempty"`
}

type LiveSummary struct {
	Posts     int            `json:"total-posts"`
	CatCounts map[string]int `json:"by-category"`
}
