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

## Scoreboard Stream (SSE)

`GET /api/scoreboard/stream`

Opens a Server-Sent Events stream that notifies clients when leaderboard/timeline caches are rebuilt.

### Events

- `ready`: sent immediately after connection.
- `scoreboard`: sent after cache rebuild.

Payload schema:

```json
{
    "scope": "all",
    "reason": "submission_correct",
    "ts": "2026-02-27T18:00:00Z"
}
```

### Scoreboard-Affecting APIs

| Action             | API                                 | Reason               |
| ------------------ | ----------------------------------- | -------------------- |
| Correct submission | `POST /api/challenges/{id}/submit`  | `submission_correct` |
| Challenge created  | `POST /api/admin/challenges`        | `challenge_created`  |
| Challenge updated  | `PUT /api/admin/challenges/{id}`    | `challenge_updated`  |
| Challenge deleted  | `DELETE /api/admin/challenges/{id}` | `challenge_deleted`  |
| Block user         | `POST /api/admin/users/{id}/block`  | `user_blocked`       |
| Unblock user       | `POST /api/admin/users/{id}/unblock`| `user_unblocked`     |

Example stream:

```
event: ready
data: {}

event: scoreboard
data: {"scope":"all","reason":"submission_correct","ts":"2026-02-27T18:00:00Z"}
```
