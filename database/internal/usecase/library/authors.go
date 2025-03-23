package library

import (
	"context"

	"github.com/project/library/internal/entity"
	"go.uber.org/zap"
)

func (l *libraryImpl) RegisterAuthor(ctx context.Context, authorName string) (entity.Author, error) {
	author, err := l.authorRepository.RegisterAuthor(ctx, entity.Author{
		Name: authorName,
	})

	if err != nil {
		if log {
			l.logger.Error("Failed register author", zap.Error(err))
		}
		return entity.Author{}, err
	}

	if log {
		l.logger.Info("Registered the author", zap.String("author's id", author.ID))
	}
	return author, nil
}

func (l *libraryImpl) ChangeAuthorInfo(ctx context.Context, idAuthor, newName string) error {
	if log {
		l.logger.Info("Changing the author", zap.String("author's id", idAuthor))
	}
	return l.authorRepository.ChangeAuthorInfo(ctx, entity.Author{
		ID:   idAuthor,
		Name: newName,
	})
}

func (l *libraryImpl) GetAuthorInfo(ctx context.Context, idAuthor string) (entity.Author, error) {
	author, err := l.authorRepository.GetAuthorInfo(ctx, idAuthor)
	if err != nil {
		if log {
			l.logger.Error("Failed get author info", zap.String("author id", idAuthor), zap.Error(err))
		}
		return entity.Author{}, err
	}

	if log {
		l.logger.Info("Get the author info", zap.String("author id", idAuthor))
	}
	return author, err
}
