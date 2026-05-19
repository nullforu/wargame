BEGIN;

CREATE TABLE IF NOT EXISTS challenge_series (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    created_by_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS challenge_series_challenges (
    id BIGSERIAL PRIMARY KEY,
    series_id BIGINT NOT NULL REFERENCES challenge_series(id) ON DELETE CASCADE,
    challenge_id BIGINT NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    position INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(series_id, challenge_id),
    UNIQUE(series_id, position)
);

CREATE INDEX IF NOT EXISTS idx_challenge_series_created ON challenge_series (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_challenge_series_challenges_challenge ON challenge_series_challenges (challenge_id);

COMMIT;
