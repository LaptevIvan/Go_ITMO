package controller

import (
	"github.com/project/library/generated/api/library"
	"github.com/project/library/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *implementation) GetAuthorBooks(req *library.GetAuthorBooksRequest, server library.Library_GetAuthorBooksServer) error {
	if err := req.ValidateAll(); logger.CheckError(err, i.logger, "Got invalid request", zap.Any("request", req), zap.Error(err)) {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	books, err := i.booksUseCase.GetAuthorBooks(server.Context(), req.GetAuthorId())

	if err != nil {
		return i.convertErr(err)
	}

	for bk := range books {
		err = server.Send(bk)
		if logger.CheckError(err, i.logger, "Sending error", zap.Error(err)) {
			return status.Error(codes.DataLoss, "Sending error")
		}
	}
	return nil
}
