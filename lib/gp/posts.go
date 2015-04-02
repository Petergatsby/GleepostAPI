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
