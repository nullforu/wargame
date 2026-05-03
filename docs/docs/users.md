---
title: Users
nav_order: 3
---

Notes:

- For authenticated `POST`, `PUT`, `PATCH`, and `DELETE` requests, send both `csrf_token` cookie and matching `X-CSRF-Token` header.

## Me

`GET /api/me`

Headers

```
Cookie: access_token=<jwt>
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
    "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg",
    "stack_count": 0,
    "stack_limit": 3,
    "blocked_reason": null,
    "blocked_at": null
}
```

Errors:

- 401 `invalid token` or `missing access_token cookie`

---

## Update Me

`PUT /api/me`

Headers

```
Cookie: access_token=<jwt>
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
    "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg",
    "stack_count": 0,
    "stack_limit": 3,
    "blocked_reason": null,
    "blocked_at": null
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie`
- 403 `user blocked`
- 409 `user already exists` (username already in use)

---

## Upload Profile Image

`POST /api/me/profile-image/upload`

Headers

```
Cookie: access_token=<jwt>
```

Request

```json
{
    "filename": "avatar.png"
}
```

Response 200

```json
{
    "user": {
        "id": 1,
        "email": "user@example.com",
        "username": "new_username",
        "role": "user",
        "affiliation_id": 2,
        "affiliation": "Blue Team",
        "bio": "Blue Team player",
        "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg",
        "stack_count": 0,
        "stack_limit": 3,
        "blocked_reason": null,
        "blocked_at": null
    },
    "upload": {
        "url": "https://media.example.com/...",
        "method": "POST",
        "fields": {
            "key": "profiles/550e8400-e29b-41d4-a716-446655440000.png",
            "Content-Type": "image/png"
        },
        "expires_at": "2026-01-01T00:00:00Z"
    }
}
```

Notes for this response:

- This endpoint only issues a presigned upload and does not update `profile_image` in DB.
- `user.profile_image` returns the current saved key (or `null`) until finalize API succeeds.

Validation and policy notes:

- Allowed filename extensions: `.png`, `.jpg`, `.jpeg`
- Key format: `profiles/{uuid}.{ext}`
- Upload method is always `POST`
- Max size is limited to `100KB` by presigned POST policy (`content-length-range`)
- API stores only the object key (for example `profiles/550e8400-e29b-41d4-a716-446655440000.png`) in DB. Client should render using CDN base URL + key.

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie`
- 403 `user blocked`
- 503 `storage unavailable`

---

## Finalize Profile Image Upload

`PUT /api/me/profile-image`

Headers

```
Cookie: access_token=<jwt>
```

Request

```json
{
    "key": "profiles/550e8400-e29b-41d4-a716-446655440000.png"
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
    "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.png",
    "stack_count": 0,
    "stack_limit": 3,
    "blocked_reason": null,
    "blocked_at": null
}
```

Finalize behavior:

- Validates key format as `profiles/{uuid}.{ext}` with `.png`, `.jpg`, `.jpeg`.
- Saves only the key into DB.
- If user already had a profile image key, DB switches to the new key first, then old object is deleted as cleanup.

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie`
- 403 `user blocked`
- 503 `storage unavailable`

---

## Delete Profile Image

`DELETE /api/me/profile-image`

Headers

```
Cookie: access_token=<jwt>
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
    "profile_image": null,
    "stack_count": 0,
    "stack_limit": 3,
    "blocked_reason": null,
    "blocked_at": null
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie`
- 403 `user blocked`
- 503 `storage unavailable`

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
            "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg",
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
            "profile_image": null,
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
            "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg",
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
    "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg",
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
