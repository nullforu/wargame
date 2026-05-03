---
title: Ranking, Leaderboard & Timeline
nav_order: 5
---

## Get Leaderboard (Legacy)

`GET /api/leaderboard?page=1&page_size=20`

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
            "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440001.jpg",
            "score": 300,
            "solves": [
                {
                    "challenge_id": 1,
                    "solved_at": "2026-01-24T12:00:00Z",
                    "is_first_blood": true
                }
            ]
        }
    ],
    "pagination": {
        "page": 1,
        "page_size": 20,
        "total_count": 37,
        "total_pages": 2,
        "has_prev": false,
        "has_next": true
    }
}

```

---

## Get User Ranking

`GET /api/rankings/users?page=1&page_size=20`

Response 200

```json
{
    "entries": [
        {
            "user_id": 3,
            "username": "user3",
            "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440003.jpg",
            "score": 500,
            "solved_count": 4,
            "affiliation_id": 2,
            "affiliation_name": "Blue Team",
            "bio": "forensics / web"
        }
    ],
    "pagination": {
        "page": 1,
        "page_size": 20,
        "total_count": 37,
        "total_pages": 2,
        "has_prev": false,
        "has_next": true
    }
}
```

Notes:

- Users are sorted by `score DESC`, `solved_count DESC`, `user_id ASC`.
- Blocked users are excluded.

---

## Get Affiliation Total Ranking

`GET /api/rankings/affiliations?page=1&page_size=20`

Response 200

```json
{
    "entries": [
        {
            "affiliation_id": 2,
            "name": "Blue Team",
            "score": 1200,
            "solved_count": 9,
            "user_count": 4
        }
    ],
    "pagination": {
        "page": 1,
        "page_size": 20,
        "total_count": 8,
        "total_pages": 1,
        "has_prev": false,
        "has_next": false
    }
}
```

Notes:

- Ranking contains affiliations only. Unaffiliated users are not part of this table.

---

## Get Ranking Users in an Affiliation

`GET /api/rankings/affiliations/:id/users?page=1&page_size=20`

Response 200

```json
{
    "entries": [
        {
            "user_id": 3,
            "username": "user3",
            "profile_image": "profiles/550e8400-e29b-41d4-a716-446655440003.jpg",
            "score": 500,
            "solved_count": 4,
            "affiliation_id": 2,
            "affiliation_name": "Blue Team",
            "bio": "forensics / web"
        }
    ],
    "pagination": {
        "page": 1,
        "page_size": 20,
        "total_count": 4,
        "total_pages": 1,
        "has_prev": false,
        "has_next": false
    }
}
```

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
