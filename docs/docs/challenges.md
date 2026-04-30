---
title: Challenges
nav_order: 4
---

## List Challenges

`GET /api/challenges`

Query parameters:

- `category` (optional, exact category)
- `level` (optional, representative level filter `0`..`10`; `0` means `Unknown`)
- `solved` (optional, `true`/`false`, requires authenticated user context)
- `sort` (optional, one of `latest`, `oldest`, `most_solved`, `least_solved`; default `latest`)
- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "challenges": [
        {
            "id": 1,
            "title": "Warmup",
            "description": "...",
            "category": "Web",
            "created_at": "2026-01-24T12:00:00Z",
            "level": 0,
            "points": 100,
            "solve_count": 12,
            "created_by_user_id": 1,
            "created_by_username": "admin",
            "created_by_affiliation_id": 3,
            "created_by_affiliation": "Blue Team High",
            "is_active": true,
            "is_locked": false,
            "is_solved": true,
            "has_file": true,
            "file_name": "challenge.zip",
            "stack_enabled": false,
            "stack_target_ports": []
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

Notes:

- `points` is fixed and equals the challenge author's configured score.
- If a challenge is locked by progression, it is returned in a reduced form with `is_locked: true`.

Errors:

- 400 `invalid input`

---

## Search Challenges

`GET /api/challenges/search`

Query parameters:

- `q` (required, challenge title keyword)
- `category` (optional, exact category)
- `level` (optional, representative level filter `0`..`10`; `0` means `Unknown`)
- `solved` (optional, `true`/`false`, requires authenticated user context)
- `sort` (optional, one of `latest`, `oldest`, `most_solved`, `least_solved`; default `latest`)
- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "challenges": [
        {
            "id": 1,
            "title": "Warmup",
            "description": "...",
            "category": "Web",
            "created_at": "2026-01-24T12:00:00Z",
            "level": 0,
            "points": 100,
            "solve_count": 12,
            "created_by_user_id": 1,
            "created_by_username": "admin",
            "created_by_affiliation_id": 3,
            "created_by_affiliation": "Blue Team High",
            "is_active": true,
            "is_locked": false,
            "is_solved": true,
            "has_file": true,
            "file_name": "challenge.zip",
            "stack_enabled": false,
            "stack_target_ports": []
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

## Get Challenge Detail

`GET /api/challenges/{id}`

Response 200

```json
{
    "id": 1,
    "title": "Warmup",
    "description": "...",
    "category": "Web",
    "created_at": "2026-01-24T12:00:00Z",
    "level": 0,
    "level_vote_counts": [
        { "level": 6, "count": 2 },
        { "level": 7, "count": 1 }
    ],
    "points": 100,
    "solve_count": 12,
    "created_by_user_id": 1,
    "created_by_username": "admin",
    "created_by_affiliation_id": 3,
    "created_by_affiliation": "Blue Team High",
    "first_blood": {
        "user_id": 7,
        "username": "alice",
        "affiliation": "Blue Team High",
        "bio": "pwn lover",
        "solved_at": "2026-01-24T12:03:00Z",
        "is_first_blood": true
    },
    "is_active": true,
    "is_locked": false,
    "is_solved": true,
    "has_file": true,
    "file_name": "challenge.zip",
    "stack_enabled": false,
    "stack_target_ports": []
}
```

Errors:

- 400 `invalid input`
- 404 `challenge not found`

---

## List Challenge Solvers

`GET /api/challenges/{id}/solvers`

Query parameters:

- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "solvers": [
        {
            "user_id": 7,
            "username": "alice",
            "solved_at": "2026-01-24T12:00:00Z",
            "is_first_blood": true
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
- 404 `challenge not found`

---

## Submit Flag

`POST /api/challenges/{id}/submit`

Headers

```
Cookie: access_token=<jwt>
```

Request

```json
{
    "flag": "flag{...}"
}
```

Response 200

```json
{
    "correct": true
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `user blocked` or `challenge locked`
- 404 `challenge not found`
- 409 `challenge already solved`
- 429 `too many submissions`

---

## Vote Challenge Level

`POST /api/challenges/{id}/vote`

Headers

```
Cookie: access_token=<jwt>
```

Request

```json
{
    "level": 7
}
```

Allowed values: integers `1` through `10`.
`0` (`Unknown`) is system-assigned when no votes exist and cannot be voted directly.

Response 200

```json
{
    "status": "ok"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `user blocked` or `challenge not solved by user`
- 404 `challenge not found`

---

## List Challenge Votes

`GET /api/challenges/{id}/votes`

Query parameters:

- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "votes": [
        {
            "user_id": 7,
            "username": "alice",
            "level": 7,
            "updated_at": "2026-01-24T12:00:00Z"
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
- 404 `challenge not found`

---

## Get My Challenge Vote

`GET /api/challenges/{id}/my-vote`

Headers

```
Cookie: access_token=<jwt>
```

Response 200

```json
{
    "level": 7
}
```

When the caller has not voted on this challenge yet:

```json
{
    "level": null
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 404 `challenge not found`

---

## Download Challenge File

`POST /api/challenges/{id}/file/download`

Headers

```
Cookie: access_token=<jwt>
```

Response 200

```json
{
    "url": "https://s3.example.com/...",
    "expires_at": "2026-01-01T00:00:00Z"
}
```

Errors:

- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `user blocked` or `challenge locked`
- 404 `challenge not found` or `challenge file not found`
- 503 `storage unavailable`

---

## List Challenge Comments

`GET /api/challenges/{id}/challenge-comments`

Query parameters:

- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "comments": [
        {
            "id": 10,
            "content": "Nice challenge.",
            "created_at": "2026-01-24T12:00:00Z",
            "updated_at": "2026-01-24T12:00:00Z",
            "author": {
                "user_id": 7,
                "username": "alice",
                "affiliation_id": 3,
                "affiliation": "Blue Team High",
                "bio": "pwn lover"
            },
            "challenge": {
                "id": 1,
                "title": "Warmup"
            }
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

Notes:

- Comments are returned in latest-first order (`created_at DESC`, then `id DESC`).
- This endpoint is publicly readable (no authentication required).

Errors:

- 400 `invalid input`
- 404 `challenge not found`

---

## Create Challenge Comment

`POST /api/challenges/{id}/challenge-comments`

Headers

```
Cookie: access_token=<jwt>
```

Request

```json
{
    "content": "Nice challenge."
}
```

Validation:

- `content` is required.
- Maximum length is `500` characters.

Response 201

```json
{
    "id": 10,
    "content": "Nice challenge.",
    "created_at": "2026-01-24T12:00:00Z",
    "updated_at": "2026-01-24T12:00:00Z",
    "author": {
        "user_id": 7,
        "username": "alice",
        "affiliation_id": 3,
        "affiliation": "Blue Team High",
        "bio": "pwn lover"
    },
    "challenge": {
        "id": 1,
        "title": "Warmup"
    }
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `user blocked`
- 404 `challenge not found`

---

## Update Challenge Comment

`PATCH /api/challenges/challenge-comments/{id}`

Headers

```
Cookie: access_token=<jwt>
```

Request

```json
{
    "content": "Updated comment."
}
```

Validation:

- `content` is required.
- Maximum length is `500` characters.

Response 200

```json
{
    "id": 10,
    "content": "Updated comment.",
    "created_at": "2026-01-24T12:00:00Z",
    "updated_at": "2026-01-24T12:30:00Z",
    "author": {
        "user_id": 7,
        "username": "alice"
    },
    "challenge": {
        "id": 1,
        "title": "Warmup"
    }
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `user blocked` or `comment access forbidden`
- 404 `comment not found`

---

## Delete Challenge Comment

`DELETE /api/challenges/challenge-comments/{id}`

Headers

```
Cookie: access_token=<jwt>
```

Response 200

```json
{
    "status": "ok"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `user blocked` or `comment access forbidden`
- 404 `comment not found`
