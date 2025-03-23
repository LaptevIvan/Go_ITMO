package repository

import (
	"context"

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
		GetAuthorBooks(ctx context.Context, idAuthor string) ([]entity.Book, error)
	}
)
