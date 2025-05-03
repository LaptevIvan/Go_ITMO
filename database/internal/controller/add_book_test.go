package controller

import (
	"context"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/project/library/generated/api/library"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAddBook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		request      *library.AddBookRequest
		codeResponse codes.Code
	}{
		{name: "Good book",
			request: &library.AddBookRequest{
				AuthorIds: []string{uuid.NewString()}},
			codeResponse: codes.OK},

		{name: "Book with invalid author id",
			request: &library.AddBookRequest{
				AuthorIds: []string{"123"}},
			codeResponse: codes.InvalidArgument},

		{name: "Book with unknown author id",
			request: &library.AddBookRequest{
				AuthorIds: []string{uuid.NewString()}},
			codeResponse: codes.NotFound},

		{name: "Book with internal error",
			request: &library.AddBookRequest{
				AuthorIds: []string{uuid.NewString()},
			}, codeResponse: codes.Internal},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, mockBooksUseCase, s := InitBooksTest(t)
			ctx := context.Background()
			code := test.codeResponse
			req := test.request
			rName := req.GetName()
			authorIDs := req.GetAuthorIds()

			if code != codes.InvalidArgument {
				mockBooksUseCase.EXPECT().AddBook(ctx, rName, authorIDs).DoAndReturn(func(ctx context.Context, name string, IDs []string) (*library.AddBookResponse, error) {
					e := convertBookCodeToError(code)
					if code != codes.OK {
						return nil, e
					}

					return &library.AddBookResponse{
						Book: &library.Book{
							Id:       uuid.NewString(),
							Name:     name,
							AuthorId: IDs,
						},
					}, e
				})
			}

			response, err := s.AddBook(ctx, req)
			require.Equal(t, status.Code(err), code)
			if err != nil {
				require.Nil(t, response)
				return
			}
			book := response.GetBook()
			err = validation.ValidateStructWithContext(
				ctx,
				book,
				validation.Field(&book.Id, is.UUID))
			require.NoError(t, err)
			require.Equal(t, book.GetName(), rName)
			require.Equal(t, book.GetAuthorId(), authorIDs)
		})
	}
}
