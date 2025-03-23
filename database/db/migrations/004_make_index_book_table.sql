-- +goose Up
CREATE INDEX id_book ON book (id);

-- +goose Down
DROP INDEX id_book;