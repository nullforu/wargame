BEGIN;

CREATE TABLE IF NOT EXISTS community_posts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    category INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    view_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_community_posts_category_created
    ON community_posts (category, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_community_posts_views_created
    ON community_posts (view_count DESC, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_community_posts_user_updated
    ON community_posts (user_id, updated_at DESC, id DESC);

COMMIT;
