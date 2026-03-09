-- +goose Up
CREATE TABLE outbox_events (
    id          BIGSERIAL   PRIMARY KEY,
    routing_key TEXT        NOT NULL,
    payload     BYTEA       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published   BOOLEAN     NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_outbox_events_unpublished ON outbox_events (created_at) WHERE NOT published;

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
