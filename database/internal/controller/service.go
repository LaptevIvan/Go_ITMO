package controller

import (
	"context"

	"github.com/project/library/generated/api/library"
	generated "github.com/project/library/generated/api/library"
	"go.uber.org/zap"
)

type (
	AuthorUseCase interface {
		RegisterAuthor(ctx context.Context, authorName string) (*library.RegisterAuthorResponse, error)
		ChangeAuthorInfo(ctx context.Context, idAuthor, newName string) error
		GetAuthorInfo(ctx context.Context, idAuthor string) (*library.GetAuthorInfoResponse, error)
	}

	BooksUseCase interface {
		AddBook(ctx context.Context, name string, authorIDs []string) (*library.AddBookResponse, error)
		GetBookInfo(ctx context.Context, bookID string) (*library.GetBookInfoResponse, error)
		UpdateBook(ctx context.Context, id, newName string, newAuthorIDs []string) error
		GetAuthorBooks(ctx context.Context, idAuthor string) (<-chan *library.Book, error)
	}
)

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
