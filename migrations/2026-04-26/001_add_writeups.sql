BEGIN;

CREATE TABLE IF NOT EXISTS writeups (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    challenge_id BIGINT NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_writeups_challenge_created ON writeups (challenge_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_writeups_user_updated ON writeups (user_id, updated_at DESC, id DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_writeups_user_challenge ON writeups (user_id, challenge_id);

COMMIT;
