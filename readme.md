#Gleepost API / V0.17


URL: https://gleepost.com/api/v0.17/

##Notes:

* Only available over HTTPS

* Parameters can be form-encoded in a POST body, or sent as a query string

##Available API endpoints:

/register [[POST]](#post-register)

/login [[POST]](#post-login)

/posts [[GET]](#get-posts) [[POST]](#post-posts)

/posts/[post-id]/comments [[GET]](#get-postspost-idcomments) [[POST]](#post-postspost-idcomments)

/posts/[post-id]/images [[POST]](#post-postspost-idimages)

/posts/[post-id]/likes [[POST]](#post-postspost-idlikes)

/conversations [[GET]](#get-conversations)

/conversations/[conversation-id] [[GET]](#get-conversationsconversation-id)

/conversations/[coversation-id]/messages [[GET]](#get-conversationsconversation-idmessages) [[POST]](#post-conversationsconversation-idmessages) [[PUT]] (#put-conversationsconversation-idmessages)

/user/[user-id] [[GET]](#get-useruser-id)

/newconversation [[POST]](#post-newconversation)

/newgroupconversation [[POST]](#post-newgroupconversation)

/longpoll [[GET]](#get-longpoll)

/contacts [[GET]](#get-contacts) [[POST]](#post-contacts)

/contacts/[contact-id] [[PUT]](#put-contactsuser)

/device [[POST]](#post-device)

/upload [[POST]](#post-upload)

/profile/profile_image [[POST]](#post-profileprofile_image)

/profile/busy [[POST]](#post-profilebusy)

/notifications [[GET]](#post-notifications) [[PUT]](#put-notifications)

##POST /register
required parameters: user, pass, email

example responses:
(HTTP 200)
```
{"id":143423424}
```
(HTTP 400)
```
{"error":"Invalid email"}
```

##POST /login
required parameters: user, pass

example responses:
(HTTP 200) 
```
{"id":9, "value":"f0e4c2f76c58916ec258f246851bea091d14d4247a2fc3e18694461b1816e13b", "expiry":"2013-09-05T14:53:34.226231725Z"}
```
(HTTP 400)
```
{"error":"Bad username/password"}
```

##GET /posts
required parameters:
id=[user-id]
token=[token]

optional parameters:
start=[count]
returns a list of 20 posts ordered by time, starting at count

example responses:
(HTTP 200)
```
[
	{
		"id":2,
		"by": {
			"id":9,
			"username":"Patrick",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"This is a cool post for cool people!",
		"comments":4,
		"likes":5,
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"]
	},
	{
		"id":1,
		"by": {
			"id":23,
			"username":"PeterGatsby"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"Sailor Moon FTW!"
		"comments":9,
		"likes":0,
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg"]
	}
]
```

##POST /posts
required parameters: id, token, text

example responses:
(http 200)
```
{"id":3}
```

##GET /posts/[post-id]/comments

required parameters: 

id=[user-id]
token=[token]

optional parameters:
start=[count]

example responses:
(http 200)
```
[
	{
		"id":51341,
		"by": {
			"id":9,
			"username":"Patrick"
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		},
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"I concur."
	},
	{
		"id":4362346,
		"by": {
			"id":545,
			"username":"SomeoneElse"
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
```
{"id":234}
```

##POST /posts/[post-id]/images
required parameters: id, token, url

This adds an image previously uploaded with [/upload](#post-upload) to this post.

example responses:
(http 201)
```
["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"]
```

##POST /posts/[post-id]/likes
required parameters: id, token, liked

[liked] must be a boolean.
If true, adds a like for this post for this user.
If false, removes a like for this post for this user.

example responses:
(http 200)
```
{"post":5, "liked":true}
```
```
{"post":5, "liked":false}
```

##GET /conversations
required parameters:
id=[user-id]
token=[token]

optional parameters:
start=[count]

returns a list of 20 of your conversations ordered by most recent message, starting at count
```
[
	{"id":1,
	"participants": [
		{"id":9, "username":"Patrick", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":23, "username":"PeterGatsby", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}
	],
	"mostRecentMessage": {"id":1234214, "by":{"id":9, "username":"Patrick"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z", "seen":false},
	},
	{"id":2,
	"participants" [
		{"id":99999, "username":"Lukas", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":232515, "username":"Ling", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}
	],
	"mostRecentMessage": {"id":123512624, "by":99999, "text":"idk lol", "timestamp":"2013-09-05T13:09:38Z", "seen":false}
	}
]
```

##GET /conversations/[conversation-id]
required parameters:
id=[user-id]
token=[token]

example responses:
(HTTP 200)
```
{
	"id":1,
	"participants": [
		{"id":9, "username":"Patrick", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":23, "username":"PeterGatsby", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}
	],
	"messages": [
		{"id":1234214, "by":{"id":23, "username":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z", "seen":false},
		{"id":1234214, "by":{"id":23, "username":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z", "seen":false},
		{"id":1234214, "by":{"id":23, "username":"PeterGatsby"}, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z", "seen":false}
	]
}
```

##GET /conversations/[conversation-id]/messages
required parameters: id=[user-id], token=[token]
optional parameters: start=[start], after=[after], before=[before]

Returns a list of 20 messages ordered by time from most recent to least recent.
Given [start], it returns messages from the [start]th most recent to [start + 20]th most recent.
Given [after], it returns at most 20 of the messages received since [after]
Given [before], it returns at most 20 of the messages received immediately before [before]

example responses:
```
[
		{"id":1234214, "by":9, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z", "seen":false},
		{"id":1234214, "by":9, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z", "seen":false},
		{"id":1234214, "by":9, "text":"asl? ;)", "timestamp":"2013-09-05T13:09:38Z", "seen":false}
]
```

##POST /conversations/[conversation-id]/messages
required parameters: id, token, text

example responses:
```
{"id":1356}
```

##PUT /conversations/[conversation-id]/messages
required parameters: id, token, seen

Marks all messages in a conversation up to [seen] 
(that were not sent by the current user) seen.

example responses:

seen=51
(HTTP 200)
```
{
	"id": 5,
	"participants": [
		{
			"id": 9,
			"username": "Patrick",
			"profile_image": "https://gleepost.com/uploads/avatar.png"
		},
		{
			"id": 1327,
			"username": "Meg",
			"profile_image": "",
		}
	],
	"messages": [
		{
			"id": 52,
			"by": {
				"id": 9,
				"username": "Patrick",
				"profile_image": "https://gleepost.com/uploads/bad2cbd1431260c2c4b9766ae5de25d6.gif",
			},
			"text": "sup",
			"timestamp": "2013-09-16T16:58:23Z",
			"seen": false
		},
		{
			"id": 51,
			"by": {
				"id": 9,
				"username": "Patrick",
				"profile_image": "https://gleepost.com/uploads/bad2cbd1431260c2c4b9766ae5de25d6.gif",
			},
			"text": "sup",
			"timestamp": "2013-09-16T16:58:30Z",
			"seen": true
		}
	]
}


```

##GET /user/[user-id]
required parameters:
id=[user-id]
token=[token]

example responses:
```
{
	"id":9,
	"username":"Patrick",
	"tagline":"I like computers",
	"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png",
	"network": { "id":1, "name":"University of Leeds" },
	"course":"Computer Science"
}
```

##POST /newconversation
required parameters: id, token

note: POST so it doesn't get accidentally repeated :)
This will return a conversation with two participants.

example responses:
```
{
	"id":2342342,
	"participants": [
		{"id":9, "username":"Patrick", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":23, "username":"PeterGatsby", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}
	]
}
```

##POST /newgroupconversation
required parameters: id, token

note: POST so it doesn't get accidentally repeated :)
This will return a conversation with more than two participants.

example responses:
```
{
	"id":2342342,
	"participants": [
		{"id":9, "username":"Patrick", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":23, "username":"PeterGatsby", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":1351, "username":"Someone", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
		{"id":6124, "username":"SomeoneElse", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"}
	]
}
```

##GET /longpoll
required parameters:
id=[user-id]
token=[token]

Longpoll will block until a message arrives for the current user (in any conversation).
If no message arrives within 60s the connection will timeout.

example responses:
```
{
	"id":53,
	"by": {"id":9,"username":"Patrick", "profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"},
	"text":"sup",
	"timestamp":"2013-09-16T16:58:30.771905595Z",
	"seen":false,
	"conversation_id":5
}
```

##GET /contacts
required parameters:
id=[user-id]
token=[token]

Gets all the current user's contacts.

If you've added someone, they_confirmed will be false until they accept you and vice versa.

example responses:

HTTP 200
```
[
	{
		"id":1234,
		"username":"calgould",
		"you_confirmed":true,
		"they_confirmed":false,
	},
	{
		"id":21,
		"username":"petergatsby",
		"you_confirmed":false,
		"they_confirmed":true,
	}
]
```

##POST /contacts
required parameters: id, token, user

Adds the user with id [user] to the current contact list.

example responses:

HTTP 201
```
{
	"id":1234,
	"username":"calgould",
	"you_confirmed":true,
	"they_confirmed":false,
}
```

##PUT /contacts/[user]
required parameters: id, token, accepted

if accepted = true, it will set that contact to "confirmed"

example responses:
HTTP 200
```
{
	"id":21,
	"username":"petergatsby",
	"you_confirmed":true,
	"they_confirmed":true,
}
```
##POST /device
required parameters: id, token, type, device_id

Type should be "android" or "ios"

This registers the push notification id "device_id" for the current user

example responses: 
HTTP 201
```
{
	"user":2395,
	"type":"android",
	"id":"APA91bFmOKOcm6v1ZJVavmvHQ3SLzADznBHhT6gDdNUDZm9wSc-yBdToyAWtR73cro5rnemVTiXdqQMlqmrs_4mdAhZbiLIfeZ4cD4L9OstvTnjzv8-Yx_fSPM1Joe_gpAEe0haNEwh3pSQah1QQQFC829jA7V-vswpuQLmLT2sK_ciMo5Hx7po"
}
```

##POST /upload
required parameters: id, token, image

/upload expects a single multipart/form-data encoded image and on success will return a url.

example responses:
HTTP 201
```
{"url":"https://s3-eu-west-1.amazonaws.com/gpimg/3acd82c15dd0e698fc59c79e445a464553e57d338a6440601551c7fb28e45bf9.jpg"}
```

##POST /profile/profile_image
required parameters: id, token, url

/profile_image expects the url of an image previously uploaded with [/upload](#post-upload).

For now its response is the same as if you issued a GET /user/[id]
but they will diverge in the future.

example responses:
HTTP 200
```
{
	"id":9,
	"username":"Patrick",
	"tagline":"I like computers",
	"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png",
	"network": { "id":1, "name":"University of Leeds" },
	"course":"Computer Science"
}
```

##POST /profile/busy
required parameters: id, token, status

status can be true or false

/profile/busy sets user [id] status to [status]

example responses:
HTTP 200
```
{ "busy":true }
```

##GET /notifications
required parameters: id, token

Returns all unread notifications for user [id]

example responses:
HTTP 200
```
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
	},
	{
		"id":135235,
		"type":"accepted_you",
		"time":"2013-09-16T16:58:30.771905595Z",
		"by": {
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
		"by": {
			"id":2395,
			"name":"testing_user,
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		}
	},
	{
		"id":12
		"type":"liked",
		"post":5,
		"time":"2013-09-16T16:58:30.771905595Z",
		"by": {
			"id":2395,
			"name":"testing_user,
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
```
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
