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

`poll-expiry` was in the future, but too soon:
```json
{"error":"Poll ending too soon"}
```

`poll-expiry` too far in the future:
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

###Messages

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

###Networks

###Approve
