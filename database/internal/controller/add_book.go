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

var AddBookDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "library_add_book_duration_ms",
	Help:    "Duration of AddBook in ms",
	Buckets: prometheus.DefBuckets,
})

func init() {
	prometheus.MustRegister(AddBookDuration)
}

func (i *implementation) AddBook(ctx context.Context, req *library.AddBookRequest) (*library.AddBookResponse, error) {
	start := time.Now()

	defer func() {
		AddBookDuration.Observe(float64(time.Since(start).Milliseconds()))
	}()

	span := trace.SpanFromContext(ctx)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	if err := req.ValidateAll(); log.ErrorAddBook(i.logger, err, "Got invalid request", traceID, req.GetName(), req.GetAuthorIds()) {
		span.SetAttributes(attribute.String("book_name", req.GetName()))
		span.SetAttributes(attribute.StringSlice("book_authors", req.GetAuthorIds()))
		span.RecordError(err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	book, err := i.booksUseCase.AddBook(ctx, req.GetName(), req.GetAuthorIds())

	if err != nil {
		return nil, i.convertErr(err)
	}

	return book, nil
}
