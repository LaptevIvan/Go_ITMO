package controller

import (
	"context"

	generated "github.com/project/library/generated/api/library"
	"github.com/project/library/internal/entity"
	"go.uber.org/zap"
)

type (
	AuthorUseCase interface {
		RegisterAuthor(ctx context.Context, authorName string) (entity.Author, error)
		ChangeAuthorInfo(ctx context.Context, idAuthor, newName string) error
		GetAuthorInfo(ctx context.Context, idAuthor string) (entity.Author, error)
	}

	BooksUseCase interface {
		AddBook(ctx context.Context, name string, authorIDs []string) (entity.Book, error)
		GetBookInfo(ctx context.Context, bookID string) (entity.Book, error)
		UpdateBook(ctx context.Context, id, newName string, newAuthorIDs []string) error
		GetAuthorBooks(ctx context.Context, idAuthor string) (<-chan entity.Book, error)
	}
)

const log = true

var _ generated.LibraryServer = (*implementation)(nil)

type implementation struct {
	logger        *zap.Logger
	booksUseCase  BooksUseCase
	authorUseCase AuthorUseCase
}

func New(
	logger *zap.Logger,
	booksUseCase BooksUseCase,
	authorUseCase AuthorUseCase,
) *implementation {
	return &implementation{
		logger:        logger,
		booksUseCase:  booksUseCase,
		authorUseCase: authorUseCase,
	}
}
