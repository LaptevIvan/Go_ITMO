package controller

import (
	"context"
	"github.com/project/library/internal/log"
	"go.opentelemetry.io/otel/attribute"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"

	"github.com/project/library/generated/api/library"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var GetBookInfoDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "library_get_book_info_duration_ms",
	Help:    "Duration of GetBookInfo in ms",
	Buckets: prometheus.DefBuckets,
})

func init() {
	prometheus.MustRegister(GetBookInfoDuration)
}

func (i *implementation) GetBookInfo(ctx context.Context, req *library.GetBookInfoRequest) (*library.GetBookInfoResponse, error) {
	start := time.Now()

	defer func() {
		GetBookInfoDuration.Observe(float64(time.Since(start).Milliseconds()))
	}()

	span := trace.SpanFromContext(ctx)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	if err := req.ValidateAll(); log.ErrorGetBookInfo(i.logger, err, "Got invalid request", traceID, req.GetId()) {
		span.SetAttributes(attribute.String("book_id", req.GetId()))
		span.RecordError(err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	book, err := i.booksUseCase.GetBookInfo(ctx, req.GetId())

	if err != nil {
		return nil, i.convertErr(err)
	}

	return book, nil
}
