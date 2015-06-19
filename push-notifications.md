#Push notifications

##You've been added to a group 
(iOS)

```json
{
    "aps":{
         "alert":{
             "loc-key":"GROUP",
             "loc-args":["Patrick", "Super Secret Group"],
         },
         "badge":12345,
         "sound":"default"
     },
     "group-id":6789
}
```

(Android)
```json
{
    "registration_ids":["APA91bF58RwLEXNBMoKxy5s1sxmxQXL8MYgGmdgAyWw5YFzNyrH876WWL20Il7j8vxCqw6Ube8puw5JkRvIaIDws94iRInE7jfHqXq-EZ34RtdHeil7cuCp-xIYMDbsE3b50W1eTlRNdHRAG0SODHfbg1yORcJ9Beg"],
    "collapse_key":"You've been added to a group",
    "data": {
         "for":8,
         "group-name":"Super Cool Group",
         "group-id":6789,
         "adder":"Patrick",
         "type":"GROUP"
    }
}
```

##Someone added you to their contacts (deprecated)

(iOS)

```json
{
    "aps":{
         "alert":{
             "loc-key":"added_you",
             "loc-args":["Patrick"],
         },
         "badge":12345,
         "sound":"default"
     },
     "adder-id":6789
}
```

(Android)

```json
{
    "registration_ids":["APA91bF58RwLEXNBMoKxy5s1sxmxQXL8MYgGmdgAyWw5YFzNyrH876WWL20Il7j8vxCqw6Ube8puw5JkRvIaIDws94iRInE7jfHqXq-EZ34RtdHeil7cuCp-xIYMDbsE3b50W1eTlRNdHRAG0SODHfbg1yORcJ9Beg"],
    "collapse_key":"Someone added you to their contacts.",
    "data": {
         "for":8,
         "adder-id":6789,
         "adder":"Patrick",
         "type":"added_you"
    }
}
```

##Someone accepted your contact request (deprecated)

(iOS)

```json
{
    "aps":{
         "alert":{
             "loc-key":"accepted_you",
             "loc-args":["Patrick"],
         },
         "badge":12345,
         "sound":"default"
     },
     "accepter-id":6789
}
```

(android)

```json
{
    "registration_ids":["APA91bF58RwLEXNBMoKxy5s1sxmxQXL8MYgGmdgAyWw5YFzNyrH876WWL20Il7j8vxCqw6Ube8puw5JkRvIaIDws94iRInE7jfHqXq-EZ34RtdHeil7cuCp-xIYMDbsE3b50W1eTlRNdHRAG0SODHfbg1yORcJ9Beg"],
    "collapse_key":"Someone accepted your contact request.",
    "data": {
         "for":8,
         "accepter-id":6789,
         "accepter":"Patrick",
         "type":"accepted_you"
    }
}
```

##You have a new conversation (deprecated)

(iOS)
```json
{
    "aps":{
         "alert":{
             "loc-key":"NEW_CONV",
             "loc-args":["Patrick"],
         },
         "badge":12345,
         "sound":"default"
     },
     "conv":6789
}
```

(android)


```json
{
    "registration_ids":["APA91bF58RwLEXNBMoKxy5s1sxmxQXL8MYgGmdgAyWw5YFzNyrH876WWL20Il7j8vxCqw6Ube8puw5JkRvIaIDws94iRInE7jfHqXq-EZ34RtdHeil7cuCp-xIYMDbsE3b50W1eTlRNdHRAG0SODHfbg1yORcJ9Beg"],
    "collapse_key":"You have a new conversation!",
    "data": {
         "for":8,
         "with-id":6789,
         "with":"Patrick",
         "type":"NEW_CONV"
    }
}
```

##Someone liked your post
(iOS)

```json
{
    "aps":{
         "alert":{
             "loc-key":"liked",
             "loc-args":["Patrick"],
         },
         "badge":12345,
         "sound":"default"
     },
     "liker-id":6789,
     "post-id":1234
}
```

(android)
```json
{
    "registration_ids":["APA91bF58RwLEXNBMoKxy5s1sxmxQXL8MYgGmdgAyWw5YFzNyrH876WWL20Il7j8vxCqw6Ube8puw5JkRvIaIDws94iRInE7jfHqXq-EZ34RtdHeil7cuCp-xIYMDbsE3b50W1eTlRNdHRAG0SODHfbg1yORcJ9Beg"],
    "collapse_key":"Someone liked your post.",
    "data": {
         "for":8,
         "liker-id":6789,
         "liker":"Patrick",
         "type":"liked",
         "post-id":12345
    }
}
```

##Someone commented on your post

(iOS)

```json
{
    "aps":{
         "alert":{
             "loc-key":"commented",
             "loc-args":["Patrick"],
         },
         "badge":12345,
         "sound":"default"
     },
     "commenter-id":6789,
     "post-id":1234
}
```

