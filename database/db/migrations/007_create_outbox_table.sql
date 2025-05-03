-- +goose Up
CREATE TYPE outbox_status as ENUM ('CREATED', 'IN_PROGRESS', 'SUCCESS', 'ABANDONED');

CREATE TABLE outbox
(
    idempotency_key TEXT PRIMARY KEY,
    data            JSONB                   NOT NULL,
    status          outbox_status           NOT NULL,
    kind            INT                     NOT NULL,
    attempts        INT                     NOT NULL,
    created_at      TIMESTAMP DEFAULT now() NOT NULL,
    updated_at      TIMESTAMP DEFAULT now() NOT NULL
);

-- +goose Down
DROP TABLE outbox

