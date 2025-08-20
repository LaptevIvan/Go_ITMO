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

var GetAuthorInfoDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "library_get_author_info_duration_ms",
	Help:    "Duration of GetAuthorInfo in ms",
	Buckets: prometheus.DefBuckets,
})

func init() {
	prometheus.MustRegister(GetAuthorInfoDuration)
}

func (i *implementation) GetAuthorInfo(ctx context.Context, req *library.GetAuthorInfoRequest) (*library.GetAuthorInfoResponse, error) {
	start := time.Now()

	defer func() {
		GetAuthorInfoDuration.Observe(float64(time.Since(start).Milliseconds()))
	}()

	span := trace.SpanFromContext(ctx)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	if err := req.ValidateAll(); log.ErrorGetAuthorInfo(i.logger, err, "Got invalid request", traceID, req.GetId()) {
		span.SetAttributes(attribute.String("author_id", req.GetId()))
		span.RecordError(err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	author, err := i.authorUseCase.GetAuthorInfo(ctx, req.GetId())

	if err != nil {
		return nil, i.convertErr(err)
	}

	return author, nil
}
