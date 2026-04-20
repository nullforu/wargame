---
title: Leaderboard & Timeline
nav_order: 5
---

## Get Leaderboard

`GET /api/leaderboard`

Response 200

```json
{
    "challenges": [
        {
            "id": 1,
            "title": "pwn-101",
            "category": "Pwn",
            "points": 300
        }
    ],
    "entries": [
        {
            "user_id": 1,
            "username": "user1",
            "score": 300,
            "solves": [
                {
                    "challenge_id": 1,
                    "solved_at": "2026-01-24T12:00:00Z",
                    "is_first_blood": true
                }
            ]
        }
    ]
}
```

Notes:

- Users are sorted by score (descending).
- Blocked users are excluded from score and solve aggregation.

---

## Get Timeline

`GET /api/timeline`

Response 200

```json
{
    "submissions": [
        {
            "timestamp": "2026-01-24T12:00:00Z",
            "user_id": 1,
            "username": "user1",
            "points": 300,
            "challenge_count": 2
        }
    ]
}
```

Notes:

- Solves are grouped by user in time windows.
- Blocked users are excluded.

---

## Cache Refresh Behavior

Leaderboard and timeline caches are invalidated and rebuilt asynchronously when scoreboard-affecting actions occur (for example correct submissions or admin challenge/user status changes).  
The API no longer provides an SSE stream endpoint.
