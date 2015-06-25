      ________.__                                       __   
     /  _____/|  |   ____   ____ ______   ____  _______/  |_ 
    /   \  ___|  | _/ __ \_/ __ \\____ \ /  _ \/  ___/\   __\
    \    \_\  \  |_\  ___/\  ___/|  |_> >  <_> )___ \  |  |  
     \______  /____/\___  >\___  >   __/ \____/____  > |__|  
            \/          \/     \/|__|              \/        

#Gleepost API / v1

#Contents

##General themes of API design

##Registration, account creation and authentication

##Core data types

###Users

###Posts

####Poll posts

 - The post will have `poll` in its categories list
 - The post will have a `poll` parameter, which will look like so:

```json
"poll": {
	"options":["option1", "option B", "another option", "joe biden"],
	"votes":{
		"option1":1234,
		"option B": 3,
		"another option": 67,
		"joe biden": 456
	},
	"expires-at":"2014-01-31T09:43:28Z",
	"your-vote":"option B"
}
```

#####Creating a poll

Same as creating a regular post, except:

 - The post must be in the category `poll`
 - `poll-expiry` is required.
  - `poll-expiry` indicates when this poll will end, and is a RFC3339 formatted string, eg `2015-04-15T01:05:03Z` OR a Unix timestamp.
 - `poll-options` is a comma-delimited list of the options available in this poll. 
  - You must specify at least 2 and at most 4 options, and the options must each be 3 <= n <= 50 characters long.
eg: `hillary clinton,alien kang, alien kodos,abstain`

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

Two or more of your options were identical:
```json
{"error":"All options must be distinct"}
```

#####Voting in a poll

`POST` to `/posts/:id/votes` with `option` = `0`, `1`, `2`, `3`

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

###Comments

###Conversations

###Presence
Within a conversation, each user has a parameter `presence`, indicating their last activity within the app, and the form factor they were active on (`desktop` or `mobile`).

Note: `desktop` presence supersedes `mobile`; if a user is connected with both a desktop and a mobile device, other users will see `desktop`.

```json
{
	"form":"mobile",
	"at":"2015-06-02T15:58:00Z"
}
```

Clients should set their presence; while the app is active, the client should submit their presence every 30 seconds by sending a message on their websocket connection:
(again, `form` must be `desktop` or `mobile`)

```json
{"action":"presence","form":"mobile"}
```

Clients with an active websocket connection will receive presence updates regularly from all users who participate in any of their conversations.

```json
{
	"type":"presence",
	"location":"/user/123",
	"data":{
		"user":123,
		"form":"mobile",
		"at":"2015-06-02T11:24:01Z"
	}
}
```

In addition, participants in a conversation will have their presence indicated within their user object:

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
	"read":[{"user":9,"last_read":1000}],
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

###Typing

A client should indicate to other members of a conversation that they are typing, by sending a typing action over their websocket connection:

```json
{"action":"typing", "conversation":123, "typing":true} 
```

The other participants in this conversation will get a `typing` event:

```json
{
	"type":"typing",
	"location":"/conversation/123",
	"data":{
		"user":456,
		"conversation":123,
		"typing":true
	}
}
```

If the client deletes all their input, then they can manually cancel the typing status:

```json
{"action":"typing", "conversation":123, "typing":false} 
```

Otherwise, clients should timeout the typing status after a few seconds, or upon receiving a message from that user.

###Messages

####Formatting

To display a message on the client, follow the following rules:

(1) Replace special embed sequences with their enhanced version; escape those which do not constitute a valid embed.
`foo &lt;bar&gt; <@123|@foo> <sup> :+1:`
becomes

`foo &lt;bar&gt <a href="/user/123">@foo</a> &lt;sup&gt; :+1:`

(2) Replace emoji sequences with unicode or images where appropriate

`foo &lt;bar&gt <a href="/user/123">@foo</a> &lt;sup&gt; <img class="emoji" src="/images/thumbsup.png" />`

####Mentions

A user may _mention_ another user, tagging them with `@name`.

On sending, clients should transform this into the format `<@123|@name>`, where `123` corresponds to that user's ID.

A client may also mention `<@all|@all>`, which matches every participant in the conversation.

Mentioned participants will get a push notification, even if they have muted the conversation.


###Notifications

 - Someone voted in your poll

```json
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
	}
```

 - Someone requested access to a group you administrate

```json
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
	}
```

 - Someone RSVP'd to your event

```json
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
```

 - Someone commented on a post you've commented on:

nb: if the conditions for this and `commented` are met, you will get a `commented` notification not a `commented2` notification.

```json
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
	}
```

###Networks

A Network is a collection of users and posts; each university is a Network.

```json
{
	"id":1,
	"name":"University of Leeds"
}
```

Users may also create networks within their university; these are Groups. These have some additional parameters:

```json
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
	"conversation":5678,
	"last_activity":"2014-11-06T23:36:24Z"
}
```

####Joining a network

Only Groups may be joined. To do so, there are two options:

If a group's `privacy` is `public`, you are allowed to add yourself to the network directly; just [POST to /networks/:id/users](#post-networksnetwork-idusers) with your own ID. 

If a group's `privacy` is `private`, you may request access to the group by sending a [POST to /networks/:id/requests](#post-networksnetwork-idrequests).

###Approve
