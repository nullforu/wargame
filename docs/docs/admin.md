---
title: Admin
nav_order: 6
---
---

## Block User

`POST /api/admin/users/{id}/block`

Headers

```
Cookie: access_token=<jwt>
```

Request

```json
{
    "reason": "policy violation"
}
```

Response 200

```json
{
    "id": 10,
    "email": "user1@example.com",
    "username": "user1",
    "role": "blocked",
    "blocked_reason": "policy violation",
    "blocked_at": "2026-01-26T12:00:00Z"
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `forbidden`
- 404 `not found`

---

## Unblock User

`POST /api/admin/users/{id}/unblock`

Headers

```
Cookie: access_token=<jwt>
```

Response 200

```json
{
    "id": 10,
    "email": "user1@example.com",
    "username": "user1",
    "role": "user",
    "blocked_reason": null,
    "blocked_at": null
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `forbidden`
- 404 `not found`

---

## Create Challenge

`POST /api/admin/challenges`

Headers

```
Cookie: access_token=<jwt>
```

Request

```json
{
    "title": "New Challenge",
    "description": "...",
    "category": "Web",
    "points": 200,
    "flag": "flag{...}",
    "previous_challenge_id": 1,
    "is_active": true,
    "stack_enabled": false,
    "stack_target_ports": [
        {
            "container_port": 80,
            "protocol": "TCP"
        }
    ],
    "stack_pod_spec": "apiVersion: v1\nkind: Pod\n..."
}
```

Response 201

```json
{
    "id": 2,
    "title": "New Challenge",
    "description": "...",
    "category": "Web",
    "level": 0,
    "points": 200,
    "solve_count": 0,
    "previous_challenge_id": 1,
    "is_active": true,
    "is_locked": false,
    "has_file": false,
    "stack_enabled": false,
    "stack_target_ports": []
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `forbidden`

---

## Update Challenge

`PUT /api/admin/challenges/{id}`

Headers

```
Cookie: access_token=<jwt>
```

Request

All fields are optional. Omitted fields are unchanged.

```json
{
    "title": "Updated Challenge",
    "points": 250,
    "flag": "flag{rotated}",
    "is_active": false,
    "stack_enabled": true,
    "stack_target_ports": [
        {
            "container_port": 80,
            "protocol": "TCP"
        }
    ],
    "stack_pod_spec": "apiVersion: v1\nkind: Pod\n..."
}
```

Response 200

```json
{
    "id": 2,
    "title": "Updated Challenge",
    "description": "...",
    "category": "Crypto",
    "level": 0,
    "points": 250,
    "solve_count": 12,
    "is_active": false,
    "is_locked": false,
    "has_file": true,
    "file_name": "challenge.zip",
    "stack_enabled": true,
    "stack_target_ports": [
        {
            "container_port": 80,
            "protocol": "TCP"
        }
    ]
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `forbidden`
- 404 `challenge not found`

---

## Get Challenge Detail (Admin)

`GET /api/admin/challenges/{id}`

Headers

```
Cookie: access_token=<jwt>
```

Response 200

```json
{
    "id": 2,
    "title": "Updated Challenge",
    "description": "...",
    "category": "Crypto",
    "level": 0,
    "points": 250,
    "solve_count": 12,
    "is_active": false,
    "is_locked": false,
    "has_file": true,
    "file_name": "challenge.zip",
    "stack_enabled": true,
    "stack_target_ports": [
        {
            "container_port": 80,
            "protocol": "TCP"
        }
    ],
    "stack_pod_spec": "apiVersion: v1\nkind: Pod\n..."
}
```

Errors:

- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `forbidden`
- 404 `challenge not found`

---

## Delete Challenge

`DELETE /api/admin/challenges/{id}`

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

- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `forbidden`
- 404 `challenge not found`

---

## Stack Management (Admin)

Headers

```
Cookie: access_token=<jwt>
```

### List All Stacks

`GET /api/admin/stacks`

Response 200

```json
{
    "stacks": [
        {
            "stack_id": "stack-716b6384dd477b0b",
            "ttl_expires_at": "2026-02-10T04:02:26.535664Z",
            "created_at": "2026-02-10T02:02:26.535664Z",
            "updated_at": "2026-02-10T02:06:33.16031Z",
            "user_id": 12,
            "username": "alice",
            "email": "alice@example.com",
            "challenge_id": 5,
            "challenge_title": "Web 1",
            "challenge_category": "Web"
        }
    ]
}
```

### Get Stack Detail

`GET /api/admin/stacks/{stack_id}`

Response 200

```json
{
    "stack_id": "stack-716b6384dd477b0b",
    "challenge_id": 5,
    "status": "running",
    "node_public_ip": "12.34.56.78",
    "ports": [
        {
            "container_port": 80,
            "protocol": "TCP",
            "node_port": 31538
        }
    ],
    "ttl_expires_at": "2026-02-10T04:02:26.535664Z",
    "created_at": "2026-02-10T02:02:26.535664Z",
    "updated_at": "2026-02-10T02:06:33.16031Z",
    "created_by_user_id": 12,
    "created_by_username": "alice",
    "challenge_title": "Web 1"
}
```

### Delete Stack

`DELETE /api/admin/stacks/{stack_id}`

Response 200

```json
{
    "deleted": true,
    "stack_id": "stack-716b6384dd477b0b"
}
```

Errors (all admin stack endpoints):

- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `forbidden`
- 404 `stack not found`
- 503 `stack feature disabled` or `stack provisioner unavailable`

---

## Upload Challenge File

`POST /api/admin/challenges/{id}/file/upload`

Headers

```
Cookie: access_token=<jwt>
```

Request

```json
{
    "filename": "challenge.zip"
}
```

Response 200

```json
{
    "challenge": {
        "id": 2,
        "title": "New Challenge",
        "description": "...",
        "category": "Web",
        "level": 0,
        "points": 200,
        "solve_count": 0,
        "is_active": true,
        "is_locked": false,
        "has_file": true,
        "file_name": "challenge.zip",
        "stack_enabled": false,
        "stack_target_ports": []
    },
    "upload": {
        "url": "https://s3.example.com/...",
        "fields": {
            "key": "uuid.zip",
            "Content-Type": "application/zip"
        },
        "expires_at": "2026-01-01T00:00:00Z"
    }
}
```

Errors:

- 400 `invalid input`
- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `forbidden`
- 404 `challenge not found`
- 503 `storage unavailable`

---

## Delete Challenge File

`DELETE /api/admin/challenges/{id}/file`

Headers

```
Cookie: access_token=<jwt>
```

Response 200

```json
{
    "id": 2,
    "title": "New Challenge",
    "description": "...",
    "category": "Web",
    "level": 0,
    "points": 200,
    "solve_count": 0,
    "is_active": true,
    "is_locked": false,
    "has_file": false,
    "stack_enabled": false,
    "stack_target_ports": []
}
```

Errors:

- 401 `invalid token` or `missing access_token cookie` or `invalid token`
- 403 `forbidden`
- 404 `challenge not found` or `challenge file not found`
- 503 `storage unavailable`
