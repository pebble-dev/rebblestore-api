Account system
==============

The account management is homebrew, except the password hashing which is handled by [bcrypt](https://godoc.org/golang.org/x/crypto/bcrypt) (adaptative hashing algorithm).

API
---

### `/user/register`

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
	"session_key": "<base64 session key>"
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
    time integer not null,
    success integer not null
);
```

* `users` contains the user account information;
* `userSessions` contains all active session (*however, an active session is not necessarily a valid session; if it is invalid/too old/whatever, a login check must ask for re-authentication and delete the session*);
* `userLogins` contains a log of all user logins for rate-limiting purposes. Can also be used for administrative purposes.