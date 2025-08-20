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

var UpdateBookDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "library_update_book_duration_ms",
	Help:    "Duration of UpdateBook in ms",
	Buckets: prometheus.DefBuckets,
})

func init() {
	prometheus.MustRegister(UpdateBookDuration)
}

func (i *implementation) UpdateBook(ctx context.Context, req *library.UpdateBookRequest) (*library.UpdateBookResponse, error) {
	start := time.Now()

	defer func() {
		UpdateBookDuration.Observe(float64(time.Since(start).Milliseconds()))
	}()

	span := trace.SpanFromContext(ctx)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	if err := req.ValidateAll(); log.ErrorUpdateBook(i.logger, err, "Got invalid request", traceID, req.GetId(), req.GetName(), req.GetAuthorIds()) {
		span.SetAttributes(attribute.String("book_id", req.GetId()))
		span.RecordError(err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err := i.booksUseCase.UpdateBook(ctx, req.GetId(), req.GetName(), req.GetAuthorIds())

	if err != nil {
		return nil, i.convertErr(err)
	}

	return &library.UpdateBookResponse{}, nil
}
