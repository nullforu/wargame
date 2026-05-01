---
title: Affiliations
nav_order: 9
---

Notes:

- For authenticated `POST`, `PUT`, `PATCH`, and `DELETE` requests, send both `csrf_token` cookie and matching `X-CSRF-Token` header.

## Create Affiliation (Admin)

`POST /api/admin/affiliations`

Headers

```
Cookie: access_token=<jwt>
```

Request

```json
{
    "name": "Blue Team"
}
```

Response 201

```json
{
    "id": 1,
    "name": "Blue Team"
}
```

Errors:

- 400 `invalid input` (required/duplicate name)
- 401 `invalid token` or `missing access_token cookie`
- 403 `forbidden`

---

## List Affiliations

`GET /api/affiliations`

Query parameters:

- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "affiliations": [
        {
            "id": 1,
            "name": "Blue Team"
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

---

## Search Affiliations

`GET /api/affiliations/search`

Query parameters:

- `q` (required, affiliation name keyword, case-insensitive partial match)
- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "affiliations": [
        {
            "id": 1,
            "name": "Blue Team"
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

- 400 `invalid input` (`q` required)

---

## List Users In Affiliation

`GET /api/affiliations/{id}/users`

Query parameters:

- `page` (optional, default `1`)
- `page_size` (optional, default `20`, max `100`)

Response 200

```json
{
    "users": [
        {
            "id": 10,
            "username": "player1",
            "role": "user",
            "affiliation_id": 1,
            "affiliation": "Blue Team",
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
- 404 `not found`

---

## Update My Affiliation

`PUT /api/me`

Headers

```
Cookie: access_token=<jwt>
```

Request examples

Set affiliation:

```json
{
    "affiliation_id": 2
}
```

Clear affiliation (nullable):

```json
{
    "affiliation_id": null
}
```
