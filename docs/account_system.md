Account system
==============

The account management is homebrew, except the password hashing which is handled by [bcrypt](https://godoc.org/golang.org/x/crypto/bcrypt) (adaptative hashing algorithm).

Expected client behavior
------------------------

The user can choose to register an account or login to an existing one. Both return a session key (see API below), which must be stored locally (via cookies for instance).

Session keys expire after a certain time spent without being used, or if 5 other sessions have been opened since the last time the key has been used. If a key is expired, a re-login is needed.

The client should then, every time a page is loaded, check the current status of the session by hitting the `/user/status` endpoint (see API below). Actions which require an account to be completed will further require a session key.

API
---

### `/user/register`

Attemps to create a user account. For now, no email is needed (though that could change in the future). Password strength checking is done client-side.

Query:
```JSON
{
    "username": "<username>",
    "password": "<password>",
    "realName": "<Real Name>",
    "captchaResponse": "<reCaptcha response data>"
}
```

Response:
```JSON
{
	"success": boolean,
	"errorMessage": "<error message>",
	"sessionKey": "<base64 session key>",
}
```

### `/user/login`

Request a new session. This should only be done if the current session key is invalid or missing (on a new device or when cookies were cleared for instance).
For security reasons, only 5 simultaneous sessions are allowed; the oldest session will automatically be invalidated if a client goes over the limit.

Query:
```JSON
{
    "username": "<username>",
    "password": "<password>",
    "captchaResponse": "<reCaptcha response data>"
}
```
`captchaResponse` is only needed when the user or client is rate-limited (too many login attempts in the last hour).

Response:
```JSON
{
	"success": boolean,
    "rateLimited" boolean,
	"errorMessage": "<error message>",
	"sessionKey": "<base64 session key>"
}
```
If `rateLimited` is `true`, retry login with CAPTCHA.

### `/user/info`

Request information about the (presumably) logged in user. This should be used every time a page is loaded.

Query:
```JSON
{
    "sessionKey": "<session key>"
}
```

Response:
```JSON
{
    "loggedIn": boolean,
    "username": "<username>",
    "realName": "<Real Name>"
}
```
If the user is not logged in (the sessionKey is invalid or has been disabled, see "Expected Client Behavior"), userName and realName will be blank.

### `/user/update/password`

Change the logged in user's password. Password strength checking is done client-side.

Query:
```JSON
{
    "sessionKey": "<session key>",
    "password": "<password>"
}
```

Response:
```JSON
{
	"success": boolean,
	"errorMessage": "<error message>"
}
```

### `/user/update/realName`

Change the logged in user's real name. Empty field is allowed.

Query:
```JSON
{
    "sessionKey": "<session key>",
    "realName": "<Real name>"
}
```

Response:
```JSON
{
	"success": boolean,
	"errorMessage": "<error message>"
}
```

SQL Structure
-------------

```SQL
create table users (
    id integer not null primary key,
    username text not null,
    passwordHash text not null,
    realName text not null,
    disabled integer not null
);

create table userSessions (
    sessionKey text not null primary key,
    userId integer not null,
    loginTime integer not null,
    lastSeenTime integer not null
);

create table userLogins (
    id integer not null primary key,
    userId integer not null,
    remoteIp text not null,
    time integer not null,
    success integer not null
);
```

* `users` contains the user account information;
* `userSessions` contains all active session (*however, an active session is not necessarily a valid session; if it is invalid/too old/whatever, a login check must ask for re-authentication and delete the session*);
* `userLogins` contains a log of all user logins for rate-limiting purposes. Can also be used for administrative purposes.