-- +goose Up
CREATE
EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE book
(
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       TEXT                           NOT NULL,
    created_at TIMESTAMP        DEFAULT now() NOT NULL,
    updated_at TIMESTAMP        DEFAULT now() NOT NULL
);

-- +goose StatementBegin
CREATE
OR REPLACE FUNCTION update_book_timestamp() RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at
= now();
RETURN NEW;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE
OR REPLACE TRIGGER trigger_update_book_timestamp
    BEFORE
UPDATE
    ON book
    FOR EACH ROW
    EXECUTE FUNCTION update_book_timestamp();

-- +goose Down
DROP TABLE book;
