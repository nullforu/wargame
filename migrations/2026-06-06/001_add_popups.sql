BEGIN;

CREATE TABLE IF NOT EXISTS popups (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    image_key TEXT NULL,
    image_name VARCHAR(255) NULL,
    link_url TEXT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    link_url TEXT NULL,
    created_by_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_popups_active_created ON popups (is_active, created_at DESC, id DESC);

COMMIT;
