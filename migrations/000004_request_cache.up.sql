CREATE TABLE IF NOT EXISTS request_cache (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(uuid),
    path VARCHAR(500) NOT NULL,
    http_verb VARCHAR(10) NOT NULL,
    request_id VARCHAR(255) NOT NULL,
    response BYTEA,
    status_code INTEGER NOT NULL,
    content_type VARCHAR(255) NOT NULL DEFAULT 'application/json',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_request_cache_unique
    ON request_cache(user_id, path, http_verb, request_id);

CREATE INDEX IF NOT EXISTS idx_request_cache_user_request_id
    ON request_cache(user_id, request_id);

CREATE INDEX IF NOT EXISTS idx_request_cache_expires_at
    ON request_cache(expires_at);
