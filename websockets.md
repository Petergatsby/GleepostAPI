#Gleepost Websockets

The websocket system is currently in beta, expect many new events in the coming weeks.

At the moment, this is a receive-only stream of "events". 

An event looks like this:

```
{
	"type":"message",
	"location":"/conversations/67",
	"data":{"id":1173,"by":{"id":9,"username":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/59bdb3c4a4151cc7ab41137eecbcc4d461291f72cfd6b6516b12de00a7ad1a94.jpg"},"text":"testing12345678901234","timestamp":"2013-12-12T15:20:54.665361234Z","seen":false}
}
```

It consists of a ["type"](#event-types), an optional "location" (A URI for the resource which has changed, where appropriate), and a data payload.


##Event types
An event type will be one of: [message](#message) [new-conversation](#new-conversation) [notification](#notification)

###Message
An event with type "message" is the replacement for a long-poll message. It contains a location (the URI of the conversation it is in) and the data payload is the same message object you find in /conversations/[id]/messages.

```
{
	"type":"message",
	"location":"/conversations/67",
	"data":{"id":1173,"by":{"id":9,"username":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/59bdb3c4a4151cc7ab41137eecbcc4d461291f72cfd6b6516b12de00a7ad1a94.jpg"},"text":"testing12345678901234","timestamp":"2013-12-12T15:20:54.665361234Z","seen":false}
}
```

###New conversation
An event with type "new-conversation" is triggered every time you are placed in a new conversation. It contains a location (the URI of the conversation) and the data payload is the conversation object.
```
{
	"type":"new-conversation",
	"location":"/conversations/1595",
	"data":{"id":1595,"lastActivity":"2013-12-16T14:13:27.609454716Z","participants":[{"id":2147,"username":"PaulLoran","profile_image":""},{"id":9,"username":"Patrick","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/59bdb3c4a4151cc7ab41137eecbcc4d461291f72cfd6b6516b12de00a7ad1a94.jpg"}],"expiry":{"time":"2013-12-16T14:23:27.609455414Z","ended":false}}
}
```

###Notification
An event with type "notification" is triggered every time you recieve a new notification. Its location is simply "/notifications" (see note). It contains a notification object.
```
 {
	"type":"notification",
	"location":"/notifications",
	"data":{"id":596,"type":"added_you","time":"2013-12-16T14:33:40.260990792Z","user":{"id":2395,"username":"TestingUser","profile_image":"https://s3-eu-west-1.amazonaws.com/gpimg/5c780da1230506100f037abf88d74d88cb0556510c49af40c95ee02e0a35ad57.png"}}
}
```

Note: The location might be changed in future to the location that the notification "happened" (particular post, user, etc).
