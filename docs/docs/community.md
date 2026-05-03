---
title: Community
nav_order: 11
---

## List Posts

`GET /api/community`

Query parameters:

- `q` (optional, title/content keyword)
- `category` (optional, `0..3`: notice/free/qna/humor)
- `sort` (optional, one of `latest`, `oldest`, `popular`)
- `exclude_notice` (optional, `true|1`, excludes notice category posts)
- `popular_only` (optional, `true|1`, `like_count >= 5`)
- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Notes:

- If `category` is set, it is applied first.
- `exclude_notice=true` removes notice posts from the result.
- `popular_only=true` returns only posts with `like_count >= 5` after other filters.

Response 200

```json
{
    "posts": [
        {
            "id": 101,
            "category": 1,
            "title": "hello",
            "content": "**markdown** body",
            "view_count": 25,
            "like_count": 12,
            "comment_count": 4,
            "liked_by_me": false,
            "created_at": "2026-05-01T12:00:00Z",
            "updated_at": "2026-05-01T12:30:00Z",
            "author": {
                "user_id": 7,
                "username": "alice",
                "affiliation_id": 2,
                "affiliation": "Semyeong High",
                "bio": "web / pwn",
                "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg"
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

Errors:

- 400 `invalid input`

---

## Get Post Detail

`GET /api/community/{id}`

Response 200

```json
{
    "post": {
        "id": 101,
        "category": 1,
        "title": "hello",
        "content": "**markdown** body",
        "view_count": 26,
        "like_count": 12,
        "comment_count": 4,
        "liked_by_me": true,
        "created_at": "2026-05-01T12:00:00Z",
        "updated_at": "2026-05-01T12:30:00Z",
        "author": {
            "user_id": 7,
            "username": "alice",
            "affiliation_id": 2,
            "affiliation": "Semyeong High",
            "bio": "web / pwn",
            "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg"
        }
    }
}
```

Errors:

- 400 `invalid input`
- 404 `community post not found`

---

## List Post Likes

`GET /api/community/{id}/likes`

Query parameters:

- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "likes": [
        {
            "user_id": 7,
            "username": "alice",
            "affiliation_id": 2,
            "affiliation": "Semyeong High",
            "bio": "web / pwn",
            "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg",
            "created_at": "2026-05-01T12:20:00Z"
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
- 404 `community post not found`

---

## List Post Comments

`GET /api/community/{id}/comments`

Query parameters:

- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "comments": [
        {
            "id": 11,
            "content": "good post",
            "created_at": "2026-05-01T12:20:00Z",
            "updated_at": "2026-05-01T12:20:00Z",
            "author": {
                "user_id": 7,
                "username": "alice",
                "affiliation_id": 2,
                "affiliation": "Semyeong High",
                "bio": "web / pwn",
                "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg"
            },
            "post": {
                "id": 101,
                "title": "hello"
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

Errors:

- 400 `invalid input`
- 404 `community post not found`

---

## Toggle Like

`POST /api/community/{id}/likes`

Headers

```text
Cookie: access_token=<jwt>
X-CSRF-Token: <csrf_token cookie value>
```

Response 200

```json
{
    "status": "ok",
    "liked": true,
    "like_count": 12
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie`
- 403 `user blocked`
- 404 `community post not found`

---

## Create Post

`POST /api/community`

Headers

```text
Cookie: access_token=<jwt>
X-CSRF-Token: <csrf_token cookie value>
```

Request

```json
{
    "category": 1,
    "title": "hello",
    "content": "**markdown** body"
}
```

Response 201

```json
{
    "id": 101,
    "category": 1,
    "title": "hello",
    "content": "**markdown** body",
    "view_count": 0,
    "like_count": 0,
    "comment_count": 0,
    "liked_by_me": false,
    "created_at": "2026-05-01T12:00:00Z",
    "updated_at": "2026-05-01T12:00:00Z",
    "author": {
        "user_id": 7,
        "username": "alice",
        "affiliation_id": 2,
        "affiliation": "Semyeong High",
        "bio": "web / pwn",
        "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg"
    }
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie`
- 403 `user blocked`
- 403 `community access forbidden`

---

## Update Post

`PATCH /api/community/{id}`

Headers

```text
Cookie: access_token=<jwt>
X-CSRF-Token: <csrf_token cookie value>
```

Request

```json
{
    "category": 1,
    "title": "updated title",
    "content": "updated markdown body"
}
```

Response 200

```json
{
    "id": 101,
    "category": 1,
    "title": "updated title",
    "content": "updated markdown body",
    "view_count": 0,
    "like_count": 0,
    "comment_count": 0,
    "liked_by_me": false,
    "created_at": "2026-05-01T12:00:00Z",
    "updated_at": "2026-05-01T12:30:00Z",
    "author": {
        "user_id": 7,
        "username": "alice",
        "affiliation_id": 2,
        "affiliation": "Semyeong High",
        "bio": "web / pwn",
        "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg"
    }
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie`
- 403 `user blocked`
- 403 `community access forbidden`
- 404 `community post not found`

---

## Create Comment

`POST /api/community/{id}/comments`

Headers

```text
Cookie: access_token=<jwt>
X-CSRF-Token: <csrf_token cookie value>
```

Request

```json
{
    "content": "good post"
}
```

Response 201

```json
{
    "id": 11,
    "content": "good post",
    "created_at": "2026-05-01T12:20:00Z",
    "updated_at": "2026-05-01T12:20:00Z",
    "author": {
        "user_id": 7,
        "username": "alice",
        "affiliation_id": 2,
        "affiliation": "Semyeong High",
        "bio": "web / pwn",
        "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg"
    },
    "post": {
        "id": 101,
        "title": "hello"
    }
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie`
- 403 `user blocked`
- 404 `community post not found`

---

## Update Comment

`PATCH /api/community/comments/{id}`

Headers

```text
Cookie: access_token=<jwt>
X-CSRF-Token: <csrf_token cookie value>
```

Request

```json
{
    "content": "updated comment"
}
```

Response 200

```json
{
    "id": 11,
    "content": "updated comment",
    "created_at": "2026-05-01T12:20:00Z",
    "updated_at": "2026-05-01T12:30:00Z",
    "author": {
        "user_id": 7,
        "username": "alice",
        "affiliation_id": 2,
        "affiliation": "Semyeong High",
        "bio": "web / pwn",
        "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440000.jpg"
    },
    "post": {
        "id": 101,
        "title": "hello"
    }
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie`
- 403 `user blocked`
- 403 `community comment access forbidden`
- 404 `community comment not found`

---

## Delete Comment

`DELETE /api/community/comments/{id}`

Headers

```text
Cookie: access_token=<jwt>
X-CSRF-Token: <csrf_token cookie value>
```

Response 200

```json
{
    "status": "ok"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie`
- 403 `user blocked`
- 403 `community comment access forbidden`
- 404 `community comment not found`

---

## Delete Post

`DELETE /api/community/{id}`

Headers

```text
Cookie: access_token=<jwt>
X-CSRF-Token: <csrf_token cookie value>
```

Response 200

```json
{
    "status": "ok"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie`
- 403 `user blocked`
- 403 `community access forbidden`
- 404 `community post not found`
