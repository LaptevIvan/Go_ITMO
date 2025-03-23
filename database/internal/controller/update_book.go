package controller

import (
	"context"

	"go.uber.org/zap"

	"github.com/project/library/generated/api/library"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *implementation) UpdateBook(ctx context.Context, req *library.UpdateBookRequest) (*library.UpdateBookResponse, error) {
	if err := req.ValidateAll(); err != nil {
		if log {
			i.logger.Error("Got invalid request", zap.Any("request", req), zap.Error(err))
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err := i.booksUseCase.UpdateBook(ctx, req.GetId(), req.GetName(), req.GetAuthorIds())

	if err != nil {
		return nil, i.convertErr(err)
	}

	return &library.UpdateBookResponse{}, nil
}
