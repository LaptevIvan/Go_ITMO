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

func TestUpdateBook(t *testing.T) {
	t.Parallel()

	const validBookName = "test book"
	tests := []struct {
		name         string
		request      *library.UpdateBookRequest
		codeResponse codes.Code
	}{
		{name: "Valid update",
			request: &library.UpdateBookRequest{
				Id:        uuid.NewString(),
				Name:      validBookName,
				AuthorIds: []string{uuid.NewString()}},
			codeResponse: codes.OK},

		{name: "Invalid id",
			request: &library.UpdateBookRequest{
				Id:   "123",
				Name: validBookName},
			codeResponse: codes.InvalidArgument},

		{name: "Book with unknown authors",
			request: &library.UpdateBookRequest{
				Id:        uuid.NewString(),
				Name:      validBookName,
				AuthorIds: []string{uuid.NewString()}},
			codeResponse: codes.NotFound},

		{name: "Unknown book",
			request: &library.UpdateBookRequest{
				Id:   uuid.NewString(),
				Name: validBookName},
			codeResponse: codes.NotFound},

		{name: "Book with internal error",
			request: &library.UpdateBookRequest{
				Id:   uuid.NewString(),
				Name: validBookName},
			codeResponse: codes.Internal},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, mockBooksUseCase, s := InitBooksTest(t)
			ctx := context.Background()
			req := test.request
			code := test.codeResponse

			if code != codes.InvalidArgument {
				mockBooksUseCase.EXPECT().UpdateBook(ctx, req.GetId(), req.GetName(), req.GetAuthorIds()).DoAndReturn(func(ctx context.Context, id, name string, ids []string) error {
					return convertBookCodeToError(code)
				})
			}
			_, err := s.UpdateBook(ctx, req)
			require.Equal(t, status.Code(err), code)
		})
	}
}
