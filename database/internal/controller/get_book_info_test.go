package controller

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/project/library/generated/api/library"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetBookInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		request      *library.GetBookInfoRequest
		codeResponse codes.Code
	}{
		{name: "Valid getting info",
			request: &library.GetBookInfoRequest{
				Id: uuid.NewString()},
			codeResponse: codes.OK},

		{name: "Invalid id",
			request: &library.GetBookInfoRequest{
				Id: "123"},
			codeResponse: codes.InvalidArgument},

		{name: "Unknown book",
			request: &library.GetBookInfoRequest{
				Id: uuid.NewString()},
			codeResponse: codes.NotFound},

		{name: "Internal error",
			request: &library.GetBookInfoRequest{
				Id: uuid.NewString()},
			codeResponse: codes.Internal},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, mockBooksUseCase, s := InitBooksTest(t)
			ctx := context.Background()
			code := test.codeResponse
			req := test.request

			if code != codes.InvalidArgument {
				mockBooksUseCase.EXPECT().GetBookInfo(ctx, req.GetId()).DoAndReturn(func(ctx context.Context, Id string) (*library.GetBookInfoResponse, error) {
					e := convertBookCodeToError(code)
					if code != codes.OK {
						return nil, e
					}

					return &library.GetBookInfoResponse{
						Book: &library.Book{
							Id:       Id,
							Name:     "Returned book",
							AuthorId: []string{uuid.NewString()},
						},
					}, e
				})
			}

			response, err := s.GetBookInfo(ctx, req)
			require.Equal(t, status.Code(err), code)
			if err != nil {
				require.Nil(t, response)
				return
			}
			require.Equal(t, response.GetBook().GetId(), req.GetId())
		})
	}
}
