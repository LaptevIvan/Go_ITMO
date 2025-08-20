package library

import (
	"context"
	"encoding/json"
	"github.com/project/library/internal/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/project/library/generated/api/library"
	"github.com/project/library/internal/entity"
	"github.com/project/library/internal/usecase/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertBook(book *entity.Book) *library.Book {
	return &library.Book{
		Id:        book.ID,
		Name:      book.Name,
		AuthorId:  book.AuthorIDs,
		CreatedAt: timestamppb.New(book.CreatedAt),
		UpdatedAt: timestamppb.New(book.UpdatedAt),
	}
}

func (l *libraryImpl) AddBook(ctx context.Context, name string, authorIDs []string) (*library.AddBookResponse, error) {
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()
	log.InfoAddBook(l.logger, "start of adding book", traceID, name, authorIDs)

	var book entity.Book
	err := l.transactor.WithTx(ctx, func(ctx context.Context) error {
		var txErr error
		book, txErr = l.booksRepository.AddBook(ctx, entity.Book{
			Name:      name,
			AuthorIDs: authorIDs,
		})

		if txErr != nil {
			return txErr
		}

		serialized, txErr := json.Marshal(book)

		if txErr != nil {
			return txErr
		}

		idempotencyKey := repository.OutboxKindBook.String() + "_" + book.ID
		txErr = l.outboxRepository.SendMessage(ctx, idempotencyKey, repository.OutboxKindBook, serialized)

		if txErr != nil {
			return txErr
		}

		return nil
	})

	if log.ErrorAddBook(l.logger, err, "Failed adding book", traceID, name, authorIDs) {
		span.SetAttributes(attribute.String("book_name", name))
		span.SetAttributes(attribute.StringSlice("book_authors", authorIDs))
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(attribute.String("book_id", book.ID))
	log.InfoAddBook(l.logger, "book was added", traceID, name, authorIDs, book.ID)

	return &library.AddBookResponse{
		Book: convertBook(&book),
	}, nil
}

func (l *libraryImpl) GetBookInfo(ctx context.Context, bookID string) (*library.GetBookInfoResponse, error) {
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()
	span.SetAttributes(attribute.String("book_id", bookID))
	log.InfoGetBookInfo(l.logger, "start of getting book info", traceID, bookID)

	book, err := l.booksRepository.GetBook(ctx, bookID)

	if log.ErrorGetBookInfo(l.logger, err, "Failed get book info", traceID, bookID) {
		span.RecordError(err)
		return nil, err
	}

	log.InfoGetBookInfo(l.logger, "got info of book", traceID, bookID)
	return &library.GetBookInfoResponse{
		Book: convertBook(&book),
	}, nil
}

func (l *libraryImpl) UpdateBook(ctx context.Context, id, newName string, newAuthorIDs []string) error {
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()
	span.SetAttributes(attribute.String("book_id", id))
	log.InfoUpdateBook(l.logger, "start of updating book info", traceID, id, newName, newAuthorIDs)

	err := l.booksRepository.UpdateBook(ctx, entity.Book{
		ID:        id,
		Name:      newName,
		AuthorIDs: newAuthorIDs,
	})

	if log.ErrorUpdateBook(l.logger, err, "Failed update book", traceID, id, newName, newAuthorIDs) {
		span.RecordError(err)
	} else {
		log.InfoUpdateBook(l.logger, "Updated the book", traceID, id, newName, newAuthorIDs)
	}

	return err
}

func (l *libraryImpl) GetAuthorBooks(ctx context.Context, idAuthor string) (<-chan *library.Book, error) {
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()
	span.SetAttributes(attribute.String("author_id", idAuthor))
	log.InfoGetAuthorBooks(l.logger, "start of getting author books", traceID, idAuthor)

	books, err := l.booksRepository.GetAuthorBooks(ctx, idAuthor)

	if log.ErrorGetAuthorBooks(l.logger, err, "Failed get author books", traceID, idAuthor) {
		span.RecordError(err)
		return nil, err
	}
	log.InfoGetAuthorBooks(l.logger, "Got the author's book", traceID, idAuthor)

	ans := make(chan *library.Book)
	go func() {
		defer close(ans)
		for b := range books {
			ans <- convertBook(&b)
		}
	}()

	return ans, err
}
