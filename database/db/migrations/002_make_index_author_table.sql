-- +goose Up
CREATE INDEX id_author ON author (id);

-- +goose Down
DROP INDEX id_author;