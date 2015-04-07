##Polling - Draft API spec

###Creating a poll

Same as creating a regular post, except:

 - Use the category name `poll`
 - Specify `poll-expiry` in RFC3339 format `2014-01-31T09:43:28Z`
 - `poll-options` must be a comma-delimited list of length 2-4

###Viewing a poll post

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
 - 
 
###Voting in a poll

 - `POST` to `/posts/:id/votes` with `option` = `0`, `1`, `2`, `3` 

###Updates

 - Subscribing to a post will be unchanged from subscribing to views updates
 - For every vote you will just get the whole poll object again.
 - Or maybe just the votes object: have not decided yet.

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
