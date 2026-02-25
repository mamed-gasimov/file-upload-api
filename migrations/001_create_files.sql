-- +goose Up
-- +goose StatementBegin
CREATE TABLE files (
    id         BIGSERIAL    PRIMARY KEY,
    name       TEXT         NOT NULL,
    size       BIGINT       NOT NULL DEFAULT 0,
    mime_type  TEXT         NOT NULL DEFAULT 'application/octet-stream',
    object_key TEXT         NOT NULL UNIQUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS files;
-- +goose StatementEnd
