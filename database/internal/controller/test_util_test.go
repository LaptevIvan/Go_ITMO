package controller

import (
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/project/library/internal/controller/mocks"

	"github.com/project/library/internal/entity"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
)

var (
	errInternal = errors.New("internal error")
	tooLongName = strings.Repeat("Too long name", 40)
)

func InitBooksTest(t *testing.T) (*gomock.Controller, *mocks.MockBooksUseCase, *implementation) {
	t.Helper()
	ctrl := gomock.NewController(t)
	booksUseCase := mocks.NewMockBooksUseCase(ctrl)
	logger, err := zap.NewProduction()
	if err != nil {
		t.Fatal("assertion error: " + err.Error())
	}
	service := New(logger, booksUseCase, nil)
	return ctrl, booksUseCase, service
}

func InitAuthorTest(t *testing.T) (*gomock.Controller, *mocks.MockAuthorUseCase, *implementation) {
	t.Helper()
	ctrl := gomock.NewController(t)
	authorUseCase := mocks.NewMockAuthorUseCase(ctrl)
	logger, err := zap.NewProduction()
	if err != nil {
		t.Fatal("assertion error: " + err.Error())
	}
	service := New(logger, nil, authorUseCase)
	return ctrl, authorUseCase, service
}

func convertBookCodeToError(code codes.Code) error {
	switch code {
	case codes.NotFound:
		return entity.ErrBookNotFound
	case codes.Internal:
		return errInternal
	default:
		return nil
	}
}

func convertAuthorCodeToError(code codes.Code) error {
	switch code {
	case codes.NotFound:
		return entity.ErrAuthorNotFound
	case codes.Internal:
		return errInternal
	default:
		return nil
	}
}
