package library

import (
	"context"

	"github.com/project/library/generated/api/library"
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