(Android)

```json
{
    "registration_ids":["APA91bF58RwLEXNBMoKxy5s1sxmxQXL8MYgGmdgAyWw5YFzNyrH876WWL20Il7j8vxCqw6Ube8puw5JkRvIaIDws94iRInE7jfHqXq-EZ34RtdHeil7cuCp-xIYMDbsE3b50W1eTlRNdHRAG0SODHfbg1yORcJ9Beg"],
    "collapse_key":"Someone commented on your post.",
    "data": {
         "for":8,
         "commenter-id":6789,
         "commenter":"Patrick",
         "type":"commented",
         "post-id":12345
    }
}
```

##An admin posted in a group you're in

(iOS)

```json
{
    "aps":{
         "alert":{
             "loc-key":"group_post",
             "loc-args":["Patrick", "Super Secret Group"],
         },
         "badge":12345,
         "sound":"default"
     },
     "group-id":6789
}

```

(android)

```json
{
    "registration_ids":["APA91bF58RwLEXNBMoKxy5s1sxmxQXL8MYgGmdgAyWw5YFzNyrH876WWL20Il7j8vxCqw6Ube8puw5JkRvIaIDws94iRInE7jfHqXq-EZ34RtdHeil7cuCp-xIYMDbsE3b50W1eTlRNdHRAG0SODHfbg1yORcJ9Beg"],
    "collapse_key":"Somoene posted in your group.",
    "data": {
         "for":8,
         "group-name":"Super Cool Group",
         "group-id":6789,
         "poster":"Patrick",
         "type":"group_post"
    }
}
```


##Version update notification


```json
{
	"aps":{
		"alert":"[arbitrary message]",
		"badge":12345,
		"sound":"default"
	},
	"version":"1.2.3"
}
```

Alert contains an arbitrary update message.
Version is the new app version number.

##Someone approved your post

(iOS)

```json
{
    "aps":{
         "alert":{
             "loc-key":"approved_post",
             "loc-args":["Patrick"],
         },
         "badge":12345,
         "sound":"default"
     },
     "approver-id":6789,
     "post-id":12345
}

```

##Someone rejected your post

(iOS)

```json
{
    "aps":{
         "alert":{
             "loc-key":"rejected_post",
             "loc-args":["Patrick"],
         },
         "badge":12345,
         "sound":"default"
     },
     "rejecter-id":6789,
     "post-id":12345
}
```

##Someone voted in your poll

(iOS)

```json
{
    "aps":{
         "alert":{
             "loc-key":"poll_vote",
             "loc-args":["Patrick"],
         },
         "badge":12345,
         "sound":"default"
     },
     "voter-id":6789,
     "post-id":12345

}
```

(Android)

```json
{
    "registration_ids":["APA91bF58RwLEXNBMoKxy5s1sxmxQXL8MYgGmdgAyWw5YFzNyrH876WWL20Il7j8vxCqw6Ube8puw5JkRvIaIDws94iRInE7jfHqXq-EZ34RtdHeil7cuCp-xIYMDbsE3b50W1eTlRNdHRAG0SODHfbg1yORcJ9Beg"],
    "collapse_key":"Someone voted in your poll.",
    "data": {
         "for":8,
         "Voter":"Patrick",
         "post-id":6789,
         "voter-id":1234,
         "type":"poll_vote"
    }
}

```

##Someone requested to join a group you administrate

(iOS)

```json
{
    "aps":{
         "alert":{
             "loc-key":"group_request",
             "loc-args":["Patrick", "Super Private Group"],
         },
         "badge":12345,
         "sound":"default"
     },
     "group-id":6789
}
```

(Android)

```json
{
    "registration_ids":["APA91bF58RwLEXNBMoKxy5s1sxmxQXL8MYgGmdgAyWw5YFzNyrH876WWL20Il7j8vxCqw6Ube8puw5JkRvIaIDws94iRInE7jfHqXq-EZ34RtdHeil7cuCp-xIYMDbsE3b50W1eTlRNdHRAG0SODHfbg1yORcJ9Beg"],
    "collapse_key":"Someone requested to join your group.",
    "data": {
         "for":8,
         "group-name":"Super Private Group",
         "group-id":6789,
         "requester":"Patrick",
         "type":"group_request"
    }
}
```

##Someone mentioned you in a conversation

(iOS)

```
{
	"aps":{
		"alert":{
			"loc-key":"mentioned",
			"loc-args":["Patrick", "Hey, what's up @silouanos?"]
		},
		"badge":12345,
		"sound":"default"
	},
	"conv":123,
	"group":345,
	"profile_image":"https://example.com/foo.jpg"
}
```
