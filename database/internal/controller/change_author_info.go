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

var ChangeAuthorInfoDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "library_change_author_info_duration_ms",
	Help:    "Duration of ChangeAuthorInfo in ms",
	Buckets: prometheus.DefBuckets,
})

func init() {
	prometheus.MustRegister(ChangeAuthorInfoDuration)
}

func (i *implementation) ChangeAuthorInfo(ctx context.Context, req *library.ChangeAuthorInfoRequest) (*library.ChangeAuthorInfoResponse, error) {
	start := time.Now()

	defer func() {
		ChangeAuthorInfoDuration.Observe(float64(time.Since(start).Milliseconds()))
	}()

	span := trace.SpanFromContext(ctx)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	if err := req.ValidateAll(); log.ErrorChangeAuthorInfo(i.logger, err, "Got invalid request", traceID, req.GetId(), req.GetName()) {
		span.SetAttributes(attribute.String("author_id", req.GetId()))
		span.RecordError(err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err := i.authorUseCase.ChangeAuthorInfo(ctx, req.GetId(), req.GetName())

	if err != nil {
		return nil, i.convertErr(err)
	}

	return &library.ChangeAuthorInfoResponse{}, nil
}
