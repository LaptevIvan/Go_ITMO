package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

const attemptsRetry = 1

func Test_outboxRepository_SendMessage(t *testing.T) {
	t.Parallel()
	type args struct {
		idempotencyKey string
		kind           OutboxKind
		message        []byte
	}
	tests := []struct {
		name       string
		args       args
		txL        txLayer
		errRequire error
	}{
		{
			name: "ok with transaction",
			args: args{
				idempotencyKey: "",
				kind:           OutboxKindUndefined,
				message:        nil,
			},
			txL:        extract,
			errRequire: nil,
		},

		{
			name: "ok without transaction",
			args: args{
				idempotencyKey: "",
				kind:           OutboxKindUndefined,
				message:        nil,
			},
			txL:        none,
			errRequire: nil,
		},

		{
			name: "err in transaction",
			args: args{
				idempotencyKey: "",
				kind:           OutboxKindUndefined,
				message:        nil,
			},
			txL:        extract,
			errRequire: errInternal,
		},

		{
			name: "err in exec",
			args: args{
				idempotencyKey: "",
				kind:           OutboxKindUndefined,
				message:        nil,
			},
			txL:        none,
			errRequire: errInternal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			ctx := context.Background()

			tID := tt.args.idempotencyKey
			tMes := tt.args.message
			tKind := tt.args.kind
			tErr := tt.errRequire

			if tt.txL == extract {
				ctx = insertTxInMock(ctx, mock)
			}
			expected := mock.ExpectExec(`INSERT INTO outbox`).WithArgs(tID, tMes, tKind)
			if tErr != nil {
				expected.WillReturnError(tErr)
			} else {
				expected.WillReturnResult(pgxmock.NewResult("INSERT", 1))
			}

			o := NewOutbox(mock, attemptsRetry)
			err = o.SendMessage(ctx, tID, tKind, tMes)
			require.Equal(t, tErr, err)
		})
	}
}

func Test_outboxRepository_GetMessages(t *testing.T) {
	t.Parallel()

	type args struct {
		batchSize     int
		inProgressTTL time.Duration
	}

	const testKind = "testKind"
	testData := ([]byte)("testData")

	tests := []struct {
		name       string
		args       args
		want       []OutboxData
		txL        txLayer
		errL       errLayer
		errRequire error
	}{
		{
			name: "ok with transaction",
			args: args{
				batchSize:     3,
				inProgressTTL: time.Second,
			},
			want: []OutboxData{
				{
					IdempotencyKey: testKind,
					Kind:           OutboxKindUndefined,
					RawData:        testData,
				},
			},
			txL:        extract,
			errL:       null,
			errRequire: nil,
		},

		{
			name: "ok without transaction",
			args: args{
				batchSize:     3,
				inProgressTTL: time.Second,
			},
			want: []OutboxData{
				{
					IdempotencyKey: testKind,
					Kind:           OutboxKindUndefined,
					RawData:        testData,
				},
			},
			txL:        none,
			errL:       null,
			errRequire: nil,
		},

		{
			name: "error with transaction",
			args: args{
				batchSize:     3,
				inProgressTTL: time.Second,
			},
			want:       nil,
			txL:        extract,
			errL:       db,
			errRequire: errInternal,
		},

		{
			name: "error without transaction",
			args: args{
				batchSize:     3,
				inProgressTTL: time.Second,
			},
			want:       nil,
			txL:        none,
			errL:       db,
			errRequire: errInternal,
		},

		{
			name: "error during scanning",
			args: args{
				batchSize:     3,
				inProgressTTL: time.Second,
			},
			want:       nil,
			txL:        extract,
			errL:       scan,
			errRequire: errInternal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			ctx := context.Background()
			tBSize := tt.args.batchSize
			tTTL := tt.args.inProgressTTL
			tWant := tt.want
			tErrL := tt.errL
			tErr := tt.errRequire
			interval := fmt.Sprintf("%d ms", tTTL.Milliseconds())

			if tt.txL == extract {
				ctx = insertTxInMock(ctx, mock)
			}
			expected := mock.ExpectQuery(`UPDATE outbox`).WithArgs(interval, tBSize)
			if tErrL == db {
				expected.WillReturnError(tErr)
			} else {
				rows := pgxmock.NewRows([]string{"idempotency_key", "data", "kind"})
				if tErrL == scan {
					rows.AddRow(-1, -1, -1)
				} else {
					for _, el := range tWant {
						rows.AddRow(el.IdempotencyKey, el.RawData, el.Kind)
					}
				}
				expected.WillReturnRows(rows)
			}

			o := NewOutbox(mock, attemptsRetry)
			data, err := o.GetMessages(ctx, tBSize, tTTL)
			require.Equal(t, tWant, data)
			if tErrL == scan {
				require.Error(t, err)
				return
			}
			require.Equal(t, tErr, err)
		})
	}
}

func Test_outboxRepository_MarkAsProcessed(t *testing.T) {
	t.Parallel()

	keys := []string{"1", "2", "3"}

	tests := []struct {
		name            string
		txL             txLayer
		idempotencyKeys []string
		errRequire      error
	}{
		{
			name:            "test success with tx",
			txL:             extract,
			idempotencyKeys: keys,
			errRequire:      nil,
		},

		{
			name:            "test success without tx",
			txL:             none,
			idempotencyKeys: keys,
			errRequire:      nil,
		},

		{
			name:            "err with tx",
			txL:             extract,
			idempotencyKeys: keys,
			errRequire:      errInternal,
		},

		{
			name:            "err with tx",
			txL:             none,
			idempotencyKeys: keys,
			errRequire:      errInternal,
		},

		{
			name:            "empty idempotencyKeys",
			txL:             none,
			idempotencyKeys: []string{},
			errRequire:      nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			ctx := context.Background()

			tKeys := tt.idempotencyKeys
			tErr := tt.errRequire

			if tt.txL == extract {
				ctx = insertTxInMock(ctx, mock)
			}
			expected := mock.ExpectExec(`UPDATE outbox`).WithArgs(Success.String(), tKeys, attemptsRetry)
			if tErr != nil {
				expected.WillReturnError(tErr)
			} else {
				expected.WillReturnResult(pgxmock.NewResult("UPDATE", int64(len(tKeys))))
			}

			o := NewOutbox(mock, attemptsRetry)
			err = o.MarkAs(ctx, tKeys, Success)
			require.Equal(t, tErr, err)
		})
	}
}
