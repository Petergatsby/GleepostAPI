#Gleepost Websockets

The websocket system is currently in beta, expect many new events in the coming weeks.

At the moment, this is a receive-only stream of "events". 

An event looks like this:

```json
{
	"type":"message",
	"location":"/conversations/67",
	"data":{"id":1173,"by":{"id":9,"username":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/59bdb3c4a4151cc7ab41137eecbcc4d461291f72cfd6b6516b12de00a7ad1a94.jpg"},"text":"testing12345678901234","timestamp":"2013-12-12T15:20:54.665361234Z","seen":false}
}
```

It consists of a ["type"](#event-types), an optional "location" (A URI for the resource which has changed, where appropriate), and a data payload.


##Event types
An event type will be one of: [message](#message) [read](#read) [new-conversation](#new-conversation) [ended-conversation](#ended-conversation) [changed-conversation](#changed-conversation) [notification](#notification) [video-ready](#video-ready)

###Message
An event with type "message" is the replacement for a long-poll message. It contains a location (the URI of the conversation it is in) and the data payload is the same message object you find in /conversations/[id]/messages.

```json
{
	"type":"message",
	"location":"/conversations/67",
	"data":{"id":1173,"by":{"id":9,"username":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/59bdb3c4a4151cc7ab41137eecbcc4d461291f72cfd6b6516b12de00a7ad1a94.jpg"},"text":"testing12345678901234","timestamp":"2013-12-12T15:20:54.665361234Z","seen":false}
}
```

##Read
An event with type "read" is triggered every time someone marks a message as seen. It contains the URI of the relevant conversation, and a userID:messageID pair to indicate what the most recent read message was.
```json
{
	"type":"read",
	"location":"/conversations/67",
	"data":{"user":1173, "last_read":1234}
}

```

###New conversation
An event with type "new-conversation" is triggered every time you are placed in a new conversation. It contains a location (the URI of the conversation) and the data payload is the conversation object.
```json
{
	"type":"new-conversation",
	"location":"/conversations/1595",
	"data":{"id":1595,"lastActivity":"2013-12-16T14:13:27.609454716Z","participants":[{"id":2147,"username":"PaulLoran","profile_image":""},{"id":9,"username":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/59bdb3c4a4151cc7ab41137eecbcc4d461291f72cfd6b6516b12de00a7ad1a94.jpg"}],"expiry":{"time":"2013-12-16T14:23:27.609455414Z","ended":false}}
}
```

###Ended conversation
An event with type "ended-conversation" is triggered every time a conversation you participate in is terminated. It contains a location (the URI of the conversation) and the data payload is the conversation object.
```json
{
	"type":"ended-conversation",
	"location":"/conversations/1595",
	"data":{"id":1595,"lastActivity":"2013-12-16T14:13:27.609454716Z","participants":[{"id":2147,"username":"PaulLoran","profile_image":""},{"id":9,"username":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/59bdb3c4a4151cc7ab41137eecbcc4d461291f72cfd6b6516b12de00a7ad1a94.jpg"}],"expiry":{"time":"2013-12-16T14:23:27.609455414Z","ended":true}}
}
```

###Changed conversation
An event with type "changed-conversation" is triggered when a conversation is converted from "live" to "regular".
It contains the URI of the affected conversation and a representation of the new conversation.
```json
{
	"type":"changed-conversation",
	"location":"/conversations/1595",
	"data":{"id":1595,"lastActivity":"2013-12-16T14:13:27.609454716Z","participants":[{"id":2147,"username":"PaulLoran","profile_image":""},{"id":9,"username":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/59bdb3c4a4151cc7ab41137eecbcc4d461291f72cfd6b6516b12de00a7ad1a94.jpg"}]}}
}
```

###Notification
An event with type "notification" is triggered every time you recieve a new notification. Its location is simply "/notifications" (see note). It contains a notification object.
```json
{
	"type":"notification",
	"location":"/notifications",
	"data":{"id":596,"type":"added_you","time":"2013-12-16T14:33:40.260990792Z","user":{"id":2395,"username":"TestingUser","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/5c780da1230506100f037abf88d74d88cb0556510c49af40c95ee02e0a35ad57.png"}}
}
```

##Video ready
An event with type "video-ready" is triggered once a video you have uploaded has finished processing. 
```json
{
	"type":"video-ready",
	"location":"/videos/2586",
	"data":{
		"status": "ready",
		"id": 2586,
		"mp4": "https://s3-us-west-1.amazonaws.com/gpcali/a28269e2de0cb2b5ca9a36a55e9b7ccaf1ae46e4cedc5054ba9667b31c4ccb9b.mp4",
		"webm": "https://s3-us-west-1.amazonaws.com/gpcali/213107680550e4964c2d25c5999d9709d1d94c138b35d394c60b851ef69b0dc0.webm",
		"thumbnails": [
			"https://s3-us-west-1.amazonaws.com/gpcali/377f566caa4da4806a66795ce9241eee54f1b3be7c4ff5b32b6b526f08fdd449.jpg"
		]
	}
}
```

Note: The location might be changed in future to the location that the notification "happened" (particular post, user, etc).

#Post events
In addition to the core events every user who subscribes to a websocket will get, a client may optionally indicate interest in specific posts.

To do so, the client must send a subscription message with the following format:

```json
{"action":"SUBSCRIBE", "posts":[123,456,789]}
```
Where `123`, `456`, `789` represent post IDs that the client is interested in.

From this point onwards, the client will be updated with realtime view counts for these posts:

```json
{
	"type":"views",
	"location":"/posts/123",
	"data":{
		"views":355
	}
}
```

Once a client is no longer interested in updates to a particular post (for example, when it has scrolled out of view) it can stop receiving more events:

```json
{"action":"UNSUBSCRIBE", "posts":[123]}
```

If you unsubscribe from an empty list of posts, this is treated as unsubscribing from everything, including notifications / messages; the websocket will then close itself.

```json
{"action":"UNSUBSCRIBE", "posts":[]}
```
