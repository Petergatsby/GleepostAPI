      ________.__                                       __   
     /  _____/|  |   ____   ____ ______   ____  _______/  |_ 
    /   \  ___|  | _/ __ \_/ __ \\____ \ /  _ \/  ___/\   __\
    \    \_\  \  |_\  ___/\  ___/|  |_> >  <_> )___ \  |  |  
     \______  /____/\___  >\___  >   __/ \____/____  > |__|  
            \/          \/     \/|__|              \/        


#Gleepost API / V1


Production URL: https://gleepost.com/api/v1/
Development URL: https://dev.gleepost.com/api/v1/

##Notes:

* Only available over HTTPS

* Parameters can be form-encoded in a POST body, or sent as a query string

##Compatibility:

The only thing that should be considered a breaking change to the API is the removal or modification of existing attributes in a previously available resource.

A resource is allowed to gain arbitrary new attributes; a client should continue to operate normally, ignoring any attributes it is not familiar with.

In addition, arbitrary new event types may be added to the websocket interface. The client should ignore any event types it is not familiar with.

##Available API endpoints:

###Public endpoints:
These endpoints are accessible to the world.

/register [[POST]](#post-register)

/login [[POST]](#post-login)

/fblogin [[POST]](#post-fblogin)

/profile/request_reset [[POST]](#post-profilerequest_reset)

/profile/reset/[user-id]/[reset-token] [[POST]](#post-profileresetuser-idreset-token)

/profile/facebook [[POST]](#post-profilefacebook).

/verify/[token] [[POST]](#post-verifytoken)

/resend_verification [[POST]](#post-resend_verification)

/contact_form [[POST]](#post-contact_form)

###Authenticated endpoints:
These endpoints require authentication to access.
You must send an <id, token> pair with a request, which you can generate with /login or /fblogin

This may be sent in a query string "?id=1234&token=foobar" (where "1234" and "foobar" are id and token respectively), as parameters in the request body, or in the header "X-GP-Auth" with the format "1234-foobar"

/posts [[GET]](#get-posts) [[POST]](#post-posts) 

/posts/[post-id]/comments [[GET]](#get-postspost-idcomments) [[POST]](#post-postspost-idcomments)

/posts/[post-id] [[GET]](#get-postspost-id) [[PUT]](#put-postspost-id)  [[DELETE]](#delete-postspost-id)

/posts/[post-id]/images [[POST]](#post-postspost-idimages)

/posts/[post-id]/videos [[POST]](#post-postspost-idvideos)

/posts/[post-id]/likes [[POST]](#post-postspost-idlikes)

/posts/[post-id]/attendees [[GET]](#get-postspost-idattendees) [[PUT]](#put-postspost-idattendees)

(DEPRECATED) /posts/[post-id]/attending [[POST]](#post-postspost-idattending) [[DELETE]](#delete-postspost-idattending)

/posts/[post-id]/votes [[POST]](#post-postspost-idvotes)

/networks [[GET]](#get-networks) [[POST]](#post-networks)

/networks/[network-id] [[GET]](#get-networksnetwork-id) [[PUT]](#put-networksnetwork-id)

/networks/[network-id]/posts [[GET]](#get-networksnetwork-idposts) [[POST]](#post-networksnetwork-idposts)

/networks/[network-id]/users [[GET]](#get-networksnetwork-idusers) [[POST]](#post-networksnetwork-idusers)

/networks/[network-id]/admins [[GET]](#get-networksnetwork-idadmins) [[POST]](#post-networksnetwork-idadmins)

/networks/[network-id]/admins/[user-id] [[DELETE]](#delete-networksnetwork-idadminsuser-id)

/networks/[network-id]/requests [[GET]](#get-networksnetwork-idrequests) [[POST]](#post-networksnetwork-idrequests)

/networks/[network-id]/requests/[user-id] [[DELETE]](#delete-networksnetwork-idrequestsuser-id)

/live [[GET]](#get-live)

/live_summary [[GET]](#get-live_summary)

/conversations [[GET]](#get-conversations) [[POST]](#post-conversations)

/conversations/read_all [[POST]](#post-conversationsread_all)

/conversations/mute_badges [[POST]](#post-conversationsmute_badges)

/conversations/[conversation-id] [[GET]](#get-conversationsconversation-id) [[DELETE]](#delete-conversationsconversation-id) [[PUT]](#get-conversationsconversation-id)

/conversations/[coversation-id]/messages [[GET]](#get-conversationsconversation-idmessages) [[POST]](#post-conversationsconversation-idmessages) [[PUT]] (#put-conversationsconversation-idmessages)

/conversations/[conversation-id]/participants [[POST]](#post-conversationsconversation-idparticipants)

/conversation/[conversation-id]/files [[GET]](#get-conversationsconversation-idfiles)

/user [[POST]](#post-user)

/user/[user-id] [[GET]](#get-useruser-id)

/user/[user-id]/posts [[GET]](#get-useruser-idposts)

/user/[user-id]/networks [[GET]](#get-useruser-idnetworks)

/user/[user-id]/attending [[GET]](#get-useruser-idattending)

/ws [[GET]](#get-ws)

/contacts [[GET]](#get-contacts) [[POST]](#post-contacts)

/contacts/[contact-id] [[PUT]](#put-contactsuser)

/devices [[POST]](#post-devices) 

/devices/[device-id] [[DELETE]](#delete-devicesdevice-id)

/upload [[POST]](#post-upload)

/flow_upload [[POST]](#post-flow_upload) [[GET]](#get-flow_upload)

/videos [[POST]](#post-videos) 

/videos/[video-id] [[GET]](#get-videosvideo-id)

/profile/profile_image [[POST]](#post-profileprofile_image)

/profile/name [[POST]](#post-profilename)

/profile/tagline [[POST]](#post-profiletagline)

/profile/change_pass [[POST]](#post-profilechange_pass)

/profile/busy [[POST]](#post-profilebusy) [[GET]](#get-profilebusy)

/profile/facebook [[POST]](#post-profilefacebook)

/profile/attending [[GET]](#get-profileattending)

/profile/pending [[GET]](#get-profilepending)

/profile/networks [[GET]](#get-profilenetworks)

/profile/networks/posts [[GET]](#get-profilenetworksposts)

/profile/networks/[network-id] [[DELETE]](#delete-profilenetworksnetwork-id)

/notifications [[GET]](#get-notifications) [[PUT]](#put-notifications)

/search/users/[name] [[GET]](#get-searchusersname)

/search/groups/[name] [[GET]](#get-searchgroupsname)

/reports [[POST]](#post-reports)

###Statistics endpoints

####Stat endpoints are currently in development. This means they may change in any way at any time for any reason.

/stats/users/[user-id]/posts/[stat-type]/[period]/[start]/[finish] [[GET]](#get-statsusersuser-idpostsstat-typeperiodstartfinish)

/stats/posts/[post-id]/[stat-type]/[period]/[start]/[finish] [[GET]](#get-statspostspost-idstat-typeperiodstartfinish)

/views/posts [[POST]](#post-viewsposts)

###Gleepost Approve endpoints

/approve/access [[GET]](#get-approveaccess)

/approve/level [[GET]](#get-approvelevel) [[POST]](#post-approvelevel)

/approve/pending [[GET]](#get-approvepending)

/approve/approved [[POST]](#post-approveapproved) [[GET]](#get-approveapproved)

/approve/rejected [[POST]](#post-approverejected) [[GET]](#get-approverejected)

##POST /register
required parameters: first, last, pass, email

optional parameters: invite

Password must be at least 5 characters long.

If 'invite' is specified and valid, the user will be added to any groups (s)he has been invited to and will not require verification.

example responses:
If invite is valid:
(HTTP 201)
```json
{"id":143423424, "status":"verified"}
```
If invite is invalid:
(HTTP 201)
```json
{"id":143423424, "status":"unverified"}
```
(HTTP 400)
```json
{"error":"Invalid email"}
```

##POST /login
required parameters: email, pass

Logging in with bad credentials gives HTTP 400.
Logging in with good credentials but an unverified account gives HTTP 403.

example responses:
(HTTP 200) 
```json
{"id":9, "value":"f0e4c2f76c58916ec258f246851bea091d14d4247a2fc3e18694461b1816e13b", "expiry":"2013-09-05T14:53:34.226231725Z"}
```
(HTTP 400)
```json
{"error":"Bad email/password"}
```
(HTTP 403)
```json
{"status":"unverified", "email":"someone@stanford.edu"}
```

##POST /fblogin
required parameters: token
optional parameters: email, invite

Please note: This is in a state of development. Expect it to change frequently.

If this facebook user has an associated, verified gleepost account, this will issue an access token in the same manner as /login.

Alternatively, if invite is supplied and valid the response will also be:
(HTTP 200) 
```json
{"id":9, "value":"f0e4c2f76c58916ec258f246851bea091d14d4247a2fc3e18694461b1816e13b", "expiry":"2013-09-05T14:53:34.226231725Z"}
```

If this facebook user does not have a gleepost account associated, the facebook login will fail and prompt you with:

(HTTP 400)
```json
{"error":"Email required"}
```

In which case you must resubmit the request including the email parameter.

If the email you have provided doesn't have an existing gleepost account registered, this will issue a verification email and respond with:


(HTTP 201)
```json
{"status":"unverified", "email":"someone@stanford.edu"}
```

If the email you have provided is already registered, the response will be:
(HTTP 200)
```json
{"status":"registered", "email":"someone@stanford.edu"}
```

Whereupon the user should be prompted to provide their password to associate their account using [/profile/facebook](#post-profilefacebook).

##GET /posts
required parameters:
id=[user-id]
token=[token]

optional parameters:
start=[count]
returns a list of 20 posts ordered by time, starting at count

before=[id]
after=[id]
returns a list of 20 posts ordered by time, starting before/after [id]

filter=[tag]
Returns only posts belonging to this category tag. 

This is effectively an alias for [/networks/[university-id]/posts](#get-networksnetwork-idposts) which returns the user's university network.

example responses:
(HTTP 200)
```json
[
	{
		"id":2,
		"by": {
			"id":9,
			"name":"Patrick",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		},
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"This is a cool post for cool people!",
		"categories":[{"id":1, "tag":"some_category", "name":"This is a category"}],
		"attribs": {
			"event-time":"2013-09-05T13:09:38Z",
			"location-desc": "1 Jermyn Street",
			"location-gps": "51.509882,-0.133541",
			"location-name": "McKinsey & Co.",
			"title": "Dead Week Grams!"
		},
		"comment_count":4,
		"like_count":5,
		"popularity":75,
		"attendee_count":3,
		"views":123,
		"likes":[{"by": {
				"id":545,
				"name":"SomeoneElse",
				"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
				},
			"timestamp":"2013-09-05T13:09:38Z"},
			{"by": {
				"id":545,
				"name":"SomeoneElse",
				"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
				},
			"timestamp":"2013-09-05T13:09:38Z"}
		],
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"],
		"videos":[
			{
				"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
				"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
				"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
			}
		]
	},
	{
		"id":1,
		"by": {
			"id":23,
			"name":"PeterGatsby",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		},
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"Sailor Moon FTW!",
		"comment_count":9,
		"like_count":0,
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg"],
		"views":123,
		"videos":[
			{
				"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
				"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
				"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
			}
		]
	}
]
```

##POST /posts
required parameters: id, token, text
optional parameters: url, tags, video, poll-expiry, poll-options

If set, url must be a url previously returned from [/upload](#post-upload).
If the image url is invalid, the post will be created without an image. 
If video contains a [valid video ID](#post-videos), the post will be created with a video.

If set, tags must be a comma-delimited list of category "tags". Any of those tags which exist will be added to the post - any which do not exist are silently ignored.

eg:
tags=for-sale,event,salsa

###In addition, any other parameters that are sent when creating the post will be available as the `attribs` object on a post.

Event posts are strongly encouraged to set `event-time`, which represents the time an event begins. This may be either RFC3339 or a unix timestamp.
Event posts may also set a `title`, to be used as a heading.

`event-time` must be in the range `Now()` < `event-time` < `Now() + 2 years`.

`event-time`s which are too soon will trigger an error:
```json
{"error":"Events can not be created in the past"}
```

while `event-time` being too far in the future will return:

```json
{"error":"Events must be within 2 years"}
```

Optionally, you can set `location-name` and/or `location-gps` to specify where an event will be occurring.

If the post is in the category `poll`, you MUST set `poll-expiry` and `poll-options`.

`poll-expiry` indicates when this poll will end, and is a RFC3339 formatted string, eg `2015-04-15T01:05:03Z` OR a Unix timestamp.

`poll-options` is a form encoded list of the options available in this poll. You must specify at least 2 and at most 4 options, and the options must each be 3 <= n <= 50 characters long.
eg: `poll-options=hillary clinton&poll-options=alien kang&poll-options=alien kodos&poll-options=abstain`

If this post requires review before it is published, the response will contain `pending` = `true`.
```json
{"id":3, "pending":true}
```
Otherwise:
(http 200)
```json
{"id":3}
```

If you have provided invalid input when creating a poll, you'll get one of the following errors:

You omitted `poll-expiry` (or it was invalid):
```json
{"error":"Missing parameter: poll-expiry"}
```

`poll-expiry` was in the past
```json
{"error":"Poll ending in the past"}
```

`poll-expiry` was in the future, but within 15 minutes:
```json
{"error":"Poll ending too soon"}
```

`poll-expiry` too far in the future (more than a month + a day away):
```json
{"error":"Poll ending too late"}
```

Less than two `poll-options` provided:
```json
{"error":"Poll: too few options"}
```

more than four `poll-options` provided:
```json
{"error":"Poll: too many options"}
```

The option at index N was too short (less than 3 characters):
```json
{"error":"Option too short: 1"}
```

The option at index N was too long (More than 50 characters):
```json
{"error":"Option too long: 1"}
```

##GET /posts/[post-id]
required parameters: id, token

This returns the full representation of this post, or 403 if the user isn't allowed to view it (ie, it is in a network that you aren't).

example responses:
(http 200)
```json
{
	"id":2,
	"by": {
		"id":9,
		"name":"Patrick",
		"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
	}
	"timestamp":"2013-09-05T13:09:38Z",
	"text":"This is a cool post for cool people!",
	"categories":[{"id":1, "tag":"some_category", "name":"This is a category"}],
	"attribs": {
		"event-time":"2013-09-05T13:09:38Z",
		"location-desc": "1 Jermyn Street",
		"location-gps": "51.509882,-0.133541",
		"location-name": "McKinsey & Co.",
		"title": "Dead Week Grams!"
	},
	"comment_count":4,
	"like_count":5,
	"popularity":25,
	"attendee_count":1,
	"comments": [{
		"id":51341,
		"by": {
			"id":9,
			"name":"Patrick"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		},
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"I concur."
	},
	{
		"id":4362346,
		"by": {
			"id":545,
			"name":"SomeoneElse"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		},
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"Have you ever / ever felt like this? / How strange things happen / like you're going round the twist?"
	}],
	"likes":[{"by": {
			"id":545,
			"name":"SomeoneElse"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
			},
		"timestamp":"2013-09-05T13:09:38Z"},
		{"by": {
			"id":545,
			"name":"SomeoneElse"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
			},
		"timestamp":"2013-09-05T13:09:38Z"}
		],
	"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"],
	"videos":[
		{
			"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
			"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
			"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
		}
	]
}

```

##PUT /posts/[post-id]
optional parameters:
`text` : replaces the body of the post

`url` : replaces the post image

`video` : replaces the post video

`tags` : replaces the post categories

`reason` : describe the changes you made

Any other parameters are used as attribs, just as in post creation.
Providing an attrib you already gave will over-write it; there is currently no way to delete an existing attrib.

Any parameters of the post you do not provide will remain the same.

Returns the updated post in the same format as [GET](#get-postspost-id).
On success, will be 200:
```json
{
	"id":2,
	"by": {
		"id":9,
		"name":"Patrick",
		"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
	}
	"timestamp":"2013-09-05T13:09:38Z",
	"text":"This is a cool post for cool people!",
	"categories":[{"id":1, "tag":"some_category", "name":"This is a category"}],
	"attribs": {
		"event-time":"2013-09-05T13:09:38Z",
		"location-desc": "1 Jermyn Street",
		"location-gps": "51.509882,-0.133541",
		"location-name": "McKinsey & Co.",
		"title": "Dead Week Grams!"
	},
	"comment_count":4,
	"like_count":5,
	"popularity":25,
	"attendee_count":1,
	"comments": [{
		"id":51341,
		"by": {
			"id":9,
			"name":"Patrick"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		},
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"I concur."
	},
	{
		"id":4362346,
		"by": {
			"id":545,
			"name":"SomeoneElse"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		},
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"Have you ever / ever felt like this? / How strange things happen / like you're going round the twist?"
	}],
	"likes":[{"by": {
			"id":545,
			"name":"SomeoneElse"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
			},
		"timestamp":"2013-09-05T13:09:38Z"},
		{"by": {
			"id":545,
			"name":"SomeoneElse"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
			},
		"timestamp":"2013-09-05T13:09:38Z"}
		],
	"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"],
	"videos":[
		{
			"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
			"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
			"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
		}
	]
}
```

##DELETE /posts/[post-id]
required parameters: 

id=[user-id]
token=[token]

On success, returns 204; if you aren't the creator of the post, will return 403.

##GET /posts/[post-id]/comments

required parameters: 

id=[user-id]
token=[token]

optional parameters:
start=[count]

If you are not allowed to view this post, it will return 403.
example responses:
(http 200)
```json
[
	{
		"id":51341,
		"by": {
			"id":9,
			"name":"Patrick",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		},
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"I concur."
	},
	{
		"id":4362346,
		"by": {
			"id":545,
			"name":"SomeoneElse",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		},
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"Have you ever / ever felt like this? / How strange things happen / like you're going round the twist?"
	}
]
```

##POST /posts/[post-id]/comments
required parameters: id, token, text

example responses:
(http 200)
```json
{"id":234}
```

If you provide a zero-length text:
(http 400)
```json
{"error":"Comment too short"}
```

##POST /posts/[post-id]/images
required parameters: id, token, url

This adds an image previously uploaded with [/upload](#post-upload) to this post.

example responses:
(http 201)
```json
["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"]
```

##POST /posts/[post-id]/videos
required parameters: id, token, video

This adds a video to this post and returns a list of all this post's videos (although this is limited to one) or 403 f you aren't the post's creator..

(HTTP 201)
```
[
	{
		"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
		"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
		"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
	}
]
```

##POST /posts/[post-id]/likes
required parameters: id, token, liked

[liked] must be a boolean.
If true, adds a like for this post for this user.
If false, removes a like for this post for this user.

If this post is in another network, will respond with 403.

example responses:
(http 200)
```json
{"post":5, "liked":true}
```
```json
{"post":5, "liked":false}
```

##GET /posts/[post-id]/attendees
Returns the popularity, attendee-count and full list of attendees of an event.

```json
{
    "popularity": 0,
    "attendee_count": 0,
    "attendees": [{
			"id":9,
			"name":"Patrick"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}]
}
```

##PUT /posts/[post-id]/attendees
Required parameters:
attending = (true|false)

`attending=true` marks the current user as attending this event.
`attending=false` cancels the attendance.

It returns the updated popularity, attendee_count and attendees list.
```json
{
    "popularity": 0,
    "attendee_count": 0,
    "attendees": [{
			"id":9,
			"name":"Patrick"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}]
}
```

##POST /posts/[post-id]/attending
##Deprecated. Please use [/attendees](#put-postspost-idattendees) instead.
required parameters: id, token

Issuing a POST to this URI should mark you as attending this event, and acts idempotently.
It will return a 204 if successful.

##DELETE /posts/[post-id]/attending
##Deprecated. Please use [/attendees](#put-postspost-idattendees) instead.
required parameters: id, token

Issuing a DELETE to this URI should mark you as not attending this event.
It should succeed even if you aren't already attending.
It will return a 204 if successful.

##POST /posts/[post-id]/votes

Required parameters:

`option` = `0`, `1`, `2`, `3`

If successful, will respond with a 204.

If this post is not, in fact, a poll, you will get a 400:
```json
{"error":"Not a poll"}
```

If the option you have specified is not valid, eg. option=3 when there are 3 poll options (index starts at 0):
```json
{"error":"Invalid option"}
```

If you have already voted:
```json
{"error":"You already voted"}
```

If the poll has ended already:
```json
{"error":"Poll has already ended"}
```

##GET /live
required parameters: `id`, `token`, `after`

Optional parameters: `until`, `filter`

`after` and `until` must be either an RFC3339 formatted time string, or a unix timestamp.

If `filter` is provided, it will only return posts in this category.

Live returns the 20 events whose event-time is soonest after `after`, which are happening before `until`.

example responses:
(http 200)
```json
[
	{
	"id":763,
	"by":{"id":2395,"name":"TestingUser","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/260a8e71eb2dbfed25b0a0de5ae328cdfc931c5023668955ba660e61705c6800.jpg"},
	"timestamp":"2014-01-31T09:43:28Z",
	"text":"Event 1",
	"images":null,
	"videos":[
		{
			"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
			"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
			"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
		}
	],
	"attribs": {
		"event-time":"2013-09-05T13:09:38Z",
		"location-desc": "1 Jermyn Street",
		"location-gps": "51.509882,-0.133541",
		"location-name": "McKinsey & Co.",
		"title": "Dead Week Grams!"
	},
	"popularity":25,
	"attendee_count":1,
	"comment_count":0,
	"like_count":0
	},
	{
	"id":760,
	"by":{"id":2395,"name":"TestingUser","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/260a8e71eb2dbfed25b0a0de5ae328cdfc931c5023668955ba660e61705c6800.jpg"},
	"timestamp":"2014-01-29T18:05:16Z",
	"text":"New event after bug!",
	"images":null,
	"videos":[
		{
			"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
			"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
			"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
		}
	],
	"attribs": {
		"event-time":"2013-09-05T13:09:38Z",
		"location-desc": "1 Jermyn Street",
		"location-gps": "51.509882,-0.133541",
		"location-name": "McKinsey & Co.",
		"title": "Dead Week Grams!"
	},
	"popularity":100,
	"attendee_count":5,
	"comment_count":0,
	"like_count":1,
	"likes":[{"by":{"id":2395,"name":"TestingUser","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/260a8e71eb2dbfed25b0a0de5ae328cdfc931c5023668955ba660e61705c6800.jpg"},
	"timestamp":"2014-02-05T07:00:54Z"}]
	}
]
```

##GET /live_summary

required parameters:
`id`, `token`, `after`, `until`

`after` and `until` must be either an RFC3339 formatted time string, or a unix timestamp.

This endpoint summarizes the state of the Campus Live (ie, upcoming events) between the two times `after` and `until`.

Note that the contents of `by-category` are not expected to sum to `total-posts`; an event may be in several categories (eg. `event` and `party`) and therefore be counted in several categories.

```json
{
	"total-posts":123,
	"by-category":{
		"party":43,
		"sports":12,
		"food":68,
	}
}
```

##GET /networks/[network-id]
required parameters:
id=[user-id]
token=[token]

A group resource, or 403 if you aren't a member of the group.
example responses (http 200):

```json
{
	"id":5345, 
	"name":"Super Cool Group", 
	"description":"Pretty cool, no?", 
	"url":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg",
	"creator":{
		"id":2491,
		"name":"Patrick",
		"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg"
	},
	"size":1234,
	"conversation":5678,
	"unread":12
}
```

##PUT /networks/[network-id]
required parameters:
id=[user-id]
token=[token]

url="URL returned from /upload"

If you created this group, you can change the group's image. If you didn't create the group -- or you didn't choose a valid image URL - it will return 403. Otherwise, returns the updated resource.

```json
{
	"id":5345, 
	"name":"Super Cool Group", 
	"description":"Pretty cool, no?", 
	"url":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg",
	"creator":{
		"id":2491,
		"name":"Patrick",
		"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg"
	},
	"size":1234,
	"conversation":5678,
	"role": {
		"name":"member",
		"level":1
	}
}
```

##GET /networks

Returns a list of 20 of the networks which are visible to you, ordered by popularity (number of members).

`role` indicates, when present, your membership status within the group.

`pending_request` will be `true` if you have an outstanding request to join this network.

optional parameters:

`start`: the pagination offset for the list

Response:

(http 200)
```json
[
  {
    "id": 2,
    "name": "funsies",
    "creator": {
      "id": 2,
      "name": "Beetle",
      "profile_image": ""
    },
    "privacy": "private",
    "size": 12345,
    "role": {
	"level":1,
	"name":"member",
    }
  }
]
```

##POST /networks
required parameters:

`name` = "Name of the group"

optional:

`desc` = "Description of the group"

`url` = uploaded image URL, the group cover image

If `url` is not valid, it will respond with a 403.

`privacy` = "public", "private" or "secret"

if privacy is not provided, it will default to "private".

`category` = `sports`, `social`, `academic`, `dorm`, `career`, `official`

`university` = `boolean` 

If set to `true`, this will create a new University, configured to accept users registering with a domain in the list `domains`; you must be an administrator to do this.

If set to `false` (default), the network created is a user group, and you are made a member.

`domains` = `universitya.edu,universityb.ac.uk`

A successful response is 201:

```json
{
	"id":5345, 
	"name":"Even Cooler Group", 
	"description":"Pretty cool, no?",
	"url":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg", 
	"creator":{"id":2491,"name":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg"},
	"size":1,
	"conversation":5678
}
```

##GET /networks/[network-id]/posts
required parameters:
id=[user-id]
token=[token]

optional parameters:
start=[count]
returns a list of 20 posts ordered by time, starting at count

before=[id]
after=[id]
returns a list of 20 posts ordered by time, starting before/after [id]

filter=[tag]
Returns only posts belonging to this category tag. 

This returns all the posts in this network, or an error 403 if the user is not allowed to view the posts in this network.

example responses:
(HTTP 200)
```json
[
	{
		"id":2,
		"by": {
			"id":9,
			"name":"Patrick",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"This is a cool post for cool people!",
		"categories":[{"id":1, "tag":"some_category", "name":"This is a category"}],
		"attribs": {
			"event-time":"2013-09-05T13:09:38Z",
			"location-desc": "1 Jermyn Street",
			"location-gps": "51.509882,-0.133541",
			"location-name": "McKinsey & Co.",
			"title": "Dead Week Grams!"
		},
		"comment_count":4,
		"like_count":5,
		"attendee_count":324,
		"attending":true,
		"popularity":75,
		"likes":[{"by": {
				"id":545,
				"name":"SomeoneElse"
				"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
				},
			"timestamp":"2013-09-05T13:09:38Z"},
			{"by": {
				"id":545,
				"name":"SomeoneElse"
				"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
				},
			"timestamp":"2013-09-05T13:09:38Z"}
		],
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"],
		"videos":[
			{
				"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
				"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
				"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
			}
		]
	},
	{
		"id":1,
		"by": {
			"id":23,
			"name":"PeterGatsby"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"Sailor Moon FTW!"
		"comment_count":9,
		"like_count":0,
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg"],
		"videos":[
			{
				"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
				"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
				"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
			}
		]
	}
]
```

##POST /networks/[network-id]/posts
Create a post in this network.

required parameters: id, token, text
optional parameters: url, tags 

If set, url must be a url previously returned from [/upload](#post-upload).
If the image url is invalid, the post will be created without an image. 

If set, tags must be a comma-delimited list of category "tags". Any of those tags which exist will be added to the post - any which do not exist are silently ignored.

eg:
tags=for-sale,event,salsa

###In addition, any other parameters that are sent when creating the post will be available as an "attribs" object within a post.

Event posts are strongly encouraged to set "event-time", which represents the time an event begins. This may be either RFC3339 or a unix timestamp.
Event posts may also set an "title", to be used as a heading.

If you are not allowed, will respond with 403.
If successful, will respond with HTTP 201
```json
{"id":345}
```

##GET /networks/[network-id]/users
required parameters:
id=[user-id]
token=[token]

A collection of all the users and their role (permissions) in this network, or 403 if you aren't a member of the network (or if it is a university network)
Example response:
```json
[{"id":2395,"name":" Younes","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/73f2d43f3b58838712f40a0a0f9b39fc6d589661ef3eb44f395773c1f7817165.jpg","role":{"name":"administrator","level":8}},{"id":2491,"name":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg","role":{"name":"creator","level":9}},{"id":2563,"name":"Auth","profile_image":"https://graph.facebook.com//picture?type=large","role":{"name":"member","level":1}}]
```

##POST /networks/[network-id/users
required parameters:
id=[user-id]
token=[token]

One or more of:
users=[other-user-id],[other-user-id],[other-user-id]

fbusers=[facebook-id],[facebook-id],[facebook-id]

email=[other-user-email]

Adds other users to this network, or records that they have been invited via facebook, or emails them an invite if they aren't on Gleepost. On success will return 204.

##GET /networks/[network-id]/admins
A collection of all the administrators of this network, or 403 if you are not a member.
```json
[
	{
		"id": 2395,
		"name": "Younes",
		"profile_image": "https://s3-eu-west-1.amazonaws.com/gpimg/73f2d43f3b58838712f40a0a0f9b39fc6d589661ef3eb44f395773c1f7817165.jpg",
		"role": {
			"name": "administrator",
			"level": 8
		}
	}
]
```

##POST /networks/[network-id]/admins
Make group member(s) into admins.
parameters:

users=[user-id],[user-id],[user-id],...

where each user-id is already a member of this network.

Returns the updated admin list.
```json
[
	{
		"id": 2395,
		"name": "Younes",
		"profile_image": "https://s3-eu-west-1.amazonaws.com/gpimg/73f2d43f3b58838712f40a0a0f9b39fc6d589661ef3eb44f395773c1f7817165.jpg",
		"role": {
			"name": "administrator",
			"level": 8
		}
	}
]
```

##DELETE /networks/[network-id]/admins/[user-id]
Delete administrative permissions for this user. You must be an administrator or group creator to use.
If you are allowed to downgrade this user, the result will be 204.

##GET /networks/[network-id]/requests
List the outstanding requests to join this network.

```json
[
	{
		"requester": {
			"id": 2395,
			"name": "Younes",
			"profile_image": "https://s3-eu-west-1.amazonaws.com/gpimg/73f2d43f3b58838712f40a0a0f9b39fc6d589661ef3eb44f395773c1f7817165.jpg"
		},
		"requested-at":"2014-01-31T09:43:28Z",
		"status":"pending"
	}
]
```

##POST /networks/[network-id]/requests
Request access to this group.

If the network you have requested does not exist (or you cannot see it) the result will be a 404:
```json
{"error": "No such network"}
```

If the network is visible to you but you cannot request access to it (because it is public, a university, or you are already a member) the result will be 403:
```json
{"error": "You're not allowed to do that!"}
```

On success, the response will be 201.

##DELETE /networks/[network-id]/requests/[user-id]

If you are an administrator of this group, you can reject a request to join the group. The request will no longer be visible in the `/networks/:id/requests` list.

Attempting to reject a user who has not made a request will result in a 404:
```json
{"error":"No such request"}
```

Attempting to reject a request in a group which does not exist will result in a 404:
```json
{"error":"No such network"}
```

Attempting to reject a request in a group in which you are not staff (admin/creator) or not a member of will result in a 403:
```json
{"error":"You're not allowed to do that!"}
```

Attempting to reject a request which is already accepted / rejected will result in a 403:

```json
{"error":"Request is already accepted"}
```
```json
{"error":"Request is already rejected"}
```

On success, the response will be a 204.

##POST /conversations/read_all
required parameters:
id=[user-id]
token=[token]

Marks all conversations as "seen".
On success, will return a 204 (no content).

##POST /conversations/mute_badges
required parameters:
`id` = `[user-id]`
`token` = `[token]`

mute_badges marks all unread messages before the current time to be ignored from any badge calculations.

##GET /conversations
required parameters:
id=[user-id]
token=[token]

optional parameters:
start=[count]

returns a list of 20 of your conversations ordered by most recent message, starting at count
```json
[
	{"id":1,
	"participants": [
		{
			"id":9,
			"name":"Patrick",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png",
			"presence":{
				"form":"mobile",
				"at":"2015-06-02T15:58:00Z"
			}
		},
		{
			"id":23,
			"name":"PeterGatsby",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
	],
	"read":[{"user":9,"last_read":1000,"at":"2015-06-02T17:35:00Z"}],
	"lastActivity":"2013-09-05T13:09:38Z",
	"mostRecentMessage": {"id":1234214, "by":{"id":9, "name":"Patrick"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
	"expiry": { "time": "2013-11-13T22:11:32.956855553Z", "ended":false },
	"unread": 123
	},
	{"id":2,
	"participants": [
		{"id":99999, "name":"Lukas", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":232515, "name":"Ling", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}
	],
	"lastActivity":"2013-09-05T13:09:38Z",
	"mostRecentMessage": {"id":123512624, "by":{"id":99999, "name":"Lukas", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}, "text":"idk lol", "timestamp":"2013-09-05T13:09:38Z"},
	"unread": 123
	}
]
```

##POST /conversations
required parameters:
id=[user-id]
token=[token]

participants=[user_id],[user_id],[user_id],...
(a comma-delimited list of up to 50 user_ids to start a conversation with.)

If started with exactly 1 other participant, it will only create a new conversation if you do not already have one with this participant. Otherwise, it will create a new conversation.

example responses:
(HTTP 200)
```json
{
	"id":1,
	"participants": [
		{"id":9, "name":"Patrick", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":23, "name":"PeterGatsby", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}
	],
	"messages": [
		{"id":1234214, "by":{"id":23, "name":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
		{"id":1234214, "by":{"id":23, "name":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
		{"id":1234214, "by":{"id":23, "name":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"}
	],
	"lastActivity":"2013-09-05T13:09:38Z"
}
```

##GET /conversations/[conversation-id]
required parameters:
id=[user-id]
token=[token]

example responses:
(HTTP 200)
```json
{
	"id":1,
	"participants": [
		{"id":9, "name":"Patrick", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":23, "name":"PeterGatsby", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}
	],
	"read":[{"user":9,"last_read":1000, "at":"2013-09-05T13:09:38Z"}],
	"messages": [
		{"id":1234214, "by":{"id":23, "name":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
		{"id":1234214, "by":{"id":23, "name":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
		{"id":1234214, "by":{"id":23, "name":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"}
	],
	"lastActivity":"2013-09-05T13:09:38Z",
	"unread": 123
}
```

##PUT /conversations/[conversation-id]
requred parameters:
muted = `true|false`

Set `muted` = `true` to suppress any push notifications from this conversation; `muted` = `false` to enable them again.

Responds with the full conversation like [[GET /conversations/:id]](#get-conversationsconversation-id).

(HTTP 200)
```json
{
	"id":1,
	"participants": [
		{"id":9, "name":"Patrick", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":23, "name":"PeterGatsby", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}
	],
	"read":[{"user":9,"last_read":1000, "at":"2013-09-05T13:09:38Z"}],
	"messages": [
		{"id":1234214, "by":{"id":23, "name":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
		{"id":1234214, "by":{"id":23, "name":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
		{"id":1234214, "by":{"id":23, "name":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"}
	],
	"lastActivity":"2013-09-05T13:09:38Z",
	"unread": 123,
	"muted": true
}
```
##DELETE /conversations/[conversation-id]
required parameters:
id=[user-id]
token=[token]

This removes a conversation from your inbox. You will no longer be able to send messages to it, no longer receive notifications, and can no longer view it.

If it is successful, it will respond with HTTP 204.

##GET /conversations/[conversation-id]/messages
required parameters: id=[user-id], token=[token]
optional parameters: start=[start], after=[after], before=[before]

Returns a list of 20 messages ordered by time from most recent to least recent.
Given [start], it returns messages from the [start]th most recent to [start + 20]th most recent.
Given [after], it returns at most 20 of the messages received since [after]
Given [before], it returns at most 20 of the messages received immediately before [before]

example responses:
```json
[
		{"id":1234214, "by":{"id":99999, "name":"Lukas", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
		{"id":1234214, "by":{"id":99999, "name":"Lukas", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
		{"id":1234214, "by":{"id":99999, "name":"Lukas", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"}
]
```

##POST /conversations/[conversation-id]/messages
required parameters: id, token, text

example responses:
```json
{"id":1356}
```

##PUT /conversations/[conversation-id]/messages
required parameters: id, token, seen

Marks all messages in a conversation up to [seen] 
(that were not sent by the current user) seen.

example responses:

seen=51
(HTTP 200)
```json
{
	"id": 5,
	"participants": [
		{
			"id": 9,
			"name": "Patrick",
			"profile_image": "https://gleepost.com/uploads/avatar.png"
		},
		{
			"id": 1327,
			"name": "Meg",
			"profile_image": "",
		}
	],
	"lastActivity":"2013-09-05T13:09:38Z",
	"expiry": { "time": "2013-11-13T22:11:32.956855553Z", "ended":false },
	"unread":123,
	"messages": [
		{
			"id": 52,
			"by": {
				"id": 9,
				"name": "Patrick",
				"profile_image": "https://gleepost.com/uploads/bad2cbd1431260c2c4b9766ae5de25d6.gif",
			},
			"text": "sup",
			"timestamp": "2013-09-16T16:58:23Z"
		},
		{
			"id": 51,
			"by": {
				"id": 9,
				"name": "Patrick",
				"profile_image": "https://gleepost.com/uploads/bad2cbd1431260c2c4b9766ae5de25d6.gif",
			},
			"text": "sup",
			"timestamp": "2013-09-16T16:58:30Z"
		}
	]
}


```

##POST /conversations/[conversation-id]/participants

Required parameters:
`id`, `token` (Auth)

`users`: a comma-delimited list of userIDs to add as participants to this conversation.

On success, returns the updated list of participants. Note: This may be different to the list you were expecting, if eg. one of the users could not be added to the conversation

```json
[
	{
		"id":9,
		"name": "Patrick",
		"profile_image":"https://gleepost.com/uploads/123.jpg"
	},
	{
		"id":9999,
		"name": "Jeff",
		"profile_image":"https://gleepost.com/uploads/456.jpg"
	}
]
```

In addition, this will trigger a "system" message in this conversation indicating that the user has joined the conversation:

```json
{
	"id":1234123,
	"by":{
		"id":123,
		"name":"Patrick Molgaard",
		"profile_image":"https://gleepost.com/uploads/foo.png"
	},
	"text":"JOINED",
	"system":true,
	"timestamp":"2015-04-20T16:19:30Z"
}
```

##GET /conversations/[conversation-id]/files

A list of files shared in this conversation.
(http 200)

```json
[
  {
    "url": "https://file.host",
    "type": "pdf",
    "message": {
      "id": 1,
      "by": {
        "id": 1,
        "name": "Patrick",
        "profile_image": ""
      },
      "text": "hey here's a file: <https://file.host|pdf>",
      "timestamp": "2015-06-29T18:29:18Z"
    }
  }
]
```

##POST /user
Use this to generate a new user in a particular network.


Required parameters:
first, last, email, pass, verified, network-id


where verified is a boolean and network-id is the network that this user will be created in.

Success is a 204.

##GET /user/[user-id]
required parameters:
id=[user-id]
token=[token]

You are only allowed to view a user's profile if they share a network with you. Attempting to access a profile resource of a user you share no networks with will result in a 403 error.

A user (such as an official university account) may have the parameter `official`. This indicates that they are deemed official by the university.

If a user is not official, the parameter will be omitted.

The `official` parameter is visible anywhere you might see a User object.

example responses:
```json
{
	"id":9,
	"name":"Patrick",
	"official":true,
	"tagline":"I like computers",
	"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png",
	"network": { "id":1, "name":"University of Leeds" },
	"course":"Computer Science",
	"full_name":"Patrick Molgaard",
	"rsvp_count":234,
	"group_count":567,
	"type":"student",
	"post_count":8910
}
```

##GET /user/[user-id]/posts
required parameters:
id=[user-id]
token=[token]

optional parameters:
start=[count]
returns a list of 20 posts ordered by time, starting at count

before=[id]
after=[id]
returns a list of 20 posts ordered by time, starting before/after [id]

filter = "category"
returns only posts matching that category
example responses:
```json
[
	{
		"id":2,
		"by": {
			"id":9,
			"name":"Patrick",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"This is a cool post for cool people!",
		"categories":[{"id":1, "tag":"some_category", "name":"This is a category"}],
		"attribs": {
			"event-time":"2013-09-05T13:09:38Z",
			"location-desc": "1 Jermyn Street",
			"location-gps": "51.509882,-0.133541",
			"location-name": "McKinsey & Co.",
			"title": "Dead Week Grams!"
		},
		"attending":true,
		"comment_count":4,
		"like_count":5,
		"likes":[{"by": {
				"id":545,
				"name":"SomeoneElse"
				"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
				},
			"timestamp":"2013-09-05T13:09:38Z"},
			{"by": {
				"id":545,
				"name":"SomeoneElse"
				"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
				},
			"timestamp":"2013-09-05T13:09:38Z"}
		],
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"],
		"videos":[
			{
				"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
				"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
				"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
			}
		]
	},
	{
		"id":1,
		"by": {
			"id":9,
			"name":"Patrick"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"Sailor Moon FTW!"
		"comment_count":9,
		"like_count":0,
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg"],
		"videos":[
			{
				"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
				"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
				"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
			}
		]
	}
]

```

##GET /user/[user-id]/networks
Lists this user's groups - if you're allowed to see them. Or 403 otherwise.
Secret groups are hidden.

If there are more than 20 results, this resource will return the first 20.

Results can be paginated by supplying `start` = `n` to offset the results by `n` groups.

`their_role` indicates this user's membership status within the group; `role`, where available, is yours (the viewing user's.

If `pending_request` is present, this indicates you have an outstanding request to join this group.

Example response: (http 200)
```json
[
	{
		"id":5345, 
		"name":"Even Cooler Group", 
		"description":"Pretty cool, no?", 
		"url":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg", 
		"creator": {
			"id":2491,
			"name":"Patrick",
			"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg"
		},
		"privacy":"private",
		"their_role": {
			"name":"member",
			"level":1
		},
	        "pending_request":true
	}
]
```

##GET /user/[user-id]/attending
Lists the events that this user is attending, most recently attended first. Only the events in groups / networks you can see.

Paginated in the same way as [posts](#get-posts).

```json
[
	{
		"id":2,
		"by": {
			"id":9,
			"name":"Patrick",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"This is a cool post for cool people!",
		"categories":[{"id":1, "tag":"some_category", "name":"This is a category"}],
		"attribs": {
			"event-time":"2013-09-05T13:09:38Z",
			"location-desc": "1 Jermyn Street",
			"location-gps": "51.509882,-0.133541",
			"location-name": "McKinsey & Co.",
			"title": "Dead Week Grams!"
		},
		"comment_count":4,
		"like_count":5,
		"attending":true,
		"likes":[{"by": {
				"id":545,
				"name":"SomeoneElse"
				"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
				},
			"timestamp":"2013-09-05T13:09:38Z"},
			{"by": {
				"id":545,
				"name":"SomeoneElse"
				"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
				},
			"timestamp":"2013-09-05T13:09:38Z"}
		],
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"],
		"videos":[
			{
				"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
				"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
				"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
			}
		]
	},
	{
		"id":1,
		"by": {
			"id":9,
			"name":"Patrick"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"Sailor Moon FTW!"
		"comment_count":9,
		"like_count":0,
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg"],
		"videos":[
			{
				"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
				"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
				"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
			}
		]
	}
]
```

##POST /newconversation

DEPRECATED, use [[/conversations]](#post-conversations)

##POST /newgroupconversation

DEPRECATED, use [[/conversations]](#post-conversations)

##GET /ws
Required parameters:
id=[user-id]
token=[token]

See [the websockets readme.](websockets.md)

##POST /devices
required parameters: `type`, `device_id`

optional parameters: `application` 

Type should be "android" or "ios"

`application` defaluts to "gleepost"; gleepost approve users should specify "approve".

This registers the push notification id "device_id" for the current user

example responses: 
HTTP 201
```json
{
	"user":2395,
	"type":"android",
	"id":"APA91bFmOKOcm6v1ZJVavmvHQ3SLzADznBHhT6gDdNUDZm9wSc-yBdToyAWtR73cro5rnemVTiXdqQMlqmrs_4mdAhZbiLIfeZ4cD4L9OstvTnjzv8-Yx_fSPM1Joe_gpAEe0haNEwh3pSQah1QQQFC829jA7V-vswpuQLmLT2sK_ciMo5Hx7po"
}
```

##DELETE /devices/[device-id]
required parameters: id, token

This will stop [device-id] receiving push notifications for this user.

If successfull, the response will be:
HTTP 204
(no content)

##POST /upload
required parameters: id, token

optional parameters: `image` or `file`.

/upload expects a single multipart/form-data encoded image or file and on success will return a url.

example responses:
HTTP 201
```json
{"url":"https://s3-eu-west-1.amazonaws.com/gpimg/3acd82c15dd0e698fc59c79e445a464553e57d338a6440601551c7fb28e45bf9.jpg"}
```

##POST /flow_upload
required parameters: id, token

/flow_upload implements flow.js style chunked upload. A completed upload responds in the same fashion as /upload:

HTTP 201
```json
{"url":"https://s3-eu-west-1.amazonaws.com/gpimg/3acd82c15dd0e698fc59c79e445a464553e57d338a6440601551c7fb28e45bf9.jpg"}
```

##GET /flow_upload
required parameters: id, token

Returns the status of a flow.js upload chunk.

##POST /videos

required parameters: `id`, `token`, `video`

optional parameters: `rotate`

/video takes a single multipart/form-data encoded video and returns an id and a status ("uploaded").
You can then check [its resource](#get-videosvideo-id) to discover when it is ready to be used.
In addition, when the video has uploaded you will get a "video-ready" event if you have a websocket connection.

If `rotate` is `true`, the output webm will be rotated 90 degrees clockwise.

HTTP 201
```json
{"status":"uploaded", "id":2780}
```

##GET /videos/[video-id]
/videos returns the status of this video - it will contain status "ready", a webm and mp4 url, and at least one thumbnail, when it is done processing.
At this point it can be posted.
(HTTP 200)
```json
{
	"status": "ready",
	"id": 2580,
	"mp4": "https://s3-us-west-1.amazonaws.com/gpcali/048de9a0ea633f53fc010428c09966996066f065c3b3396d782e1d2b1b37d260.mp4",
	"webm": "https://s3-us-west-1.amazonaws.com/gpcali/8a6a1896eb473f1d9138b9a4bbd73969cfda26b928c49702642004c87792f1e3.webm",
	"thumbnails": [
		"https://s3-us-west-1.amazonaws.com/gpcali/234232a6aba24196c3228cc5c8efe191ad959f7783e4ade2a65e2b4e5644b9a0.jpg"
	]
}
```

##POST /profile/profile_image
required parameters: id, token, url

/profile_image expects the url of an image previously uploaded with [/upload](#post-upload).

For now its response is the same as if you issued a GET /user/[id]
but they will diverge in the future.

example responses:
HTTP 200
```json
{
	"id":9,
	"name":"Patrick",
	"tagline":"I like computers",
	"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png",
	"network": { "id":1, "name":"University of Leeds" },
	"course":"Computer Science"
}
```

##POST /profile/name
required parameters: id, token, first, last

/name allows the user to set their name if it is not set already.

On success, it will return HTTP 204.

##POST /profile/tagline
required parameters: id, token, tagline

/tagline allows the user to set their tagline.

On success, it will return HTTP 204.

##POST /profile/change_pass
required parameters: id, token, old, new

old is the user's old password; new is the password the user is changing to.

If it fails it will return 400, on success 204.

##POST /profile/busy
required parameters: id, token, status

status can be true or false

/profile/busy sets user [id] status to [status]

example responses:
HTTP 200
```json
{ "busy":true }
```

##POST /profile/facebook
required parameters: email, pass, fbtoken
where fbtoken is a facebook session token

Alternatively, you may provide the normal gleepost [authentication](#authenticated-endpoints) and fbtoken.

This associates the facebook account logged in with fbtoken with the user signed in with email, pass.

On success, will return 204.

##POST /profile/request_reset
required parameters: email

This will issue a password recovery email, if that email is registered.
A successful response is 204.
Unsuccessful response is 400.

##POST /profile/reset/[user-id]/[reset-token]
required parameters: user-id, reset-token, pass

user-id and reset-token are in the password reset link sent to the users' email address.
pass is the new password.

A successful response (password changed) will be 204.
An unsuccessful response (bad reset token, password too short) will be 400.

##GET /profile/busy
required parameters: id, token

The current busy/free status for this user.

example responses:
HTTP 200
```json
{ "busy":true }
```

##GET /profile/attending
required parameters:
id=[user-id]
token=[token]

This will return an array containing the id of every event this user is attending.
Example response: (http 200)
```json
[1,5,764,34,345]
```

##GET /profile/pending

Displays all your current pending (not yet on the campus wall) posts.
Not sure what this is? See [/approve/pending](#get-approvepending) and related handlers.

HTTP 200:
```json
[
	{
		"id":1976,
		"by":{
			"id":2783,
			"name":"Amy",
			"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/9aabc002cf0b78f2471fa8078335d13471bcb02a672e6da41971fde37135ac70.png"
		},
		"timestamp":"2014-11-06T21:29:02Z",
		"text":"This post should be pending",
		"images":null,
		"comment_count":0,
		"like_count":0,
		"review_history":[
			{
				"action":"rejected",
				"by":{
					"id":2783,
					"name":"Amy",
					"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/9aabc002cf0b78f2471fa8078335d13471bcb02a672e6da41971fde37135ac70.png"
				},
				"at":"2014-11-06T23:36:24Z",
				"reason":"That shit's offensive yo"
			}
		]
	}
]
```

##GET /profile/networks

optional parameters:

`start` - the number of groups this page should be offset by.

This returns a list of up to 20 (non-university) groups this user belongs to.

The list is ordered by last activity: most recent post or message first.


Example response: (http 200)
```json
[
	{
		"id":5345, 
		"name":"Even Cooler Group", 
		"description":"Pretty cool, no?", 
		"url":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg", 
		"creator": {
			"id":2491,
			"name":"Patrick",
			"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg"
		},
		"privacy":"private",
		"role": {
			"name":"member",
			"level":1
		},
		"size":1234,
		"conversation":5678,
		"unread":3,
		"new_posts":4,
		"last_activity":"2014-11-06T23:36:24Z"
	}
]
```

##GET /profile/networks/posts
required parameters:
id=[user-id]
token=[token]

This resource is a combined feed of posts in groups you are a member of.
It functions identically to [/posts](#get-posts) but with one exception:
- Posts also embed information about the group they were posted in.

```json
[
	{
		"id":886,
		"by":
			{
				"id":2491,
				"name":"Patrick",
				"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg"
			},
		"timestamp":"2014-03-04T20:57:39Z",
		"text":"",
		"images":null,
		"videos":[
			{
				"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
				"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
				"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
			}
		],
		"network":
			{
				"id":5345,
				"name":"Super Cool Group",
				"description":"Pretty cool, no?",
				"url":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg",
				 "creator":{
						"id":2491,
						"name":"Patrick",
						"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg"
				},
				"size":1234,
				"conversation":2345
			},
		"comment_count":0,
		"like_count":1
	}
]
```

##DELETE /profile/networks/[network-id]
required parameters:
id=[user-id]
token=[token]

This revokes your membership of the group network-id, if you are a member.
If you attempt this on an official network (a university) you will get an error 403.
Otherwise, you will get 204 No Content.

##GET /notifications
required parameters: id, token

optional parameters: include_seen = (true|false)

before=[id] after=[id] returns a list of 20 posts ordered by time, starting before/after [id]

Returns all unread notifications for user [id]

If include_seen is false, then only the notifications which have not been seen yet will be returned. This is the default behaviour if include_seen is unspecified.

If include_seen is true, the notifications that you have already marked as "seen" will also be visible; they will have the attribute `seen` = `true`.

example responses:
HTTP 200
```json
[
	{
		"id":99999,
		"type":"added_you",
		"time":"2013-09-16T16:58:30.771905595Z",
		"user": {
			"id":9,
			"name":"Patrick",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
	},
	{
		"id":135235,
		"type":"accepted_you",
		"time":"2013-09-16T16:58:30.771905595Z",
		"user": {
			"id":21,
			"name":"Petergatsby",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
	},
	{
		"id":1525345,
		"type":"commented",
		"post":5,
		"time":"2013-09-16T16:58:30.771905595Z",
		"user": {
			"id":2395,
			"name":"testing_user",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		},
		"preview":"Great idea for an event, Peter!"
	},
	{
		"id":3010,
		"type":"commented2",
		"post":5,
		"time":"2013-09-16T16:58:30.771905595Z",
		"user": {
			"id":2395,
			"name":"testing_user",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		},
		"preview":"Great idea for an event, Peter!"
	},
	{
		"id":1525345,
		"type":"added_group",
		"network":1913,
		"time":"2013-09-16T16:58:30.771905595Z",
		"user": {
			"id":2395,
			"name":"testing_user",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
	},
	{
		"id":1525355,
		"type":"group_post",
		"network":1913,
		"post":5,
		"time":"2013-09-16T16:58:30.771905595Z",
		"user": {
			"id":2395,
			"name":"testing_user",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
	},
	{
		"id":12,
		"type":"liked",
		"post":5,
		"time":"2013-09-16T16:58:30.771905595Z",
		"user": {
			"id":2395,
			"name":"testing_user",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
	},
	{
		"id":3006,
		"type":"approved_post",
		"post":12345,
		"time":"2014-11-12T22:51:35Z",
		"user":{
			"id":2783,
			"name":"Amy",
			"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/9aabc002cf0b78f2471fa8078335d13471bcb02a672e6da41971fde37135ac70.png"
		},
		"seen":false
	},
	{
		"id":3007,
		"type":"rejected_post",
		"post":12345,
		"time":"2014-11-12T22:51:35Z",
		"user":{
			"id":2783,
			"name":"Amy",
			"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/9aabc002cf0b78f2471fa8078335d13471bcb02a672e6da41971fde37135ac70.png"
		},
		"seen":false
	},
	{
		"id":3008,
		"type":"group_request",
		"network":12345,
		"time":"2014-11-12T22:51:35Z",
		"user":{
			"id":2783,
			"name":"Amy",
			"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/9aabc002cf0b78f2471fa8078335d13471bcb02a672e6da41971fde37135ac70.png"
		},
		"seen":false
	},
	{
		"id":3007,
		"type":"poll_vote",
		"post":12345,
		"time":"2014-11-12T22:51:35Z",
		"user":{
			"id":2783,
			"name":"Amy",
			"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/9aabc002cf0b78f2471fa8078335d13471bcb02a672e6da41971fde37135ac70.png"
		},
		"seen":false
	},
	{
		"id":3009,
		"type":"attended",
		"post":12345,
		"time":"2014-11-12T22:51:35Z",
		"user":{
			"id":2783,
			"name":"Amy",
			"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/9aabc002cf0b78f2471fa8078335d13471bcb02a672e6da41971fde37135ac70.png"
		},
		"seen":false
	}
] 

```

##PUT /notifications
required parameters: id, token, seen

Marks all notifications for user [id] seen up to and including the notification with id [seen]
Responds with an array containing any unseen notifications.

example responses:
HTTP 200
```json
[
	{
		"id":99999,
		"type":"added_you",
		"time":"2013-09-16T16:58:30.771905595Z",
		"by": {
			"id":9,
			"name":"Patrick",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
	}
] 

```

##POST /verify/[token]

This will verify the account this verification-token is associated with, or create a verified account for a new facebook user.

If it fails it will return HTTP 400 and the error.

Example responses:
HTTP 200
```json
{"verified":true}
```

##POST /resend_verification

required parameters: email
Resend a verification email.

If successful, will respond with HTTP 204.

##POST /contact_form

Required parameters: `name`, `college`, `email`, `phoneNo` 

This records someone reaching out for contact via the form on gleepost.com.

On success, 200.
```json
{"success":true}
```

##GET /search/users/[name]
required parameters: id, token, name

Returns a list of all the users within your primary (ie university) network, who match a search for name.

You can supply partial names (with a minimum length of two characters for the first) and the second name is optional.

If there is a user called "Jonathan Smith", all the searches "Jon" "jonathan" "Jon S" "Jonathan Smi" will match him.

A user may optionally have a `full_name`.

Example response: (HTTP 200)
```json
[
	{
		"id":9, 
		"name":"Steph", 
		"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png",
		"full_name":"Steph Smith"
	},
	{
		"id":23, 
		"name":"Steve", 
		"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png",
		"full_name":"Steve Smith"
	}
]
```

##GET /search/groups/[name]
required parameters: id, token, name

Searches your network for groups matching [name].

Where available, `role` indicates your membership status in this group.

If `pending_request` is present, this indicates you have an outstanding request to join this group.

Example response:
```json
[
	{
		"id":5345, 
		"name":"Even Cooler Group", 
		"description":"Pretty cool, no?", 
		"url":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg", 
		"creator": {
			"id":2491,
			"name":"Patrick",
			"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg"
		},
		"privacy":"private",
		"size":1234,
		"role": {
			"name":"member",
			"level":1
		},
                "pending_request":true
	}
]
```

##POST /reports
required parameters: post
optional parameters: reason

Reports the given post ID to moderators, optionally with a reason.
On success, will give an HTTP 204.

##GET /stats/users/[user-id]/posts/[stat-type]/[period]/[start]/[finish]
required parameters: id, token

- user-id is any user ID you want to see the stats for. At the moment there is no limitation on who can see whose stats.
- stat-type is one of "posts", "likes", "views", "comments", "rsvps", "interactions"
- The special stat type "overview" will give you a combined view containing all the above stat types for this interval.
- period is either "hour", "day" or "week" and indicates how the counts are bucketed (the interval within which counts are summed)
- start and finish are RFC3339 formatted strings which indicate the beginning and end of the period you are viewing stats for.

Example:
GET https://dev.gleepost.com/api/v0.34/stats/user/2395/posts/rsvps/week/2013-01-01T00:00:00Z/2015-01-01T00:00:00Z
```json
{
"start":"2013-01-01T00:00:00Z",
"finish":"2015-01-01T00:00:00Z",
"period":604800,
"data":
	{"rsvps":[
		{"start":"2014-02-11T00:00:00Z","count":1},
		{"start":"2014-02-18T00:00:00Z","count":4},
		{"start":"2014-02-25T00:00:00Z","count":5}
	         ]
	}
}
```

##GET /stats/posts/[post-id]/[stat-type]/[period]/[start]/[finish]
required parameters: id, token

- stat-type is one of "likes", "comments", "views", "rsvps", "interactions"
- The special stat type "overview" will give you a combined view containing all the above stat types for this interval.
- period is either "hour", "day" or "week" and indicates how the counts are bucketed (the interval within which counts are summed)
- start and finish are RFC3339 formatted strings which indicate the beginning and end of the period you are viewing stats for.

Example:
GET https://dev.gleepost.com/api/v1/stats/posts/2395/rsvps/week/2013-01-01T00:00:00Z/2015-01-01T00:00:00Z
```json
{
"start":"2013-01-01T00:00:00Z",
"finish":"2015-01-01T00:00:00Z",
"period":604800,
"data":
	{"rsvps":[
		{"start":"2014-02-11T00:00:00Z","count":1},
		{"start":"2014-02-18T00:00:00Z","count":4},
		{"start":"2014-02-25T00:00:00Z","count":5}
	         ]
	}
}
```

##POST /views/posts

Unlike every other API method, this expects a JSON-encoded post body.
You should submit an array of post:time pairs, like so:

```json
[ 
    {"post":123, "time":"2013-09-05T13:09:38Z"}, 
    {"post":456, "time":"2013-09-05T13:09:38Z"}
]
```

This should respond with a 204.

##GET /approve/access

Indicates whether you are allowed to access (a) Gleepost Approve in general (`access`) and (b) whether you are allowed to change the approval level (`settings`).

```json
{"access":true,"settings":false}
```

##GET /approve/level
`/approve/level` represents the current approval level of the app. A response will look like one of:
```json
{
"level":0,
"categories":[]
}
{
"level":1,
"categories":["party"]
}
{
"level":2,
"categories":["event"]
}
{
"level":3,
"categories":["all"]
}
```

##POST /approve/level

If you are an administrator, you may POST `level` = `0..3` to this endpoint to change the approval level. Responds with the updated approval level in the same format as [GET /approve/level](#get-approve-level), or 403 if you are not allowed.

##GET /approve/pending

Returns all the posts that are currently pending review in your university network, or 403 if you aren't allowed to see them.

These follow exactly the same format as [regular posts](#get-posts) but they are enhanced with an additional `review_history` parameter, which records the events which have happened to this post in the review process.

Most of the time review_history will be empty, but if a post has been rejected and then resubmitted that will be shown here.

```json
[
	{
	"id": 1976,
	"by": {
		"id": 2783,
		"name": "Amy",
		"profile_image": "https://s3-eu-west-1.amazonaws.com/gpimg/9aabc002cf0b78f2471fa8078335d13471bcb02a672e6da41971fde37135ac70.png"
	},
	"timestamp": "2014-11-06T21:29:02Z",
	"text": "This post should be pending",
	"images": null,
	"comment_count": 0,
	"like_count": 0,
	"review_history": [ ]
	}
]
```

##POST /approve/approved

Marks this post as approved. 

Parameters:
`post` : id of post to approve
`reason` : string description of why you approved the post. Optional.

On success, returns 204

##GET /approve/approved

optional parameters:

`start`
`before`
`after`

For pagination, see [/posts](#get-posts)

Displays the history of approved posts. Posts which were approved more recently are displayed at the top.
The `review_history` property contains all the events which happened to this post while it was in revew.

HTTP 200:
```json
[
	{
		"id":1976,
		"by":{
			"id":2783,
			"name":"Amy",
			"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/9aabc002cf0b78f2471fa8078335d13471bcb02a672e6da41971fde37135ac70.png"
		},
		"timestamp":"2014-11-06T21:29:02Z",
		"text":"This post should be pending",
		"images":null,
		"comment_count":0,
		"like_count":0,
		"review_history":[
			{
				"action":"approved",
				"by":{
					"id":2783,
					"name":"Amy",
					"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/9aabc002cf0b78f2471fa8078335d13471bcb02a672e6da41971fde37135ac70.png"
				},
				"at":"2014-11-06T23:36:24Z"
			}
		]
	}
]
```

##POST /approve/rejected

Marks this post as rejected.

Parameters:
`post` : id of post to reject
`reason` : string description of why you rejected the post. Optional.

On success, returns 204.

##GET /approve/rejected

optional parameters:

`start`
`before`
`after`

For pagination, see [/posts](#get-posts)

Displays the history of rejected posts. Most recently rejected first.

The `review_history` property contains all the events which happened to this post while it was in revew.

HTTP 200:
```json
[
	{
		"id":1976,
		"by":{
			"id":2783,
			"name":"Amy",
			"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/9aabc002cf0b78f2471fa8078335d13471bcb02a672e6da41971fde37135ac70.png"
		},
		"timestamp":"2014-11-06T21:29:02Z",
		"text":"This post should be pending",
		"images":null,
		"comment_count":0,
		"like_count":0,
		"review_history":[
			{
				"action":"rejected",
				"by":{
					"id":2783,
					"name":"Amy",
					"profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/9aabc002cf0b78f2471fa8078335d13471bcb02a672e6da41971fde37135ac70.png"
				},
				"at":"2014-11-06T23:36:24Z",
				"reason":"That shit's offensive yo"
			}
		]
	}
]
```

