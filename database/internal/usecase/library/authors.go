package library

import (
	"context"
	"encoding/json"
	"github.com/project/library/internal/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/project/library/generated/api/library"
	"github.com/project/library/internal/usecase/repository"

	"github.com/project/library/internal/entity"
)

func (l *libraryImpl) RegisterAuthor(ctx context.Context, authorName string) (*library.RegisterAuthorResponse, error) {
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()
	log.InfoRegisterAuthor(l.logger, "Start of register author", traceID, authorName)

	var author entity.Author
	err := l.transactor.WithTx(ctx, func(ctx context.Context) error {
		var txErr error
		author, txErr = l.authorRepository.RegisterAuthor(ctx, entity.Author{
			Name: authorName,
		})

		if txErr != nil {
			return txErr
		}

		serialized, txErr := json.Marshal(author)

		if txErr != nil {
			return txErr
		}

		idempotencyKey := repository.OutboxKindAuthor.String() + "_" + author.ID
		txErr = l.outboxRepository.SendMessage(ctx, idempotencyKey, repository.OutboxKindAuthor, serialized)

		if txErr != nil {
			return txErr
		}

		return nil
	})

	if log.ErrorRegisterAuthor(l.logger, err, "Failed register author", traceID, authorName) {
		span.SetAttributes(attribute.String("author_name", authorName))
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(attribute.String("author_id", author.ID))
	log.InfoRegisterAuthor(l.logger, "Registered the author", traceID, authorName, author.ID)
	return &library.RegisterAuthorResponse{
		Id: author.ID,
	}, nil
}

func (l *libraryImpl) ChangeAuthorInfo(ctx context.Context, idAuthor, newName string) error {
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()
	span.SetAttributes(attribute.String("author_id", idAuthor))
	log.InfoChangeAuthorInfo(l.logger, "Start of change author info", traceID, idAuthor, newName)

	err := l.authorRepository.ChangeAuthorInfo(ctx, entity.Author{
		ID:   idAuthor,
		Name: newName,
	})

	if log.ErrorChangeAuthorInfo(l.logger, err, "Failed changing author", traceID, idAuthor, newName) {
		span.RecordError(err)
	} else {
		log.InfoChangeAuthorInfo(l.logger, "Changed the author with id", traceID, idAuthor, newName)
	}

	return err
}

func (l *libraryImpl) GetAuthorInfo(ctx context.Context, idAuthor string) (*library.GetAuthorInfoResponse, error) {
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()
	span.SetAttributes(attribute.String("author_id", idAuthor))
	log.InfoGetAuthorInfo(l.logger, "start of getting author info", traceID, idAuthor)

	author, err := l.authorRepository.GetAuthorInfo(ctx, idAuthor)

	if log.ErrorGetAuthorInfo(l.logger, err, "Failed get author info", traceID, idAuthor) {
		span.RecordError(err)
		return nil, err
	} else {
		log.InfoGetAuthorInfo(l.logger, "Got the author info", traceID, idAuthor)
	}

	return &library.GetAuthorInfoResponse{
		Id:   author.ID,
		Name: author.Name,
	}, err
}
