-- +goose Up
-- +goose StatementBegin
ALTER TABLE files ADD COLUMN resume TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE files DROP COLUMN IF EXISTS resume;
-- +goose StatementEnd
