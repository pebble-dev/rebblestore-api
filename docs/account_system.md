Account system
==============

The account management is homebrew, except the password hashing which is handled by [bcrypt](https://godoc.org/golang.org/x/crypto/bcrypt) (adaptative hashing algorithm).

API
---

### `/user/register`

Attemps to create a user account. For now, no email is needed (though that could change in the future).

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