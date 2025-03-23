package library

import (
	"context"

	"github.com/project/library/internal/entity"
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
		GetAuthorBooks(ctx context.Context, idAuthor string) ([]entity.Book, error)
	}
)
