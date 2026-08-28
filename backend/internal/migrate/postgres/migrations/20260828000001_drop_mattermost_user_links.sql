-- +goose Up

DROP TABLE IF EXISTS mattermost_user_links;

CREATE INDEX IF NOT EXISTS idx_users_mattermost_id ON users(mattermost_id);

-- +goose Down

DROP INDEX IF EXISTS idx_users_mattermost_id;

CREATE TABLE mattermost_user_links (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    mm_user_id TEXT NOT NULL UNIQUE
);
