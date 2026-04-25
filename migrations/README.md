Example use of the migration script:

```bash
PGPASSWORD=app_password psql -U app_user -d app_db -h 10.10.0.1 -p 5432 < migrations/2026-04-25/001_add_user_bio.sql
PGPASSWORD=app_password psql -U app_user -d app_db -h 10.10.0.1 -p 5432 < migrations/2026-04-25/999_rollback.sql
```
