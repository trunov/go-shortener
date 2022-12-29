-- +goose Up
-- +goose StatementBegin
ALTER TABLE shortener
ADD is_deleted boolean DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
