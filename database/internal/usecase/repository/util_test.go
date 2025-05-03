package repository

import (
	"context"
	"errors"

	"github.com/pashagolub/pgxmock/v4"
)

type txLayer uint

const (
	none txLayer = iota
	extract
)

type errLayer uint

const (
	null errLayer = iota
	db
	scan
	f
	beginTx
	commitTx
	rollBackTx
)

var errInternal = errors.New("internal error")

func insertTxInMock(ctx context.Context, mock pgxmock.PgxPoolIface) context.Context {
	mock.ExpectBegin()
	tx, _ := mock.Begin(ctx)
	ctx = context.WithValue(ctx, txInjector{}, tx)
	return ctx
}
