package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/project/library/pkg/logger"
	"go.uber.org/zap"
)

type GetterTx interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

var _ Transactor = (*transactorImpl)(nil)

type transactorImpl struct {
	logger *zap.Logger
	db     GetterTx
}

func NewTransactor(logger *zap.Logger, db GetterTx) *transactorImpl {
	return &transactorImpl{
		logger: logger,
		db:     db,
	}
}
func (t *transactorImpl) WithTx(ctx context.Context, function func(ctx context.Context) error) (txErr error) {
	ctxWithTx, tx, err := injectTx(ctx, t.db)

	if err != nil {
		return fmt.Errorf("can not inject transaction, error: %w", err)
	}

	defer func() {
		if txErr != nil {
			err = tx.Rollback(ctxWithTx)
			logger.CheckError(err, t.logger, "failed Rollback of tx", zap.Error(err))
			return
		}

		err = tx.Commit(ctxWithTx)
		logger.CheckError(err, t.logger, "failed commit of tx", zap.Error(err))
	}()

	err = function(ctxWithTx)

	if err != nil {
		return fmt.Errorf("function execution error: %w", err)
	}

	return nil
}

type txInjector struct{}

var ErrTxNotFound = errors.New("tx not found in context")

func injectTx(ctx context.Context, pool GetterTx) (context.Context, pgx.Tx, error) {
	tx, err := pool.Begin(ctx)

	if err != nil {
		return nil, nil, err
	}

	return context.WithValue(ctx, txInjector{}, tx), tx, nil
}

func extractTx(ctx context.Context) (pgx.Tx, error) {
	tx, ok := ctx.Value(txInjector{}).(pgx.Tx)

	if !ok {
		return nil, ErrTxNotFound
	}

	return tx, nil
}
