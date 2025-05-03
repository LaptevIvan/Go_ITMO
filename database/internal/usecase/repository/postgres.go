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

func (p *postgresRepository) makeCommit(ctx context.Context, tx pgx.Tx, txErr error) {
	if txErr != nil {
		p.makeRollBack(ctx, tx)
		return
	}
	if txErr = tx.Commit(ctx); txErr != nil && p.logger != nil {
		p.logger.Error("Error during commit", zap.Error(txErr))
	}
}

func (p *postgresRepository) makeRollBack(ctx context.Context, tx pgx.Tx) {
	err := tx.Rollback(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) && p.logger != nil {
		p.logger.Error("Error during rollback", zap.Error(err))
	}
}
func errAuthorConvert(err error) error {
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
	newAuthorRows := make([][]any, len(authors))
	for i := 0; i < len(newAuthorRows); i++ {
		newAuthorRows[i] = []any{authors[i], bookID}
	}

	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"author_book"},
		[]string{"author_id", "book_id"},
		pgx.CopyFromRows(newAuthorRows))

	return errAuthorConvert(err)
}

func (p *postgresRepository) AddBook(ctx context.Context, book entity.Book) (resBook entity.Book, txErr error) {
	var (
		tx  pgx.Tx
		err error
	)

	if tx, err = extractTx(ctx); err != nil {
		tx, err = p.db.Begin(ctx)

		if err != nil {
			return entity.Book{}, err
		}

		defer p.makeCommit(ctx, tx, txErr)
	}

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
	const query = `
SELECT b.id, b.name, b.created_at, b.updated_at, NULLIF(array_agg(ab.author_id), '{NULL}') AS authors
FROM book b
         LEFT JOIN
     author_book ab ON b.id = ab.book_id
WHERE b.id = $1
GROUP BY b.id
`
	var book entity.Book
	err := p.db.QueryRow(ctx, query, idBook).
		Scan(&book.ID, &book.Name, &book.CreatedAt, &book.UpdatedAt, &book.AuthorIDs)

	if errors.Is(err, sql.ErrNoRows) {
		return entity.Book{}, entity.ErrBookNotFound
	}

	if err != nil {
		return entity.Book{}, err
	}

	return book, nil
}

func (p *postgresRepository) GetAuthorBooks(ctx context.Context, idAuthor string) (<-chan entity.Book, error) {
	tx, err := p.db.Begin(ctx)

	if err != nil {
		return nil, err
	}

	const queryBook = `
DECLARE booksCursor CURSOR FOR
    SELECT b.id, b.name, b.created_at, b.updated_at, array_agg(subquery.author_id) AS authors
    FROM book b
             INNER JOIN
         (SELECT ab.book_id, ab.author_id
          FROM author_book ab
                   INNER JOIN (SELECT author_id, book_id FROM author_book WHERE author_id = $1) sub
                              ON ab.book_id = sub.book_id) subquery ON b.id = subquery.book_id
    GROUP BY b.id
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
			if err != nil && p.logger != nil {
				p.logger.Error("error getting books by cursor", zap.Error(err))
				return
			}
			var rowsRead int
			for rows.Next() {
				rowsRead++
				var book entity.Book
				if err = rows.Scan(&book.ID, &book.Name, &book.CreatedAt, &book.UpdatedAt, &book.AuthorIDs); err != nil {
					rows.Close()
					if p.logger != nil {
						p.logger.Error("error getting books by cursor", zap.Error(err))
					}
					return
				}
				select {
				case <-ctx.Done():
					return
				case ans <- book:
				}
			}
			rows.Close()

			if rowsRead == 0 {
				if err = tx.Commit(ctx); err != nil && p.logger != nil {
					p.logger.Error("error making commit", zap.Error(err))
				}
				return
			}
		}
	}()

	return ans, nil
}

func (p *postgresRepository) RegisterAuthor(ctx context.Context, author entity.Author) (entity.Author, error) {
	const queryBook = `
INSERT INTO author (name)
VALUES ($1)
RETURNING id, created_at, updated_at
`
	result := entity.Author{
		Name: author.Name,
	}

	err := p.db.QueryRow(ctx, queryBook, author.Name).Scan(&result.ID, &result.CreatedAt, &result.UpdatedAt)

	if err != nil {
		return entity.Author{}, err
	}

	return result, nil
}

func (p *postgresRepository) ChangeAuthorInfo(ctx context.Context, updAuthor entity.Author) error {
	const queryBook = `
UPDATE author SET name=$1 WHERE id=$2
`
	_, err := p.db.Exec(ctx, queryBook, updAuthor.Name, updAuthor.ID)

	return errAuthorConvert(err)
}

func (p *postgresRepository) GetAuthorInfo(ctx context.Context, idAuthor string) (entity.Author, error) {
	const query = `
SELECT id, name, created_at, updated_at
FROM author
WHERE id = $1
`

	var author entity.Author
	err := p.db.QueryRow(ctx, query, idAuthor).
		Scan(&author.ID, &author.Name, &author.CreatedAt, &author.UpdatedAt)

	if err != nil {
		return entity.Author{}, errAuthorConvert(err)
	}

	return author, nil
}
