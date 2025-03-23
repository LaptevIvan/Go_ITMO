package controller

import (
	"context"

	"go.uber.org/zap"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/project/library/generated/api/library"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *implementation) GetBookInfo(ctx context.Context, req *library.GetBookInfoRequest) (*library.GetBookInfoResponse, error) {
	if err := req.ValidateAll(); err != nil {
		if log {
			i.logger.Error("Got invalid request", zap.Any("request", req), zap.Error(err))
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	book, err := i.booksUseCase.GetBookInfo(ctx, req.GetId())

	if err != nil {
		return nil, i.convertErr(err)
	}

	return &library.GetBookInfoResponse{
		Book: &library.Book{
			Id:        book.ID,
			Name:      book.Name,
			AuthorId:  book.AuthorIDs,
			CreatedAt: timestamppb.New(book.CreatedAt),
			UpdatedAt: timestamppb.New(book.UpdatedAt),
		},
	}, nil
}
