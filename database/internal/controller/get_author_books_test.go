package controller

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/project/library/generated/api/library"
	"github.com/project/library/internal/controller/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetAuthorBooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		request      *library.GetAuthorBooksRequest
		codeResponse codes.Code
	}{
		{name: "Valid getting book",
			request: &library.GetAuthorBooksRequest{
				AuthorId: uuid.NewString()},
			codeResponse: codes.OK},

		{name: "Invalid id",
			request: &library.GetAuthorBooksRequest{
				AuthorId: "123"},
			codeResponse: codes.InvalidArgument},

		{name: "Internal error",
			request: &library.GetAuthorBooksRequest{
				AuthorId: uuid.NewString()},
			codeResponse: codes.Internal},

		{name: "Error during sending data",
			request: &library.GetAuthorBooksRequest{
				AuthorId: uuid.NewString()},
			codeResponse: codes.DataLoss},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl, mockBooksUseCase, s := InitBooksTest(t)
			mockServer := mocks.NewMockLibrary_GetAuthorBooksServer(ctrl)
			ctx := context.Background()
			code := test.codeResponse
			req := test.request

			if code != codes.InvalidArgument {
				mockServer.EXPECT().Context().Return(context.Background())
				mockBooksUseCase.EXPECT().GetAuthorBooks(ctx, req.GetAuthorId()).DoAndReturn(func(ctx context.Context, Id string) (<-chan *library.Book, error) {
					e := convertBookCodeToError(code)
					if code == codes.Internal {
						return nil, e
					}
					books := make(chan *library.Book, 1)
					books <- &library.Book{}
					close(books)
					return books, e
				})
				if code != codes.Internal {
					mockServer.EXPECT().Send(gomock.Eq(&library.Book{})).DoAndReturn(func(book *library.Book) error {
						if code != codes.DataLoss {
							return nil
						}
						return errInternal
					})
				}
			}

			err := s.GetAuthorBooks(req, mockServer)
			require.Equal(t, status.Code(err), code)
		})
	}
}
