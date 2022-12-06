-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS shortener
(
    id              SERIAL PRIMARY KEY,
    short_url       TEXT,
    original_url    TEXT,
    user_id         VARCHAR(24),
    created_at      TIMESTAMP NOT NULL DEFAULT now(),
    updated_at      TIMESTAMP NOT NULL DEFAULT now(),
    CONSTRAINT original_url_unique UNIQUE (original_url)
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
