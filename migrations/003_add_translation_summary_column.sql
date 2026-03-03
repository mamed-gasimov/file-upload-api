-- +goose Up
-- +goose StatementBegin
ALTER TABLE files ADD COLUMN translation_summary TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE files DROP COLUMN IF EXISTS translation_summary;
-- +goose StatementEnd
