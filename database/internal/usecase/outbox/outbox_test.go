package outbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/project/library/config"
	"github.com/project/library/internal/usecase/library/mocks"
	mocks2 "github.com/project/library/internal/usecase/outbox/mocks"
	"github.com/project/library/internal/usecase/repository"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

type errLayer uint

const (
	none errLayer = iota
	transactor
	getMessage
	globalHandler
	kindHandler
	markAsSuccess
	markAsCreated
)

var errInternal = errors.New("Internal")

var testGlobalHandler = func(kind repository.OutboxKind) (KindHandler, error) {
	switch kind {
	case repository.OutboxKindUndefined:
		return nil, errInternal
	case repository.OutboxKindAuthor:
		return testAuthorHandler, nil
	case repository.OutboxKindBook:
		return testBookHandler, nil
	default:
		panic("unreachable")
	}
}

func testAuthorHandler(_ context.Context, _ []byte) error {
	return nil
}

func testBookHandler(_ context.Context, _ []byte) error {
	return errInternal
}

func Test_outboxImpl_worker(t *testing.T) {
	t.Parallel()

	type args struct {
		wg            *sync.WaitGroup
		batchSize     int
		waitTime      time.Duration
		inProgressTTL time.Duration
	}
	standartArgs := args{
		wg:            new(sync.WaitGroup),
		batchSize:     1,
		waitTime:      time.Nanosecond,
		inProgressTTL: time.Second,
	}

	tests := []struct {
		name                   string
		args                   args
		errL                   errLayer
		enabled                bool
		transactorCount        int
		outboxGetCount         int
		outboxMarkSuccessCount int
		outboxMarkCreatedCount int
		ctxCall                int
	}{
		{
			name:                   "ok iteration",
			args:                   standartArgs,
			errL:                   none,
			enabled:                true,
			transactorCount:        1,
			outboxGetCount:         1,
			outboxMarkSuccessCount: 1,
			outboxMarkCreatedCount: 1,
			ctxCall:                3,
		},

		{
			name:                   "iteration with false enabled",
			args:                   standartArgs,
			errL:                   none,
			enabled:                false,
			transactorCount:        0,
			outboxGetCount:         0,
			outboxMarkSuccessCount: 0,
			outboxMarkCreatedCount: 0,
			ctxCall:                3,
		},

		{
			name:                   "transactor err",
			args:                   standartArgs,
			errL:                   transactor,
			enabled:                true,
			transactorCount:        1,
			outboxGetCount:         0,
			outboxMarkSuccessCount: 0,
			outboxMarkCreatedCount: 0,
			ctxCall:                3,
		},

		{
			name:                   "GetMessages err",
			args:                   standartArgs,
			errL:                   getMessage,
			enabled:                true,
			transactorCount:        1,
			outboxGetCount:         1,
			outboxMarkSuccessCount: 0,
			outboxMarkCreatedCount: 0,
			ctxCall:                3,
		},

		{
			name:                   "GlobalHandler err",
			args:                   standartArgs,
			errL:                   globalHandler,
			enabled:                true,
			transactorCount:        1,
			outboxGetCount:         1,
			outboxMarkSuccessCount: 1,
			outboxMarkCreatedCount: 1,
			ctxCall:                3,
		},

		{
			name:                   "KindHandler err",
			args:                   standartArgs,
			errL:                   kindHandler,
			enabled:                true,
			transactorCount:        1,
			outboxGetCount:         1,
			outboxMarkSuccessCount: 1,
			outboxMarkCreatedCount: 1,
			ctxCall:                3,
		},

		{
			name:                   "MarkAs 'SUCCESS' err",
			args:                   standartArgs,
			errL:                   markAsSuccess,
			enabled:                true,
			transactorCount:        1,
			outboxGetCount:         1,
			outboxMarkSuccessCount: 1,
			outboxMarkCreatedCount: 0,
			ctxCall:                3,
		},

		{
			name:                   "MarkAs 'CREATED' err",
			args:                   standartArgs,
			errL:                   markAsCreated,
			enabled:                true,
			transactorCount:        1,
			outboxGetCount:         1,
			outboxMarkSuccessCount: 1,
			outboxMarkCreatedCount: 1,
			ctxCall:                3,
		},

		{
			name:                   "Fact ctx Done",
			args:                   standartArgs,
			errL:                   none,
			enabled:                true,
			transactorCount:        0,
			outboxGetCount:         0,
			outboxMarkSuccessCount: 0,
			outboxMarkCreatedCount: 0,
			ctxCall:                1,
		},

		{
			name:                   "ctx Done after wait",
			args:                   standartArgs,
			errL:                   none,
			enabled:                true,
			transactorCount:        0,
			outboxGetCount:         0,
			outboxMarkSuccessCount: 0,
			outboxMarkCreatedCount: 0,
			ctxCall:                2,
		},
	}
	logger, e := zap.NewProduction()
	require.NoError(t, e)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tErr := tt.errL
			ctrl := gomock.NewController(t)

			outboxRepo := mocks.NewMockOutboxRepository(ctrl)
			ctx := mocks2.NewMockContext(ctrl)
			tt.args.wg.Add(1)

			inc := 0
			done := make(chan struct{})
			tCtxCall := tt.ctxCall
			ctx.EXPECT().Done().DoAndReturn(func() <-chan struct{} {
				if inc == tCtxCall-1 {
					close(done)
				}
				inc++
				return done
			}).Times(tCtxCall)

			outboxRepo.EXPECT().GetMessages(ctx, tt.args.batchSize, tt.args.inProgressTTL).DoAndReturn(func(ctx context.Context, batchSize int, inProgressTTL time.Duration) ([]repository.OutboxData, error) {
				switch tErr {
				case getMessage:
					return nil, errInternal
				case kindHandler:
					return []repository.OutboxData{{
						Kind: repository.OutboxKindBook,
					}}, nil
				case globalHandler:
					return []repository.OutboxData{{
						Kind: repository.OutboxKindUndefined,
					}}, nil
				default:
					return []repository.OutboxData{{
						Kind: repository.OutboxKindAuthor,
					}}, nil
				}
			}).Times(tt.outboxGetCount)

			outboxRepo.EXPECT().MarkAs(ctx, gomock.Any(), repository.Success).DoAndReturn(func(ctx context.Context, idempotencyKeys []string, s repository.Status) error {
				if tErr == markAsSuccess {
					return errInternal
				}
				return nil
			}).Times(tt.outboxMarkSuccessCount)

			outboxRepo.EXPECT().MarkAs(ctx, gomock.Any(), repository.Created).DoAndReturn(func(ctx context.Context, idempotencyKeys []string, s repository.Status) error {
				if tErr == markAsCreated {
					return errInternal
				}
				return nil
			}).Times(tt.outboxMarkCreatedCount)

			tr := mocks2.NewMockTransactor(ctrl)
			tr.EXPECT().WithTx(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, f func(ctx context.Context) error) error {
				if tErr == transactor {
					return errInternal
				}
				return f(ctx)
			}).Times(tt.transactorCount)

			cfg := &config.Config{
				Outbox: struct {
					Enabled         bool          `env:"OUTBOX_ENABLED"`
					Workers         int           `env:"OUTBOX_WORKERS"`
					BatchSize       int           `env:"OUTBOX_BATCH_SIZE"`
					WaitTimeMS      time.Duration `env:"OUTBOX_WAIT_TIME_MS"`
					InProgressTTLMS time.Duration `env:"OUTBOX_IN_PROGRESS_TTL_MS"`
					AuthorSendURL   string        `env:"OUTBOX_AUTHOR_SEND_URL"`
					BookSendURL     string        `env:"OUTBOX_BOOK_SEND_URL"`
					AttemptsRetry   int           `env:"OUTBOX_ATTEMPTS_RETRY"`
				}{Enabled: tt.enabled},
			}

			o := &outboxImpl{
				logger:           logger,
				outboxRepository: outboxRepo,
				globalHandler:    testGlobalHandler,
				cfg:              cfg,
				transactor:       tr,
			}
			o.worker(ctx, tt.args.wg, tt.args.batchSize, tt.args.waitTime, tt.args.inProgressTTL)
		})
	}
}
