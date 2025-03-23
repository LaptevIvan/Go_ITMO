-- +goose Up
CREATE INDEX book_id ON author_book (book_id);

-- +goose Down
DROP INDEX book_id;
