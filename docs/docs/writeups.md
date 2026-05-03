---
title: Writeups
nav_order: 10
---

Notes:

- For authenticated `POST`, `PUT`, `PATCH`, and `DELETE` requests, send both `csrf_token` cookie and matching `X-CSRF-Token` header.

## List Challenge Writeups

`GET /api/challenges/{id}/writeups`

Query parameters:

- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "writeups": [
        {
            "id": 11,
            "created_at": "2026-04-26T10:00:00Z",
            "updated_at": "2026-04-26T10:30:00Z",
            "author": {
                "user_id": 7,
                "username": "alice",
                "affiliation_id": 2,
                "affiliation": "Semyeong High",
                "bio": "web / pwn",
                "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg"
            },
            "challenge": {
                "id": 1,
                "title": "Web Warmup",
                "category": "Web",
                "points": 100,
                "level": 4
            },
            "is_mine": false
        }
    ],
    "can_view_content": false,
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

- `content` is included only when `can_view_content` is `true`.
- Metadata fields are always returned.

Errors:

- 400 `invalid input`
- 404 `challenge not found`

---

## Get Writeup Detail

`GET /api/writeups/{id}`

Response 200

```json
{
    "writeup": {
        "id": 11,
        "content": "# Step 1\n...",
        "created_at": "2026-04-26T10:00:00Z",
        "updated_at": "2026-04-26T10:30:00Z",
        "author": {
            "user_id": 7,
            "username": "alice",
            "affiliation_id": 2,
            "affiliation": "Semyeong High",
            "bio": "web / pwn",
            "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg"
        },
        "challenge": {
            "id": 1,
            "title": "Web Warmup",
            "category": "Web",
            "points": 100,
            "level": 4
        },
        "is_mine": false
    },
    "can_view_content": true
}
```

Notes:

- `content` may be omitted when `can_view_content` is `false`.

Errors:

- 400 `invalid input`
- 404 `writeup not found`

---

## Create Writeup

`POST /api/challenges/{id}/writeups`

Headers

```
Cookie: access_token=<jwt>
```

Request

```json
{
    "content": "# Overview\n..."
}
```

Response 201

```json
{
    "id": 11,
    "content": "# Overview\n...",
    "created_at": "2026-04-26T10:00:00Z",
    "updated_at": "2026-04-26T10:00:00Z",
    "author": {
        "user_id": 7,
        "username": "alice",
        "affiliation_id": 2,
        "affiliation": "Semyeong High",
        "bio": "web / pwn",
        "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg"
    },
    "challenge": {
        "id": 1,
        "title": "Web Warmup",
        "category": "Web",
        "points": 100,
        "level": 4
    },
    "is_mine": true
}
```

Rules:

- One writeup per `(user, challenge)`.
- Only users who solved the challenge can create.

Errors:

- 400 `invalid input`
- 401 `invalid credentials`
- 403 `user blocked`
- 403 `challenge not solved by user`
- 404 `challenge not found`
- 409 `writeup already exists`

---

## Update Writeup

`PATCH /api/writeups/{id}`

Headers

```
Cookie: access_token=<jwt>
```

Request

```json
{
    "content": "Updated markdown body"
}
```

Response 200

```json
{
    "id": 11,
    "content": "Updated markdown body",
    "created_at": "2026-04-26T10:00:00Z",
    "updated_at": "2026-04-26T10:30:00Z",
    "author": {
        "user_id": 7,
        "username": "alice",
        "affiliation_id": 2,
        "affiliation": "Semyeong High",
        "bio": "web / pwn",
        "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg"
    },
    "challenge": {
        "id": 1,
        "title": "Web Warmup",
        "category": "Web",
        "points": 100,
        "level": 4
    },
    "is_mine": true
}
```

Errors:

- 400 `invalid input`
- 401 `invalid credentials`
- 403 `user blocked`
- 403 `writeup access forbidden`
- 404 `writeup not found`

---

## Delete Writeup

`DELETE /api/writeups/{id}`

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
- 401 `invalid credentials`
- 403 `user blocked`
- 403 `writeup access forbidden`
- 404 `writeup not found`

---

## List My Writeups

`GET /api/me/writeups`

Headers

```
Cookie: access_token=<jwt>
```

Query parameters:

- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

- Same shape as challenge writeup list.
- `can_view_content` is always `true`.
- `content` is always included.

Errors:

- 400 `invalid input`
- 401 `invalid credentials`

---

## List User Writeups

`GET /api/users/{id}/writeups`

Query parameters:

- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

- Same shape as challenge writeup list.
- `content` is included per item only when requester solved that challenge.
- Top-level `can_view_content` becomes `true` if at least one item includes `content`.

Errors:

- 400 `invalid input`
- 404 `not found`
