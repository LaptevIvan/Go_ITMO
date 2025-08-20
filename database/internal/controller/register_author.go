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

var RegisterAuthorDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "library_register_author_duration_ms",
	Help:    "Duration of RegisterAuthor in ms",
	Buckets: prometheus.DefBuckets,
})

func init() {
	prometheus.MustRegister(RegisterAuthorDuration)
}

func (i *implementation) RegisterAuthor(ctx context.Context, req *library.RegisterAuthorRequest) (*library.RegisterAuthorResponse, error) {
	start := time.Now()

	defer func() {
		RegisterAuthorDuration.Observe(float64(time.Since(start).Milliseconds()))
	}()

	span := trace.SpanFromContext(ctx)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	if err := req.ValidateAll(); log.ErrorRegisterAuthor(i.logger, err, "Got invalid request", traceID, req.GetName()) {
		span.SetAttributes(attribute.String("author_name", req.GetName()))
		span.RecordError(err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	author, err := i.authorUseCase.RegisterAuthor(ctx, req.GetName())

	if err != nil {
		return nil, i.convertErr(err)
	}

	return author, nil
}
