CREATE TABLE IF NOT EXISTS files (
    id         BIGSERIAL    PRIMARY KEY,
    name       TEXT         NOT NULL,
    size       BIGINT       NOT NULL DEFAULT 0,
    mime_type  TEXT         NOT NULL DEFAULT 'application/octet-stream',
    object_key TEXT         NOT NULL UNIQUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);
