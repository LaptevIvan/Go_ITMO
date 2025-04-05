package library

import (
	"context"

	"github.com/project/library/internal/entity"
	"go.uber.org/zap"
)

func (l *libraryImpl) AddBook(ctx context.Context, name string, authorIDs []string) (entity.Book, error) {
	book, err := l.booksRepository.AddBook(ctx, entity.Book{
		Name:      name,
		AuthorIDs: authorIDs,
	})

	if err != nil {
		if log {
			l.logger.Error("Failed added book", zap.Error(err))
		}
		return entity.Book{}, err
	}
	if log {
		l.logger.Info("Added the book", zap.String("id of book", book.ID))
	}

	return book, nil
}

func (l *libraryImpl) GetBookInfo(ctx context.Context, bookID string) (entity.Book, error) {
	book, err := l.booksRepository.GetBook(ctx, bookID)
	if err != nil {
		if log {
			l.logger.Error("Failed get book info", zap.String("id of book", bookID), zap.Error(err))
		}
		return entity.Book{}, err
	}

	if log {
		l.logger.Info("Get the book", zap.String("id of book", bookID))
	}
	return book, nil
}

func (l *libraryImpl) UpdateBook(ctx context.Context, id, newName string, newAuthorIDs []string) error {
	if log {
		l.logger.Info("Updating the book with id", zap.String("id of book", id))
	}

	err := l.booksRepository.UpdateBook(ctx, entity.Book{
		ID:        id,
		Name:      newName,
		AuthorIDs: newAuthorIDs,
	})
	if err != nil && log {
		l.logger.Error("Failed update book", zap.Error(err))
	}
	return err
}

func (l *libraryImpl) GetAuthorBooks(ctx context.Context, idAuthor string) (<-chan entity.Book, error) {
	books, err := l.booksRepository.GetAuthorBooks(ctx, idAuthor)
	if err != nil {
		if log {
			l.logger.Error("Failed get author books", zap.Error(err))
		}
		return nil, err
	}

	if log {
		l.logger.Info("Got the author's book", zap.String("author's id", idAuthor))
	}
	return books, err
}
