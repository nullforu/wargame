---
title: Auth
nav_order: 2
---

## Register

`POST /api/auth/register`

Request

```json
{
    "email": "user@example.com",
    "username": "user1",
    "password": "strong-password"
}
```

Response 201

```json
{
    "id": 1,
    "email": "user@example.com",
    "username": "user1"
}
```

Errors:

- 400 `invalid input`
- 409 `user already exists`

Validation notes:

- `password` must be at most 72 bytes (bcrypt input limit).

---

## Login

`POST /api/auth/login`

Request

```json
{
    "email": "user@example.com",
    "password": "strong-password"
}
```

Response 200

```json
{
    "user": {
        "id": 1,
        "email": "user@example.com",
        "username": "user1",
        "role": "user",
        "affiliation_id": null,
        "affiliation": null,
        "bio": null,
        "profile_image": null,
        "stack_count": 0,
        "stack_limit": 3,
        "blocked_reason": null,
        "blocked_at": null
    }
}
```

Errors:

- 400 `invalid input`
- 401 `invalid credentials`

Notes:

- `stack_count` and `stack_limit` are calculated per user.
- `access_token` and `refresh_token` are issued as `HttpOnly` cookies.
- `csrf_token` is issued as a readable cookie for double-submit CSRF protection.

---

## Refresh Token

`POST /api/auth/refresh`

Request: send `refresh_token` cookie (and `X-CSRF-Token` header matching `csrf_token` cookie).

Response 200

```json
{
    "status": "ok"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid credentials`

---

## Logout

`POST /api/auth/logout`

Request: send `refresh_token` cookie (and `X-CSRF-Token` header matching `csrf_token` cookie).

Response 200

```json
{
    "status": "ok"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid credentials`
