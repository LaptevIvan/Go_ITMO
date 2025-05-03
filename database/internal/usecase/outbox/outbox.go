package outbox

import (
	"context"
	"sync"
	"time"

	"github.com/project/library/pkg/logger"

	"github.com/project/library/config"
	"github.com/project/library/internal/usecase/repository"
	"go.uber.org/zap"
)

type (
	GlobalHandler = func(kind repository.OutboxKind) (KindHandler, error)
	KindHandler   = func(ctx context.Context, data []byte) error

	Repository interface {
		SendMessage(ctx context.Context, idempotencyKey string, kind repository.OutboxKind, message []byte) error
		GetMessages(ctx context.Context, batchSize int, inProgressTTL time.Duration) ([]repository.OutboxData, error)
		MarkAs(ctx context.Context, idempotencyKeys []string, s repository.Status) error
	}

	Transactor interface {
		WithTx(ctx context.Context, function func(ctx context.Context) error) error
	}
)

var _ Outbox = (*outboxImpl)(nil)

type outboxImpl struct {
	logger           *zap.Logger
	outboxRepository Repository
	globalHandler    GlobalHandler
	cfg              *config.Config
	transactor       Transactor
}

func New(
	logger *zap.Logger,
	outboxRepository Repository,
	globalHandler GlobalHandler,
	cfg *config.Config,
	transactor Transactor,
) *outboxImpl {
	return &outboxImpl{
		logger:           logger,
		outboxRepository: outboxRepository,
		globalHandler:    globalHandler,
		cfg:              cfg,
		transactor:       transactor,
	}
}

func (o *outboxImpl) Start(
	ctx context.Context,
	workers int,
	batchSize int,
	waitTime time.Duration,
	inProgressTTL time.Duration,
) {
	wg := new(sync.WaitGroup)

	for workerID := 1; workerID <= workers; workerID++ {
		wg.Add(1)
		go o.worker(ctx, wg, batchSize, waitTime, inProgressTTL)
	}
}

func (o *outboxImpl) worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	batchSize int,
	waitTime time.Duration,
	inProgressTTL time.Duration,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(waitTime)
			select {
			case <-ctx.Done():
				return
			default:
				if !o.cfg.Outbox.Enabled {
					continue
				}

				err := o.transactor.WithTx(ctx, func(ctx context.Context) error {
					messages, err := o.outboxRepository.GetMessages(ctx, batchSize, inProgressTTL)

					if logger.CheckError(err, o.logger, "can not fetch messages from outbox", zap.Error(err)) {
						return err
					}
					if o.logger != nil {
						o.logger.Info("messages fetched", zap.Int("size", len(messages)))
					}

					successKeys := make([]string, 0, len(messages))
					failKeys := make([]string, 0, len(messages))
					for i := 0; i < len(messages); i++ {
						message := messages[i]
						key := message.IdempotencyKey

						kindHandler, taskErr := o.globalHandler(message.Kind)

						if logger.CheckError(taskErr, o.logger, "unexpected kind", zap.Error(taskErr)) {
							failKeys = append(failKeys, key)
							continue
						}

						taskErr = kindHandler(ctx, message.RawData)

						if logger.CheckError(taskErr, o.logger, "kind error", zap.Error(taskErr)) {
							failKeys = append(failKeys, key)
							continue
						}

						successKeys = append(successKeys, key)
					}
					err = o.outboxRepository.MarkAs(ctx, successKeys, repository.Success)
					if logger.CheckError(err, o.logger, "Mark as 'Success' outbox error", zap.Error(err)) {
						return err
					}
					err = o.outboxRepository.MarkAs(ctx, failKeys, repository.Created)
					if logger.CheckError(err, o.logger, "Mark as 'Created' for fail task outbox error", zap.Error(err)) {
						return err
					}

					return nil
				})
				logger.CheckError(err, o.logger, "worker stage error", zap.Error(err))
			}
		}
	}
}
