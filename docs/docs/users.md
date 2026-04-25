---
title: Users
nav_order: 3
---

## Me

`GET /api/me`

Headers

```
Authorization: Bearer <access_token>
```

Response 200

```json
{
    "id": 1,
    "email": "user@example.com",
    "username": "user1",
    "role": "user",
    "affiliation_id": 2,
    "affiliation": "Blue Team",
    "bio": "Blue Team player",
    "stack_count": 0,
    "stack_limit": 3,
    "blocked_reason": null,
    "blocked_at": null
}
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`

---

## Update Me

`PUT /api/me`

Headers

```
Authorization: Bearer <access_token>
```

Request

```json
{
    "username": "new_username",
    "affiliation_id": 2,
    "bio": "Blue Team player"
}
```

Response 200

```json
{
    "id": 1,
    "email": "user@example.com",
    "username": "new_username",
    "role": "user",
    "affiliation_id": 2,
    "affiliation": "Blue Team",
    "bio": "Blue Team player",
    "stack_count": 0,
    "stack_limit": 3,
    "blocked_reason": null,
    "blocked_at": null
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `user blocked`

---

## List Users

`GET /api/users`

Query parameters:

- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "users": [
        {
            "id": 1,
            "username": "user1",
            "role": "user",
            "affiliation_id": 2,
            "affiliation": "Blue Team",
            "bio": "Blue Team player",
            "blocked_reason": null,
            "blocked_at": null
        },
        {
            "id": 2,
            "username": "admin",
            "role": "admin",
            "affiliation_id": null,
            "affiliation": null,
            "bio": null,
            "blocked_reason": null,
            "blocked_at": null
        }
    ],
    "pagination": {
        "page": 1,
        "page_size": 20,
        "total_count": 2,
        "total_pages": 1,
        "has_prev": false,
        "has_next": false
    }
}
```

Errors:

- 400 `invalid input`

---

## Search Users

`GET /api/users/search`

Query parameters:

- `q` (required, username keyword)
- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "users": [
        {
            "id": 1,
            "username": "user1",
            "role": "user",
            "affiliation_id": 2,
            "affiliation": "Blue Team",
            "bio": "Blue Team player",
            "blocked_reason": null,
            "blocked_at": null
        }
    ],
    "pagination": {
        "page": 1,
        "page_size": 20,
        "total_count": 1,
        "total_pages": 1,
        "has_prev": false,
        "has_next": false
    }
}
```

Errors:

- 400 `invalid input`

---

## Get User

`GET /api/users/{id}`

Response 200

```json
{
    "id": 1,
    "username": "user1",
    "role": "user",
    "affiliation_id": 2,
    "affiliation": "Blue Team",
    "bio": "Blue Team player",
    "blocked_reason": null,
    "blocked_at": null
}
```

Errors:

- 400 `invalid input`
- 404 `not found`

---

## Get User Solved Challenges

`GET /api/users/{id}/solved`

Query parameters:

- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "solved": [
        {
            "challenge_id": 1,
            "title": "Warmup",
            "points": 100,
            "solved_at": "2026-01-24T12:00:00Z"
        }
    ],
    "pagination": {
        "page": 1,
        "page_size": 20,
        "total_count": 1,
        "total_pages": 1,
        "has_prev": false,
        "has_next": false
    }
}
```

Errors:

- 400 `invalid input`
- 404 `not found`
