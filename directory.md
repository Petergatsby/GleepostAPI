#Stanford directory search

You can search the stanford student directory with this endpoint:

`/directory/stanford/[query]`

##Default behaviour
The default behaviour of this endpoint is to proxy a search to the stanford directory website:

https://stanfordwho.stanford.edu/SWApp/

Because the query is just searching this website, it will be pretty slow (probably >1s response time). It is, therefore, not suitable for progressive/partial searches.

You can search by name, email and stanford student ID (Maybe more, but I have not tested the others)

Results in this mode are limited to ~100 matches: 100 per "kind" of match, with the three kinds being "email matches", "firstname matches" and "lastname matches".

There is no way to retrieve additional pages.

A query for the name "john" might look like this:

```json
[
  {
    "name": "John Bunnell",
    "id": "DS484V847",
    "affiliations": [
      {
        "name": "University - Emeritus staff",
        "department": "Retiree",
        "position": "Staff Emeritus Retiree"
      }
    ]
  },
  {
    "name": "John Chambliss",
    "id": "DR436M117",
    "affiliations": [
      {
        "name": "University - Staff",
        "department": "Plumbing Shop",
        "position": "Plumber Specialist"
      }
    ]
  },
  {
    "name": "John Baugh",
    "id": "DS162D607",
    "affiliations": [
      {
        "name": "University - Emeritus faculty",
        "department": "Graduate School of Education",
        "position": "Emeritus Faculty, Acad Council"
      }
    ]
  }
]
```

If your query is specific enough to return only one result, by e.g. querying an exact ID or email address, you will get more detail in the result.

For example, the query "DS484V847":

```json
[
  {
    "name": "John Bunnell",
    "id": "DS484V847",
    "email": "John.Bunnell@stanford.edu",
    "affiliations": [
      {
        "name": "University - Emeritus staff",
        "department": "Retiree",
        "position": "Staff Emeritus Retiree",
        "phones": [
          "(650) 725-2840"
        ]
      }
    ]
  }
]
```

With a single result, you are guaranteed to have an `email` field.

##Cache mode
In addition, you may also use the directory in "cache" mode by appending `?cache=true`

This will query the locally API-cached results, and supports partial (character-by-character) searching on the person name. It is fast enough to support "instant" search.

In exchange, however, the results may be incomplete or slightly inaccurate. 

Cache mode also will not yet return the additional detail for a specific query.


##Implementing gleepost/stanford search

Given the constraints of these systems, when searching the entire university a client should adopt the following behaviour:

1. As the user types, search both the app-users (`/search/users/{query}`) and the cached directory (`/directory/stanford/{query}?cache=true`)

2. Once the user stops typing for a client-configured timeout length, do an additional stanford proxy search (`/directory/stanford/{query}`)

3. If the user selects a user from the stanford directory, perform a second exact search using that user's Stanford ID ( eg. `/directory/stanford/DS484V847`) to retrieve their additional details (ie, their email)
