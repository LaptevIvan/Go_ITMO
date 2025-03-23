package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/project/library/internal/entity"
)

const ErrForeignKeyViolation = "23503"
const log = false

var _ AuthorRepository = (*postgresRepository)(nil)
var _ BooksRepository = (*postgresRepository)(nil)

type postgresRepository struct {
	logger *zap.Logger
	db     *pgxpool.Pool
}

func New(logger *zap.Logger, db *pgxpool.Pool) *postgresRepository {
	return &postgresRepository{
		logger: logger,
		db:     db,
	}
}

func (p *postgresRepository) AddBook(ctx context.Context, book entity.Book) (entity.Book, error) {
	tx, err := p.db.Begin(ctx)

	if err != nil {
		return entity.Book{}, err
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err = tx.Rollback(ctx)
		if err != nil && log {
			p.logger.Error("Error during rollback: " + err.Error())
		}
	}(tx, ctx)

	const queryBook = `
INSERT INTO book (name)
VALUES ($1)
RETURNING id, created_at, updated_at
`
	result := entity.Book{
		Name:      book.Name,
		AuthorIDs: book.AuthorIDs,
	}

	err = tx.QueryRow(ctx, queryBook, book.Name).Scan(&result.ID, &result.CreatedAt, &result.UpdatedAt)

	if err != nil {
		return entity.Book{}, err
	}

	const queryAuthorBooks = `
INSERT INTO author_book
(author_id, book_id)
VALUES ($1, $2)
`
	bookID := result.ID
	for _, authorID := range book.AuthorIDs {
		_, err = tx.Exec(ctx, queryAuthorBooks, authorID, bookID)

		if err != nil {
			var pgErr *pgconn.PgError

			if errors.As(err, &pgErr) && pgErr.Code == ErrForeignKeyViolation {
				return entity.Book{}, fmt.Errorf("author with ID %s does not exist: %w",
					authorID, entity.ErrAuthorNotFound)
			}

			return entity.Book{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return entity.Book{}, err
	}

	return result, nil
}

func (p *postgresRepository) UpdateBook(ctx context.Context, updBook entity.Book) error {
	tx, err := p.db.Begin(ctx)

	if err != nil {
		return err
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err = tx.Rollback(ctx)
		if err != nil && log {
			p.logger.Error("Error during rollback: " + err.Error())
		}
	}(tx, ctx)

	const queryBook = `
UPDATE book SET name=$1
WHERE id=$2
`
	bookID := updBook.ID
	_, err = tx.Exec(ctx, queryBook, updBook.Name, bookID)
	if err != nil {
		return err
	}

	const queryDeleteOldAuthor = `
DELETE FROM author_book WHERE book_id=$1
`
	_, err = tx.Exec(ctx, queryDeleteOldAuthor, bookID)
	if err != nil {
		return err
	}

	const queryAuthorBooks = `
INSERT INTO author_book
(author_id, book_id)
VALUES ($1, $2)
`
	for _, authorID := range updBook.AuthorIDs {
		_, err = tx.Exec(ctx, queryAuthorBooks, authorID, bookID)

		if err != nil {
			var pgErr *pgconn.PgError

			if errors.As(err, &pgErr) && pgErr.Code == ErrForeignKeyViolation {
				return fmt.Errorf("author with ID %s does not exist: %w",
					authorID, entity.ErrAuthorNotFound)
			}
			return err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (p *postgresRepository) GetBook(ctx context.Context, idBook string) (entity.Book, error) {
	tx, err := p.db.Begin(ctx)

	if err != nil {
		return entity.Book{}, err
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err = tx.Rollback(ctx)
		if err != nil && log {
			p.logger.Error("Error during rollback: " + err.Error())
		}
	}(tx, ctx)

	const query = `
SELECT id, name, created_at, updated_at
FROM book
WHERE id = $1 FOR UPDATE 
`
	var book entity.Book
	err = tx.QueryRow(ctx, query, idBook).
		Scan(&book.ID, &book.Name, &book.CreatedAt, &book.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return entity.Book{}, entity.ErrBookNotFound
	}

	if err != nil {
		return entity.Book{}, err
	}

	const queryAuthors = `
SELECT author_id
FROM author_book
WHERE book_id = $1
`

	rows, err := tx.Query(ctx, queryAuthors, idBook)

	if err != nil {
		return entity.Book{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var authorID string

		if err = rows.Scan(&authorID); err != nil {
			return entity.Book{}, err
		}

		book.AuthorIDs = append(book.AuthorIDs, authorID)
	}

	if err = tx.Commit(ctx); err != nil {
		return entity.Book{}, err
	}
	return book, nil
}

func (p *postgresRepository) GetAuthorBooks(ctx context.Context, idAuthor string) ([]entity.Book, error) {
	tx, err := p.db.Begin(ctx)

	if err != nil {
		return nil, err
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err = tx.Rollback(ctx)
		if err != nil && log {
			p.logger.Error("Error during rollback: " + err.Error())
		}
	}(tx, ctx)

	const queryAuthors = `
SELECT book_id
FROM author_book
WHERE author_id = $1
`
	rows, err := tx.Query(ctx, queryAuthors, idAuthor)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ans := make([]entity.Book, 0)
	for rows.Next() {
		var bookID string

		if err = rows.Scan(&bookID); err != nil {
			return nil, err
		}
		book, errGet := p.GetBook(ctx, bookID)
		if errGet != nil {
			return nil, errGet
		}
		ans = append(ans, book)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return ans, nil
}

func (p *postgresRepository) RegisterAuthor(ctx context.Context, author entity.Author) (entity.Author, error) {
	tx, err := p.db.Begin(ctx)

	if err != nil {
		return entity.Author{}, err
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err = tx.Rollback(ctx)
		if err != nil && log {
			p.logger.Error("Error during rollback: " + err.Error())
		}
	}(tx, ctx)

	const queryBook = `
INSERT INTO author (name)
VALUES ($1)
RETURNING id, created_at, updated_at
`
	result := entity.Author{
		Name: author.Name,
	}

	err = tx.QueryRow(ctx, queryBook, author.Name).Scan(&result.ID, &result.CreatedAt, &result.UpdatedAt)

	if err != nil {
		return entity.Author{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return entity.Author{}, err
	}

	return result, nil
}

func (p *postgresRepository) ChangeAuthorInfo(ctx context.Context, updAuthor entity.Author) error {
	tx, err := p.db.Begin(ctx)

	if err != nil {
		return err
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err = tx.Rollback(ctx)
		if err != nil && log {
			p.logger.Error("Error during rollback: " + err.Error())
		}
	}(tx, ctx)

	const queryBook = `
UPDATE author SET name=$1 WHERE id=$2
`
	_, err = tx.Exec(ctx, queryBook, updAuthor.Name, updAuthor.ID)

	if err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (p *postgresRepository) GetAuthorInfo(ctx context.Context, idAuthor string) (entity.Author, error) {
	tx, err := p.db.Begin(ctx)

	if err != nil {
		return entity.Author{}, err
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err = tx.Rollback(ctx)
		if err != nil && log {
			p.logger.Error("Error during rollback: " + err.Error())
		}
	}(tx, ctx)

	const query = `
SELECT id, name, created_at, updated_at
FROM author
WHERE id = $1
`

	var author entity.Author
	err = tx.QueryRow(ctx, query, idAuthor).
		Scan(&author.ID, &author.Name, &author.CreatedAt, &author.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return entity.Author{}, entity.ErrAuthorNotFound
	}

	if err != nil {
		return entity.Author{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return entity.Author{}, err
	}

	return author, nil
}
