BEGIN;

CREATE TABLE IF NOT EXISTS challenge_comments (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    challenge_id BIGINT NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    content VARCHAR(500) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_challenge_comments_challenge_created
    ON challenge_comments (challenge_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_challenge_comments_user_updated
    ON challenge_comments (user_id, updated_at DESC, id DESC);

COMMIT;
