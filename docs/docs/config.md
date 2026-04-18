---
title: Config
nav_order: 7
---

## Get Site Configuration

`GET /api/config`

Response 200

```json
{
    "title": "Welcome to Wargame.",
    "description": "Check out the repository for setup instructions.",
    "header_title": "CTF",
    "header_description": "Capture The Flag",
    "wargame_start_at": "2099-12-31T10:00:00Z",
    "wargame_end_at": "2099-12-31T18:00:00Z",
    "updated_at": "2026-01-26T12:00:00Z"
}
```

Notes:

- Response includes `ETag` and `Cache-Control: no-cache` for caching.
- `wargame_start_at` and `wargame_end_at` are RFC3339 timestamps. Empty values mean the wargame is always active.

Errors:

- 500 `internal error`
