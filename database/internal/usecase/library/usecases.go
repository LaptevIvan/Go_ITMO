package library

import (
	"context"
	"time"

	"github.com/project/library/internal/usecase/repository"

	"github.com/project/library/internal/entity"
	"go.uber.org/zap"
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
		SendMessage(ctx context.Context, idempotencyKey string, kind repository.OutboxKind, message []byte) error
		GetMessages(ctx context.Context, batchSize int, inProgressTTL time.Duration) ([]repository.OutboxData, error)
		MarkAs(ctx context.Context, idempotencyKeys []string, s repository.Status) error
	}

	Transactor interface {
		WithTx(ctx context.Context, function func(ctx context.Context) error) error
	}
)

var _ AuthorUseCase = (*libraryImpl)(nil)
var _ BooksUseCase = (*libraryImpl)(nil)

type libraryImpl struct {
	logger           *zap.Logger
	authorRepository AuthorRepository
	booksRepository  BooksRepository
	outboxRepository OutboxRepository
	transactor       repository.Transactor
}

func New(
	logger *zap.Logger,
	authorRepository AuthorRepository,
	booksRepository BooksRepository,
	outboxRepository OutboxRepository,
	transactor Transactor,
) *libraryImpl {
	return &libraryImpl{
		logger:           logger,
		authorRepository: authorRepository,
		booksRepository:  booksRepository,
		outboxRepository: outboxRepository,
		transactor:       transactor,
	}
}
