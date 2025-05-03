package controller

import (
	"context"

	"github.com/project/library/pkg/logger"

	"go.uber.org/zap"

	"github.com/project/library/generated/api/library"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *implementation) ChangeAuthorInfo(ctx context.Context, req *library.ChangeAuthorInfoRequest) (*library.ChangeAuthorInfoResponse, error) {
	if err := req.ValidateAll(); logger.CheckError(err, i.logger, "Got invalid request", zap.Any("request", req), zap.Error(err)) {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err := i.authorUseCase.ChangeAuthorInfo(ctx, req.GetId(), req.GetName())

	if err != nil {
		return nil, i.convertErr(err)
	}

	return &library.ChangeAuthorInfoResponse{}, nil
}
