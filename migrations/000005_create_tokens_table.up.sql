CREATE TABLE IF NOT EXISTS tokens (
    hash bytea PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users ON DELETE CASCADE,
    expiry TIMESTAMPTZ NOT NULL,
    scope text NOT NULL
);

CREATE INDEX IF NOT EXISTS tokens_user_id_idx ON tokens (user_id);