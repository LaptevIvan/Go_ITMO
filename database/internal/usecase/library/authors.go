package library

import (
	"context"
	"encoding/json"

	"github.com/project/library/pkg/logger"

	"github.com/project/library/generated/api/library"
	"github.com/project/library/internal/usecase/repository"

	"github.com/project/library/internal/entity"
	"go.uber.org/zap"
)

func (l *libraryImpl) RegisterAuthor(ctx context.Context, authorName string) (*library.RegisterAuthorResponse, error) {
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

	if logger.CheckError(err, l.logger, "Failed register author", zap.Error(err)) {
		return nil, err
	}
	if l.logger != nil {
		l.logger.Info("Registered the author", zap.String("author's id", author.ID))
	}

	return &library.RegisterAuthorResponse{
		Id: author.ID,
	}, nil
}

func (l *libraryImpl) ChangeAuthorInfo(ctx context.Context, idAuthor, newName string) error {
	err := l.authorRepository.ChangeAuthorInfo(ctx, entity.Author{
		ID:   idAuthor,
		Name: newName,
	})

	if !logger.CheckError(err, l.logger, "Failed changing author", zap.Error(err)) {
		if l.logger != nil {
			l.logger.Info("Changed the author with id", zap.String("id of author", idAuthor))
		}
	}
	return err
}

func (l *libraryImpl) GetAuthorInfo(ctx context.Context, idAuthor string) (*library.GetAuthorInfoResponse, error) {
	author, err := l.authorRepository.GetAuthorInfo(ctx, idAuthor)

	if logger.CheckError(err, l.logger, "Failed get author info", zap.String("author id", idAuthor), zap.Error(err)) {
		return nil, err
	}
	if l.logger != nil {
		l.logger.Info("Get the author info", zap.String("author id", idAuthor))
	}

	return &library.GetAuthorInfoResponse{
		Id:   author.ID,
		Name: author.Name,
	}, err
}
