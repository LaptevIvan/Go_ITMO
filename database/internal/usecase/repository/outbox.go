package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"time"

	"github.com/jackc/pgx/v5"
)

type Status uint

const (
	Created Status = iota
	InProgress
	Success
	Abandoned
)

func (s Status) String() string {
	switch s {
	case Created:
		return "CREATED"
	case InProgress:
		return "IN_PROGRESS"
	case Success:
		return "SUCCESS"
	case Abandoned:
		return "ABANDONED"
	}
	panic("unreachable")
}

type DataBase interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

var _ OutboxRepository = (*outboxRepository)(nil)

type outboxRepository struct {
	db            DataBase
	attemptsRetry int
}

func NewOutbox(db DataBase, attemptsRetry int) *outboxRepository {
	return &outboxRepository{
		db:            db,
		attemptsRetry: attemptsRetry,
	}
}

func (o *outboxRepository) SendMessage(ctx context.Context, idempotencyKey string, kind OutboxKind, message []byte) error {
	const query = `
INSERT INTO outbox (idempotency_key, data, status, kind, attempts)
VALUES($1, $2, 'CREATED', $3, 0)
ON CONFLICT (idempotency_key) DO NOTHING`

	var err error
	if tx, txErr := extractTx(ctx); txErr == nil {
		_, err = tx.Exec(ctx, query, idempotencyKey, message, kind)
	} else {
		_, err = o.db.Exec(ctx, query, idempotencyKey, message, kind)
	}

	if err != nil {
		return err
	}

	return nil
}

// status == CREATED || (status == IN_PROGRESS && time.Now() - updated_at > TTL)
func (o *outboxRepository) GetMessages(ctx context.Context, batchSize int, inProgressTTL time.Duration) ([]OutboxData, error) {
	const query = `
UPDATE outbox
SET status = 'IN_PROGRESS'
WHERE idempotency_key IN (
    SELECT idempotency_key
    FROM outbox
    WHERE
        (status = 'CREATED'
            OR (status = 'IN_PROGRESS' AND updated_at < now() - $1::interval))
    ORDER BY created_at
    LIMIT $2
    FOR UPDATE SKIP LOCKED
	)
	RETURNING idempotency_key, data, kind;`

	interval := fmt.Sprintf("%d ms", inProgressTTL.Milliseconds())

	var (
		err  error
		rows pgx.Rows
	)
	if tx, txErr := extractTx(ctx); txErr == nil {
		rows, err = tx.Query(ctx, query, interval, batchSize)
	} else {
		rows, err = o.db.Query(ctx, query, interval, batchSize)
	}

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make([]OutboxData, 0)

	for rows.Next() {
		var key string
		var rawData []byte
		var kind OutboxKind

		if err := rows.Scan(&key, &rawData, &kind); err != nil {
			return nil, err
		}

		result = append(result, OutboxData{
			IdempotencyKey: key,
			RawData:        rawData,
			Kind:           kind,
		})
	}

	return result, rows.Err()
}

func (o *outboxRepository) MarkAs(ctx context.Context, idempotencyKeys []string, s Status) error {
	if len(idempotencyKeys) == 0 {
		return nil
	}

	const query = `
UPDATE outbox
SET 
    status = CASE 
        WHEN status = 'IN_PROGRESS' 
        AND $1::outbox_status = 'CREATED' 
        AND attempts + 1 > $3 THEN 'ABANDONED'
        ELSE $1::outbox_status 
    END,
    attempts = CASE 
        WHEN status = 'IN_PROGRESS' 
        AND ($1::outbox_status = 'CREATED' 
        OR $1::outbox_status = 'SUCCESS') THEN attempts + 1 
        ELSE attempts 
    END
WHERE idempotency_key = ANY($2)
`

	var err error
	if tx, txErr := extractTx(ctx); txErr == nil {
		_, err = tx.Exec(ctx, query, s.String(), idempotencyKeys, o.attemptsRetry)
	} else {
		_, err = o.db.Exec(ctx, query, s.String(), idempotencyKeys, o.attemptsRetry)
	}

	if err != nil {
		return err
	}

	return nil
}
