-- +goose Up

CREATE TABLE realm_mattermost (
    realm_id UUID PRIMARY KEY REFERENCES realms(id) ON DELETE CASCADE,
    bot_token TEXT NOT NULL,
    bot_user_id TEXT NOT NULL DEFAULT '',
    channel_id TEXT NOT NULL DEFAULT '',
    webhook_secret TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE mattermost_user_links (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    mm_user_id TEXT NOT NULL UNIQUE
);

CREATE INDEX idx_mattermost_user_links_mm_user ON mattermost_user_links(mm_user_id);

-- +goose Down

DROP TABLE IF EXISTS mattermost_user_links;
DROP TABLE IF EXISTS realm_mattermost;
