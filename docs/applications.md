Applications
==============

Applications refer to both apps and watchfaces (as they are nearly identical from a technical POV).

Expected client behavior
------------------------

No authentication is needed to fetch applications using the endpoints described in this file.

The times are all full length [ISO 8601](https://en.wikipedia.org/wiki/ISO%208601) with timezone.

The platforms are `aplite` (Classic, Steel), `basalt` (Time, Time Steel), `chalk` (Time Round), `diorite` (Pebble 2). Note, however, that apps crawled from the Pebble App Store may have additional platform support (`emery` from the Pebble Time 2 and `ios` or `android`). You shouldn't care about these.

API
---

### `/dev/apps/get_app/id/{id}`

Fetches detailed information about the app `{id}`.

Query: `GET` request. No parameters needed.

Response:
```JSON
{
	"id": "<Application ID>",
	"title": "<Application title>",
	"author": {
		"id": "<Author id>",
		"name": "<Author name>"
	},
	"description": "<Plaintext app description>",
	"thumbs_up": integer,
	"type": "watchapp" | "watchface",
	"supported_platforms": [
		"<platform 1>",
		"<platform 2>",
		"<...>"
	],
	"published_date": "<Time the app was published>",
	"appInfo": {
		"pbwUrl": "<Url of the .pbw file>",
		"rebbleReady": boolean, // Set to false if the app is a mirror pulled from the original pebble app store, or true if it is linked to a real account.
		"tags": [
			{
				"id": "<Tag ID>",
				"name": "<Tag human readable name>",
				"color": "<Tag color>"
			}
		],
		"updated": "<Last update time>",
		"version": "<App version>",
		"supportUrl": "<App support URL (can be an email)>",
		"authorUrl": "<App author website URL>",
		"sourceUrl": "<Source code URL>"
	},
	"assets": {
		"appBanner": "<App banner image URL>",
		"appIcon": "<App icon image URL>",
		"screenshots": [
			{
				"platform": "<platform1>",
				"screenshots": [
					"<screenshot 1>",
                    "<screenshot 2>",
                    "<...>"
				]
			},
			{
				"platform": "<platform2>",
				"screenshots": [
					"<screenshot 1>",
                    "<screenshot 2>",
                    "<...>"
				]
			},
			{
				"platform": "<...>",
				"screenshots": [
					"<screenshot 1>",
                    "<screenshot 2>",
                    "<...>"
				]
			}
		]
	},
	"doomsday_backup": boolean // Set to true if the app developper allow his or her app to be added to a downloadable version of the app store in case the Rebble servers shut down.
}
```

### `/dev/apps/get_tags/id/{id}`

Fetches the tag list for the app `{id}`.

Query: `GET` request. No parameters needed.

Response:
```json
{
	"tags": [
		{
			"id": "<Tag ID>",
			"name": "<Tag human readable name>",
			"color": "<Tag color>"
		},
		{
			"id": "<Tag ID>",
			"name": "<Tag human readable name>",
			"color": "<Tag color>"
		},
        {
            ...
        }
	]
}
```

### `/dev/apps/get_versions/id/{id}`

Fetches the version list for the app `{id}`.

Query: `GET` request. No parameters needed.

Response:
```json
{
	"versions": [
		{
			"number": "<version number>",
			"release_date": "<version release time>",
			"description": "<version changelog>"
		},
		{
			"number": "<version number>",
			"release_date": "<version release time>",
			"description": "<version changelog>"
		},
		{
            ...
		},
	]
}
```

### `/dev/apps/get_collection/id/{id}?order={order}&platform={platform}&page={page}`

Fetches a cards list from the collection `{id}`.

Query: `GET` request. All parameters are optional (defaults to first page with all platforms and ordered by most recent).

Parameters:
* `order`: `new` or `popular`;
* `platform`: One of the pebble platforms.

Response:
```json
{
	"id": "<Collection ID>",
	"name": "<Human readable collection name>",
	"pages": integer, // The number of pages is capped for performance reasons
	"cards": [
		{
			"id": "<App id>",
			"title": "<App title>",
			"type": "watchapp"|"watchface",
			"image_url": "<URL of the cover image>",
			"thumbs_up": integer
		},
		{
			"id": "<App id>",
			"title": "<App title>",
			"type": "watchapp"|"watchface",
			"image_url": "<URL of the cover image>",
			"thumbs_up": integer
		},
		{
            ...
		},
	]
}
```

### `/dev/apps/search/{query}`

Fetches a cards list matching the `query` string.

Query: `GET` request.

Response:
```json
{
	"cards": [
		{
			"id": "<App id>",
			"title": "<App title>",
			"type": "watchapp"|"watchface",
			"image_url": "<URL of the cover image>",
			"thumbs_up": integer
		},
		{
			"id": "<App id>",
			"title": "<App title>",
			"type": "watchapp"|"watchface",
			"image_url": "<URL of the cover image>",
			"thumbs_up": integer
		},
        {
            ...
        }
	]
}
```

### `/dev/author/id/{id}`

Fetches information about author `id`.

Query: `GET` request.

Response:
```json
{
	"id": integer,
	"name": "<Author name>",
	"cards": [ // List of the author's apps
		{
			"id": "<App id>",
			"title": "<App title>",
			"type": "watchapp"|"watchface",
			"image_url": "<URL of the cover image>",
			"thumbs_up": integer
		},
		{
			"id": "<App id>",
			"title": "<App title>",
			"type": "watchapp"|"watchface",
			"image_url": "<URL of the cover image>",
			"thumbs_up": integer
		},
        {
            ...
        }
	]
}
```

SQL Structure
-------------

```SQL
create table apps (
    id text not null primary key,
    name text,
    author_id integer,
    tag_ids blob,
    description text,
    thumbs_up integer,
    type text,
    supported_platforms blob,
    published_date integer,
    pbw_url text,
    rebble_ready integer,
    updated integer,
    version text,
    support_url text,
    author_url text,
    source_url text,
    screenshots blob,
    banner_url text,
    icon_url text,
    doomsday_backup integer,
    versions blob
);

create table authors (
    id text not null primary key,
    name text
);

create table collections (
    id text not null primary key,
    name text,
    color text,
    apps blob
);
```

* `apps` contains information about specific apps;
* `authors` contains information about specific authors;
* `collections` contains a list of all applications belonging to said applications, because generating such a list on the fly from blob'd data from the `tag_ids` field in `apps` would be ridiculously expensive.