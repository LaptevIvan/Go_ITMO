package controller

import (
	"github.com/project/library/generated/api/library"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *implementation) GetAuthorBooks(req *library.GetAuthorBooksRequest, server library.Library_GetAuthorBooksServer) error {
	if err := req.ValidateAll(); err != nil {
		if log {
			i.logger.Error("Got invalid request", zap.Any("request", req), zap.Error(err))
		}
		return status.Error(codes.InvalidArgument, err.Error())
	}

	books, err := i.booksUseCase.GetAuthorBooks(server.Context(), req.GetAuthorId())

	if err != nil {
		return i.convertErr(err)
	}

	for _, bk := range books {
		err = server.Send(&library.Book{
			Id:       bk.ID,
			Name:     bk.Name,
			AuthorId: bk.AuthorIDs,
		})
		if err != nil {
			if log {
				i.logger.Error("Send error", zap.Error(err))
			}
			return status.Error(codes.DataLoss, "Sending error")
		}
	}
	return nil
}
