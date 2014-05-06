      ________.__                                       __   
     /  _____/|  |   ____   ____ ______   ____  _______/  |_ 
    /   \  ___|  | _/ __ \_/ __ \\____ \ /  _ \/  ___/\   __\
    \    \_\  \  |_\  ___/\  ___/|  |_> >  <_> )___ \  |  |  
     \______  /____/\___  >\___  >   __/ \____/____  > |__|  
            \/          \/     \/|__|              \/        


#Gleepost API / V0.34


Production URL: https://gleepost.com/api/v1/
Development URL: https://dev.gleepost.com/api/v0.34/

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

###Authenticated endpoints:
These endpoints require authentication to access.
You must send an <id, token> pair with a request, which you can generate with /login or /fblogin

This may be sent in a query string "?id=1234&token=foobar" (where "1234" and "foobar" are id and token respectively), as parameters in the request body, or in the header "X-GP-Auth" with the format "1234-foobar"

/posts [[GET]](#get-posts) [[POST]](#post-posts) 

/posts/[post-id]/comments [[GET]](#get-postspost-idcomments) [[POST]](#post-postspost-idcomments)

/posts/[post-id] [[GET]](#get-postspost-id) [[DELETE]](#delete-postspost-id)

/posts/[post-id]/images [[POST]](#post-postspost-idimages)

/posts/[post-id]/likes [[POST]](#post-postspost-idlikes)

/posts/[post-id]/attending [[POST]](#post-postspost-idattending) [[DELETE]](#delete-postspost-idattending)

/networks [[POST]](#post-networks)

/networks/[network-id] [[GET]](#get-networksnetwork-id) [[PUT]](#put-networksnetwork-id)

/networks/[network-id]/posts [[GET]](#get-networksnetwork-idposts) [[POST]](#post-networksnetwork-idposts)

/networks/[network-id]/users [[GET]](#get-networksnetwork-idusers) [[POST]](#post-networksnetwork-idusers)

/live [[GET]](#get-live)

/conversations [[GET]](#get-conversations) [[POST]](#post-conversations)

/conversations/live [[GET]](#get-conversationslive)

/conversations/read_all [[POST]](#post-conversationsread_all)

/conversations/[conversation-id] [[GET]](#get-conversationsconversation-id) [[DELETE]](#delete-conversationsconversation-id) [[PUT]](#get-conversationsconversation-id)

/conversations/[coversation-id]/messages [[GET]](#get-conversationsconversation-idmessages) [[POST]](#post-conversationsconversation-idmessages) [[PUT]] (#put-conversationsconversation-idmessages)

/user [[POST]](#post-user)

/user/[user-id] [[GET]](#get-useruser-id)

/user/[user-id]/posts [[GET]](#get-useruser-idposts)

/longpoll [[GET]](#get-longpoll)

/ws [[GET]](#get-ws)

/contacts [[GET]](#get-contacts) [[POST]](#post-contacts)

/contacts/[contact-id] [[PUT]](#put-contactsuser)

/devices [[POST]](#post-devices) 

/devices/[device-id] [[DELETE]](#delete-devicesdevice-id)

/upload [[POST]](#post-upload)

/profile/profile_image [[POST]](#post-profileprofile_image)

/profile/name [[POST]](#post-profilename)

/profile/change_pass [[POST]](#post-profilechange_pass)

/profile/busy [[POST]](#post-profilebusy) [[GET]](#get-profilebusy)

/profile/facebook [[POST]](#post-profilefacebook)

/profile/attending [[GET]](#get-profileattending)

/profile/networks [[GET]](#get-profilenetworks)

/profile/networks/posts [[GET]](#get-profilenetworksposts)

/profile/networks/[network-id] [[DELETE]](#delete-profilenetworksnetwork-id)

/notifications [[GET]](#get-notifications) [[PUT]](#put-notifications)

/search/users/[name] [[GET]](#get-searchusersname)

###Statistics endpoints

####Stat endpoints are currently in development. This means they may change in any way at any time for any reason.

/stats/users/[user-id]/posts/[stat-type]/[period]/[start]/[finish] [[GET]](#get-statsusersuser-idpostsstat-typeperiodstartfinish)

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
{"status":"unverified"}
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
{"status":"unverified"}
```

If the email you have provided is already registered, the response will be:
(HTTP 200)
```json
{"status":"registered"}
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
		}
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"This is a cool post for cool people!",
		"categories":[{"id":1, "tag":"some_category", "name":"This is a category"}],
		"attribs": {
			"event-time":"2013-09-05T13:09:38Z"
		},
		"comment_count":4,
		"like_count":5,
		"popularity":75,
		"attendee_count":3,
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
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"]
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
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg"]
	}
]
```

##POST /posts
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

example responses:
(http 200)
```json
{"id":3}
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
		"event-time":"2013-09-05T13:09:38Z"
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
	"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"]
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

example responses:
(http 200)
```json
[
	{
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

##POST /posts/[post-id]/images
required parameters: id, token, url

This adds an image previously uploaded with [/upload](#post-upload) to this post.

example responses:
(http 201)
```json
["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"]
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

##POST /posts/[post-id]/attending
required parameters: id, token

Issuing a POST to this URI should mark you as attending this event, and acts idempotently.
It will return a 204 if successful.

##DELETE /posts/[post-id]/attending
required parameters: id, token

Issuing a DELETE to this URI should mark you as not attending this event.
It should succeed even if you aren't already attending.
It will return a 204 if successful.

##GET /live
required parameters: id, token, after

[after] must be either an RFC3339 formatted time string, or a unix timestamp.

Live returns the 20 events whose event-time is soonest after "after".

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
	"attribs":{"event-time":"2014-02-05T12:47:59Z"},
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
	"attribs":{"event-time":"2014-02-05T15:34:39Z"},
	"popularity":100,
	"attendee_count":5,
	"comment_count":0,
	"like_count":1,
	"likes":[{"by":{"id":2395,"name":"TestingUser","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/260a8e71eb2dbfed25b0a0de5ae328cdfc931c5023668955ba660e61705c6800.jpg"},
	"timestamp":"2014-02-05T07:00:54Z"}]
	}
]
```

##GET /networks/[network-id]
required parameters:
id=[user-id]
token=[token]

A group resource, or 403 if you aren't a member of the group.
example responses (http 200):

```json
{"id":5345, "name":"Super Cool Group", "description":"Pretty cool, no?", "url":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg", "creator":{"id":2491,"name":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg"}}
```

##PUT /networks/[network-id]
required parameters:
id=[user-id]
token=[token]

url="URL returned from /upload"

If you created this group, you can change the group's image. If you didn't create the group -- or you didn't choose a valid image URL - it will return 403. Otherwise, returns the updated resource.

```json
{"id":5345, "name":"Super Cool Group", "description":"Pretty cool, no?", "url":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg", "creator":{"id":2491,"name":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg"}}
```

##POST /networks
required parameters:
id=[user-id]
token=[token]
name="Name of the group"

optional:
desc="Description of the group"
url = uploaded image URL
This creates a new group named `name` and adds you as a member.

If url is not valid, it will respond with a 403.

A successful response is 201:

```json
{"id":5345, "name":"Even Cooler Group", "description":"Pretty cool, no?", "url":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg", "creator":{"id":2491,"name":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg"}}
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
			"event-time":"2013-09-05T13:09:38Z"
		},
		"comment_count":4,
		"like_count":5,
		"attendee_count":324,
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
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"]
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
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg"]
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

A collection of all the users in this network, or 403 if you aren't a member of the network (or if it is a university network)
Example response:
```json
[{"id":9, "name":"Patrick", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},{"id":23, "name":"PeterGatsby", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}]
```

##POST /networks/[network-id/users
required parameters:
id=[user-id]
token=[token]

One of:
users=[other-user-id],[other-user-id],[other-user-id]

fbusers=[facebook-id],[facebook-id],[facebook-id]

email=[other-user-email]

Adds other users to this network, or records that they have been invited via facebook, or emails them an invite if they aren't on Gleepost. On success will return 204.

##GET /conversations/live
required parameters:
id=[user-id]
token=[token]
Returns up to three live conversations (whose "ended" attribute is false) for the current user.

```json
[
	{"id":1,
	"participants": [
		{"id":9, "name":"Patrick", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":23, "name":"PeterGatsby", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}
	],
	"read":[{"user":9,"last_read":1000}],
	"lastActivity":"2013-09-05T13:09:38Z",
	"mostRecentMessage": {"id":1234214, "by":{"id":9, "name":"Patrick"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
	"expiry": { "time": "2013-11-13T22:11:32.956855553Z", "ended":false }
	}
]
```

##POST /conversations/read_all
required parameters:
id=[user-id]
token=[token]

Marks all conversations as "seen".
On success, will return a 204 (no content).

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
		{"id":9, "name":"Patrick", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":23, "name":"PeterGatsby", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}
	],
	"read":[{"user":9,"last_read":1000}],
	"lastActivity":"2013-09-05T13:09:38Z",
	"mostRecentMessage": {"id":1234214, "by":{"id":9, "name":"Patrick"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
	"expiry": { "time": "2013-11-13T22:11:32.956855553Z", "ended":false }
	},
	{"id":2,
	"participants" [
		{"id":99999, "name":"Lukas", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":232515, "name":"Ling", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}
	],
	"lastActivity":"2013-09-05T13:09:38Z",
	"mostRecentMessage": {"id":123512624, "by":99999, "text":"idk lol", "timestamp":"2013-09-05T13:09:38Z"}
	}
]
```

##POST /conversations
required parameters:
id=[user-id]
token=[token]

optional parameters:
random=[true/false], defaults to true

If random = true, you should provide:
participant_count=[2 <= n <= 4], defaults to 2

if random = false, you should provide:
participants=[user_id],[user_id],[user_id],...
(a comma-delimited list of user_ids to start a conversation with.

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
	"lastActivity":"2013-09-05T13:09:38Z",
	"expiry": { "time": "2013-11-13T22:11:32.956855553Z", "ended":false }
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
	"read":[{"user":9,"last_read":1000}],
	"messages": [
		{"id":1234214, "by":{"id":23, "name":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
		{"id":1234214, "by":{"id":23, "name":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
		{"id":1234214, "by":{"id":23, "name":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"}
	],
	"lastActivity":"2013-09-05T13:09:38Z",
	"expiry": { "time": "2013-11-13T22:11:32.956855553Z", "ended":false }
}
```

##DELETE /conversations/[conversation-id]
required parameters:
id=[user-id]
token=[token]

This ends a live conversation. If you try this on a regular conversation, I don't know what will happen!

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
		{"id":1234214, "by":9, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
		{"id":1234214, "by":9, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"},
		{"id":1234214, "by":9, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z"}
]
```

##PUT /conversations/[conversation-id]
required parameters:
id=[user-id]
token=[token]
expiry=[bool]

Set expiry = false and a conversation's expiry will be deleted.
Will return the updated conversation object.
NB: This probably isn't the right place to put this. Will change in a future release.


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

example responses:
```json
{
	"id":9,
	"name":"Patrick",
	"tagline":"I like computers",
	"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png",
	"network": { "id":1, "name":"University of Leeds" },
	"course":"Computer Science",
	"full_name":"Patrick Molgaard"
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
			"event-time":"2013-09-05T13:09:38Z"
		},
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
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"]
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
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg"]
	}
]

```

##POST /newconversation

DEPRECATED, use [[/conversations]](#post-conversations)

##POST /newgroupconversation

DEPRECATED, use [[/conversations]](#post-conversations)

##GET /longpoll
required parameters:
id=[user-id]
token=[token]

Longpoll will block until a message arrives for the current user (in any conversation).
If no message arrives within 60s the response will be empty-object "{}".

example responses:
```json
{
	"id":53,
	"by": {"id":9,"name":"Patrick", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
	"text":"sup",
	"timestamp":"2013-09-16T16:58:30.771905595Z",
	"conversation_id":5
}
```

##GET /ws
Required parameters:
id=[user-id]
token=[token]

See [the websockets readme.](websockets.md)

##GET /contacts
required parameters:
id=[user-id]
token=[token]

Gets all the current user's contacts.

If you've added someone, they_confirmed will be false until they accept you and vice versa.

example responses:

HTTP 200
```json
[
	{
		"id":1234,
		"name":"calgould",
		"you_confirmed":true,
		"they_confirmed":false,
	},
	{
		"id":21,
		"name":"petergatsby",
		"you_confirmed":false,
		"they_confirmed":true,
	}
]
```

##POST /contacts
required parameters: id, token, user

Adds the user with id [user] to the current contact list.
If this user has already added you, it will accept them.

example responses:

HTTP 201
```json
{
	"id":1234,
	"name":"calgould",
	"you_confirmed":true,
	"they_confirmed":false,
}
```

##PUT /contacts/[user]
required parameters: id, token, accepted

if accepted = true, it will set that contact to "confirmed"

example responses:
HTTP 200
```json
{
	"id":21,
	"name":"petergatsby",
	"you_confirmed":true,
	"they_confirmed":true,
}
```

##POST /devices
required parameters: id, token, type, device_id

Type should be "android" or "ios"

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
required parameters: id, token, image

/upload expects a single multipart/form-data encoded image and on success will return a url.

example responses:
HTTP 201
```json
{"url":"https://s3-eu-west-1.amazonaws.com/gpimg/3acd82c15dd0e698fc59c79e445a464553e57d338a6440601551c7fb28e45bf9.jpg"}
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

##GET /profile/networks
required parameters:
id=[user-id]
token=[token]

This returns a list of all (non-university) groups this user belongs to.

Example response: (http 200)
```json
[{"id":5345, "name":"Stanford Catan Club"}]
```

##GET /profile/networks/posts
required parameters:
id=[user-id]
token=[token]

This resource is a combined feed of posts in groups you are a member of.
It functions identically to [/posts](#get-posts) but with one exception:
- Posts also embed information about the group they were posted in.

```json
[{"id":886,"by":{"id":2491,"name":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg"},"timestamp":"2014-03-04T20:57:39Z","text":"","images":null,"network":{"id":5345, "name":"Super Cool Group", "description":"Pretty cool, no?", "url":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg", "creator":{"id":2491,"name":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/45661eff6323f17ee42d90fe2fa0ad8dcf29d28a67619f8a95babf4ace48ff96.jpg"}},
"comment_count":0,"like_count":1}]
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

Returns all unread notifications for user [id]

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
		}
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
		"id":12
		"type":"liked",
		"post":5,
		"time":"2013-09-16T16:58:30.771905595Z",
		"user": {
			"id":2395,
			"name":"testing_user",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
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

##GET /search/users/[name]
required parameters: id, token, name

Returns a list of all the users within your primary (ie university) network, who match a search for name.

You can supply partial names (with a minimum length of two characters for the first) and the second name is optional.

If there is a user called "Jonathan Smith", all the searches "Jon" "jonathan" "Jon S" "Jonathan Smi" will match him.

Example response: (HTTP 200)
```json
[{"id":9, "name":"Steph", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},{"id":23, "name":"Steve", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}]
```

##GET /stats/users/[user-id]/posts/[stat-type]/[period]/[start]/[finish]
required parameters: id, token

- user-id is any user ID you want to see the stats for. At the moment there is no limitation on who can see whose stats.
- stat-type is one of "posts", "likes", "comments", "rsvps", "interactions"
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
