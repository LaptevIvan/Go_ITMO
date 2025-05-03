package repository

import (
	"context"
	"time"

	"github.com/project/library/internal/entity"
)

type (
	AuthorRepository interface {
		RegisterAuthor(ctx context.Context, author entity.Author) (entity.Author, error)
		ChangeAuthorInfo(ctx context.Context, updAuthor entity.Author) error
		GetAuthorInfo(ctx context.Context, idAuthor string) (entity.Author, error)
	}

	BooksRepository interface {
		AddBook(ctx context.Context, book entity.Book) (entity.Book, error)
		UpdateBook(ctx context.Context, updBook entity.Book) error
		GetBook(ctx context.Context, idBook string) (entity.Book, error)
		GetAuthorBooks(ctx context.Context, idAuthor string) (<-chan entity.Book, error)
	}

	OutboxRepository interface {
		SendMessage(ctx context.Context, idempotencyKey string, kind OutboxKind, message []byte) error
		GetMessages(ctx context.Context, batchSize int, inProgressTTL time.Duration) ([]OutboxData, error)
		MarkAs(ctx context.Context, idempotencyKeys []string, s Status) error
	}

	OutboxData struct {
		IdempotencyKey string
		Kind           OutboxKind
		RawData        []byte
	}

	Transactor interface {
		WithTx(ctx context.Context, function func(ctx context.Context) error) error
	}
)

type OutboxKind int

const (
	OutboxKindUndefined OutboxKind = iota
	OutboxKindAuthor
	OutboxKindBook
)

func (o OutboxKind) String() string {
	switch o {
	case OutboxKindAuthor:
		return "author"
	case OutboxKindBook:
		return "book"
	default:
		return "undefined"
	}
}
