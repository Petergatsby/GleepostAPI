#Gleepost Websockets

##Maintaining synchronisation

When displaying data to the user which require immediate, reliable delivery - the most obvious example being person-to-person messaging - one rule must be strictly adhered to:

*every time the websocket connection is opened, the current state of the resource MUST be updated to reflect the canonical (server) state.*

Note: This may sometimes result in the client receiving a piece of information almost-simultaneously from two different sources.

For example, if a message is sent concurrently with the client re-establishing its connection they may both see it (a) when they reload the /conversations/:id/messages resource AND (b) as an event over the websocket connection.

These conflicts might come in two categories:
	- A distinct sub-entity, such as a message, has a unique identity and cannot change between copies. Therefore state updates can be idempotent - effectively, you can ignore a second copy
	- An update to a particular parameter, which should only increase (such as a timestamp, or a `read` marker) in which case you can take the latest (largest) value.

###YES: (everything will work OK)
```
No websocket connection <-------------
         |                           ^
         |                           |
         V                           |
Establish connection                 |
         |                           |
         |                           |
         V                           |
Retrieve conversation resource       |
         |                           |
         |                           |
         |                           |
         V                           |
Websocket connection lost ----------->
```

###NO: (messages will be missed)
```
No websocket connection
         |
         |
         V
Retrieve conversation resource
         |
         |
         V
Establish connection
```

###NO: (messages will be missed even this way)
```
No websocket connection
         |
         |
Establish connection
         |
         |
         V
Retrieve conversation resource
         |
         |
         V
Websocket connection lost
         |
         |
         V
Immediately attempt to establish connetcion
         |
         |
         V
Continue as normal
```

##Function of the websocket service

At the moment, this is primarily a receive-only stream of "events". 

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
An event with type "message" is the replacement for a long-poll message. It contains a location (the URI of the conversation it is in) and the data payload is the same message object you find in /conversations/[id]/messages with one variation: it may optionally contain a `group` parameter, if the message belongs to a conversation in a group.

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
Note: if `group` is present in the conversation, this conversation should not be displayed in the inbox.
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

From this point onwards, the client will be updated with information about these posts.

This includes realtime view counts:

```json
{
	"type":"views",
	"location":"/posts/123",
	"data":{
		"views":355
	}
}
```

Realtime poll updates:

```json
{
	"type":"vote",
	"location":"/posts/123",
	"data":{	
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
}
```

And realtime comments:

```json
{
	"type":"comment",
	"location":"/posts/123",
	"data":{
		"id":51341,
		"by": {
			"id":9,
			"name":"Patrick",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		},
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"I concur."
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

##Network events

You may also subscribe to a particular network, in a similar manner to subscribing to posts.

```json
{"action":"SUBSCRIBE", "networks":[123,456,789]}
```

This can be combined with subsribing to posts in the same action.

```json
{
	"action":"SUBSCRIBE",
	"networks":[123,456,789],
	"posts":[123,789]
}
```

Once you've subscribed to a particular network you will get all new posts in that network in realtime:

```json
{
	"type":"post",
	"location":"/networks/123/posts",
	"data":{
		"id":51341,
		"by": {
			"id":9,
			"name":"Patrick",
			"profile_image":"https://gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png"
		},
		"timestamp":"2013-09-05T13:09:38Z",
		"text":"I concur.",
		"categories":[{"id":1, "tag":"some_category", "name":"This is a category"}],
		"attribs": {
			"event-time":"2013-09-05T13:09:38Z",
			"location-desc": "1 Jermyn Street",
			"location-gps": "51.509882,-0.133541",
			"location-name": "McKinsey & Co.",
			"title": "Dead Week Grams!"
		},
		"images": ["https://gleepost.com/uploads/7911970371089d6d59a8a056fe6580a0.jpg", "https://gleepost.com/uploads/3cdcbfbb3646709450d0fb25132ba681.jpg"],
		"videos":[
			{
				"mp4":"https://s3-us-west-1.amazonaws.com/gpcali/038c00d4c7b335f20f793b899a753ba0767324edfec74685fd189d81d76334ec.mp4",
				"webm":"https://s3-us-west-1.amazonaws.com/gpcali/bd4ad39805768915de8a50b8e1cfae8ac518f206d031556de7886612f5e8dd3e.webm",
				"thumbnails":["https://s3-us-west-1.amazonaws.com/gpcali/6e6162b65b83262df79da102bbdbdb824f0cc4149cc51507631eecd53c7635a7.jpg"]
			}
		]
	}
}
```

##Presence

A user can broadcast their presence by sending the following event over their websocket connection:
(valid `form`s are `desktop` or `mobile`

```json
{"action":"presence", "form":"desktop"} 
```

All users who share conversations with this user will receive a `presence` event:

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

##Typing

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
