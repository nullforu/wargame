---
title: Challenges
nav_order: 4
---

## List Challenges

`GET /api/challenges`

Query parameters:

- `q` (optional, title keyword)
- `category` (optional, exact category)
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
            "level": 0,
            "points": 100,
            "solve_count": 12,
            "created_by_user_id": 1,
            "created_by_username": "admin",
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
            "level": 0,
            "points": 100,
            "solve_count": 12,
            "created_by_user_id": 1,
            "created_by_username": "admin",
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
    "level": 0,
    "level_vote_counts": [
        { "level": 6, "count": 2 },
        { "level": 7, "count": 1 }
    ],
    "points": 100,
    "solve_count": 12,
    "created_by_user_id": 1,
    "created_by_username": "admin",
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
Authorization: Bearer <access_token>
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
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `user blocked` or `challenge locked`
- 404 `challenge not found`
- 409 `challenge already solved`
- 429 `too many submissions`

---

## Vote Challenge Level

`POST /api/challenges/{id}/vote`

Headers

```
Authorization: Bearer <access_token>
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
- 401 `invalid token` or `missing authorization` or `invalid authorization`
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
Authorization: Bearer <access_token>
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
- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 404 `challenge not found`

---

## Download Challenge File

`POST /api/challenges/{id}/file/download`

Headers

```
Authorization: Bearer <access_token>
```

Response 200

```json
{
    "url": "https://s3.example.com/...",
    "expires_at": "2026-01-01T00:00:00Z"
}
```

Errors:

- 401 `invalid token` or `missing authorization` or `invalid authorization`
- 403 `user blocked` or `challenge locked`
- 404 `challenge not found` or `challenge file not found`
- 503 `storage unavailable`
