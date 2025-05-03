package repository

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

var fNoError = func(ctx context.Context) error {
	return nil
}

var fError = func(ctx context.Context) error {
	return errInternal
}

func Test_transactorImpl_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		f          func(ctx context.Context) error
		errL       errLayer
		errRequire error
	}{
		{
			name:       "ok create new tx",
			f:          fNoError,
			errL:       null,
			errRequire: nil,
		},

		{
			name:       "error in function",
			f:          fError,
			errL:       f,
			errRequire: errInternal,
		},

		{
			name:       "error in begin transaction",
			f:          nil,
			errL:       beginTx,
			errRequire: errInternal,
		},

		{
			name:       "error in commit transaction",
			f:          fNoError,
			errL:       commitTx,
			errRequire: nil,
		},

		{
			name:       "error in RollBack transaction",
			f:          fError,
			errL:       rollBackTx,
			errRequire: errInternal,
		},
	}
	logger, e := zap.NewProduction()
	require.NoError(t, e)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)

			ctx := context.Background()
			tErr := tt.errRequire
			tErrL := tt.errL

			begin := mock.ExpectBegin()
			if tErrL == beginTx {
				begin.WillReturnError(tErr)
			}

			if tErr != nil {
				expectRollBack := mock.ExpectRollback()
				if tErrL == rollBackTx {
					expectRollBack.WillReturnError(errInternal)
				}
			} else {
				expectCommit := mock.ExpectCommit()
				if tErrL == commitTx {
					expectCommit.WillReturnError(errInternal)
				}
			}
			transactor := NewTransactor(logger, mock)

			err = transactor.WithTx(ctx, tt.f)
			require.ErrorIs(t, err, tErr)
		})
	}
}

func Test_extractTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		errRequire error
	}{
		{
			name:       "ok extract",
			errRequire: nil,
		},

		{
			name:       "extract with failure",
			errRequire: ErrTxNotFound,
		},
	}
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tErr := tt.errRequire
			ctx := context.Background()
			if tErr == nil {
				mock.ExpectBegin()
				tx, _ := mock.Begin(ctx)
				ctx = context.WithValue(ctx, txInjector{}, tx)
			}

			tx, e := extractTx(ctx)
			require.ErrorIs(t, e, tErr)
			if e != nil {
				require.Nil(t, tx)
			}
		})
	}
}
