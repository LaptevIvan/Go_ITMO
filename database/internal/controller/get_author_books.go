package controller

import (
	"github.com/project/library/internal/log"
	"go.opentelemetry.io/otel/attribute"
	"time"

	"github.com/project/library/generated/api/library"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var GetAuthorBooksDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "library_get_author_books_duration_ms",
	Help:    "Duration of GetAuthorBooks in ms",
	Buckets: prometheus.DefBuckets,
})

func init() {
	prometheus.MustRegister(GetAuthorBooksDuration)
}

func (i *implementation) GetAuthorBooks(req *library.GetAuthorBooksRequest, server library.Library_GetAuthorBooksServer) error {
	start := time.Now()

	defer func() {
		GetAuthorBooksDuration.Observe(float64(time.Since(start).Milliseconds()))
	}()

	ctx := server.Context()

	span := trace.SpanFromContext(ctx)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	if err := req.ValidateAll(); log.ErrorGetAuthorBooks(i.logger, err, "Got invalid request", traceID, req.GetAuthorId()) {
		span.SetAttributes(attribute.String("author_id", req.GetAuthorId()))
		span.RecordError(err)
		return status.Error(codes.InvalidArgument, err.Error())
	}

	books, err := i.booksUseCase.GetAuthorBooks(ctx, req.GetAuthorId())

	if err != nil {
		return i.convertErr(err)
	}

	for bk := range books {
		err = server.Send(bk)
		if log.ErrorSendBook(i.logger, err, "Sending error", traceID, bk.Id, req.GetAuthorId()) {
			span.RecordError(err)
			return status.Error(codes.DataLoss, "Sending error")
		}
		log.InfoSendBook(i.logger, "Sent book of author", traceID, bk.Id, req.GetAuthorId())
	}
	return nil
}
