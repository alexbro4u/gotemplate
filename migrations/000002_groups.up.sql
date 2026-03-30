CREATE TABLE IF NOT EXISTS groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE INDEX IF NOT EXISTS idx_groups_name ON groups(name);

INSERT INTO groups (name) VALUES ('users'), ('admin') ON CONFLICT (name) DO NOTHING;
