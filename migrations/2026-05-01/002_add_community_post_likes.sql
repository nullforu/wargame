BEGIN;

CREATE TABLE IF NOT EXISTS community_post_likes (
    post_id BIGINT NOT NULL REFERENCES community_posts (id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_community_post_likes_post_created
    ON community_post_likes (post_id, created_at DESC, user_id DESC);

CREATE INDEX IF NOT EXISTS idx_community_post_likes_user_created
    ON community_post_likes (user_id, created_at DESC, post_id DESC);

COMMIT;
