package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"maps"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/project/library/internal/entity"
	"go.uber.org/zap"
)

const ErrForeignKeyViolation = "23503"
const log = true

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

func (p *postgresRepository) makeRollBack(ctx context.Context, tx pgx.Tx) {
	err := tx.Rollback(ctx)
	if err != nil && log {
		p.logger.Error("Error during rollback", zap.Error(err))
	}
}

func errConvert(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == ErrForeignKeyViolation {
		return fmt.Errorf("Unknown author was: %w", entity.ErrAuthorNotFound)
	}

	if errors.Is(err, sql.ErrNoRows) {
		return entity.ErrAuthorNotFound
	}

	return err
}

func (p *postgresRepository) addBookAuthors(ctx context.Context, tx pgx.Tx, bookID string, authors []string) error {
	newAuthorRows := make([][]interface{}, len(authors))
	for i := 0; i < len(newAuthorRows); i++ {
		newAuthorRows[i] = []interface{}{authors[i], bookID}
	}

	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"author_book"},
		[]string{"author_id", "book_id"},
		pgx.CopyFromRows(newAuthorRows))

	if err != nil {
		return errConvert(err)
	}

	return nil
}

func (p *postgresRepository) AddBook(ctx context.Context, book entity.Book) (entity.Book, error) {
	tx, err := p.db.Begin(ctx)

	if err != nil {
		return entity.Book{}, err
	}
	defer p.makeRollBack(ctx, tx)

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

	err = p.addBookAuthors(ctx, tx, result.ID, book.AuthorIDs)
	if err != nil {
		return entity.Book{}, err
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
	defer p.makeRollBack(ctx, tx)

	const queryBookUpdate = `
UPDATE book SET name=$1 where id=$2
`
	_, err = tx.Exec(ctx, queryBookUpdate, updBook.Name, updBook.ID)
	if err != nil {
		return err
	}

	const queryGetCurrentAuthor = `
SELECT (author_id) FROM author_book WHERE book_id=$1 
`
	rows, err := tx.Query(ctx, queryGetCurrentAuthor, updBook.ID)
	if err != nil {
		return err
	}

	curAuthors := make(map[string]struct{}, 0)
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return err
		}
		curAuthors[id] = struct{}{}
	}
	newAuthors := make([]string, 0)
	for _, id := range updBook.AuthorIDs {
		if _, ok := curAuthors[id]; !ok {
			newAuthors = append(newAuthors, id)
			continue
		}
		delete(curAuthors, id)
	}

	const queryDeleteExcessAuthors = `
DELETE FROM author_book where author_id = ANY($1)
`
	excessAuthors := make([]string, 0)
	maps.Keys(curAuthors)(func(id string) bool {
		excessAuthors = append(excessAuthors, id)
		return true
	})

	_, err = tx.Exec(ctx, queryDeleteExcessAuthors, excessAuthors)
	if err != nil {
		return err
	}

	err = p.addBookAuthors(ctx, tx, updBook.ID, newAuthors)
	if err != nil {
		return err
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
	defer p.makeRollBack(ctx, tx)

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

func (p *postgresRepository) GetAuthorBooks(ctx context.Context, idAuthor string) (<-chan entity.Book, error) {
	tx, err := p.db.Begin(ctx)

	if err != nil {
		return nil, err
	}

	const queryBook = `DECLARE booksCursor CURSOR FOR 
SELECT book_id FROM author_book
WHERE author_id=$1
`
	_, err = tx.Exec(ctx, queryBook, idAuthor)
	if err != nil {
		return nil, err
	}

	const n = 10
	queryGetBook := fmt.Sprintf("FETCH %d FROM booksCursor", n)
	ans := make(chan entity.Book, n)
	go func() {
		defer p.makeRollBack(ctx, tx)
		defer close(ans)
		for {
			rows, err := tx.Query(ctx, queryGetBook)
			if err != nil && log {
				p.logger.Error("error getting books by cursor", zap.Error(err))
				return
			}
			var rowsRead int
			for rows.Next() {
				rowsRead++
				var id string
				if err = rows.Scan(&id); err != nil {
					rows.Close()
					if log {
						p.logger.Error("error getting books by cursor", zap.Error(err))
					}
					return
				}
				book, _ := p.GetBook(ctx, id)
				ans <- book
			}
			rows.Close()

			if rowsRead == 0 {
				if err = tx.Commit(ctx); err != nil && log {
					p.logger.Error("error making commit", zap.Error(err))
				}
				return
			}
		}
	}()

	return ans, nil
}

func (p *postgresRepository) RegisterAuthor(ctx context.Context, author entity.Author) (entity.Author, error) {
	tx, err := p.db.Begin(ctx)

	if err != nil {
		return entity.Author{}, err
	}
	defer p.makeRollBack(ctx, tx)

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
	defer p.makeRollBack(ctx, tx)

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
	defer p.makeRollBack(ctx, tx)

	const query = `
SELECT id, name, created_at, updated_at
FROM author
WHERE id = $1
`

	var author entity.Author
	err = tx.QueryRow(ctx, query, idAuthor).
		Scan(&author.ID, &author.Name, &author.CreatedAt, &author.UpdatedAt)

	if err != nil {
		return entity.Author{}, errConvert(err)
	}

	if err = tx.Commit(ctx); err != nil {
		return entity.Author{}, err
	}

	return author, nil
}
