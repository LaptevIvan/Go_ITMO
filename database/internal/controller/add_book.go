package controller

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"go.uber.org/zap"

	"github.com/project/library/generated/api/library"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *implementation) AddBook(ctx context.Context, req *library.AddBookRequest) (*library.AddBookResponse, error) {
	if err := req.ValidateAll(); err != nil {
		if log {
			i.logger.Error("Got invalid request", zap.Any("request", req), zap.Error(err))
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	book, err := i.booksUseCase.AddBook(ctx, req.GetName(), req.GetAuthorIds())

	if err != nil {
		return nil, i.convertErr(err)
	}

	return &library.AddBookResponse{
		Book: &library.Book{
			Id:        book.ID,
			Name:      book.Name,
			AuthorId:  book.AuthorIDs,
			CreatedAt: timestamppb.New(book.CreatedAt),
			UpdatedAt: timestamppb.New(book.UpdatedAt),
		},
	}, nil
}
