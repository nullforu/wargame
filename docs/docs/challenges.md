---
title: Challenges
nav_order: 4
---

## List Challenges

`GET /api/challenges`

Query parameters:

- `q` (optional, title keyword)
- `category` (optional, exact category)
- `level` (optional, integer `1..10`)
- `solved` (optional, `true`/`false`, requires authenticated user context)
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
            "level": 3,
            "points": 100,
            "solve_count": 12,
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
- `level` (optional, integer `1..10`)
- `solved` (optional, `true`/`false`, requires authenticated user context)
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
            "level": 3,
            "points": 100,
            "solve_count": 12,
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
    "level": 3,
    "points": 100,
    "solve_count": 12,
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
