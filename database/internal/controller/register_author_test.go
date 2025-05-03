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

func TestRegisterAuthor(t *testing.T) {
	t.Parallel()

	const validAuthorName = "Test testovich"
	tests := []struct {
		name         string
		request      *library.RegisterAuthorRequest
		codeResponse codes.Code
	}{
		{name: "Valid registration",
			request: &library.RegisterAuthorRequest{
				Name: validAuthorName},
			codeResponse: codes.OK},

		{name: "Empty author's name",
			request: &library.RegisterAuthorRequest{
				Name: ""},
			codeResponse: codes.InvalidArgument},

		{name: "Too long author's name",
			request: &library.RegisterAuthorRequest{
				Name: tooLongName},
			codeResponse: codes.InvalidArgument},

		{name: "Internal error",
			request: &library.RegisterAuthorRequest{
				Name: validAuthorName},
			codeResponse: codes.Internal},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, mockAuthorUseCase, s := InitAuthorTest(t)
			ctx := context.Background()
			code := test.codeResponse
			req := test.request
			if code != codes.InvalidArgument {
				mockAuthorUseCase.EXPECT().RegisterAuthor(ctx, req.GetName()).DoAndReturn(func(ctx context.Context, name string) (*library.RegisterAuthorResponse, error) {
					e := convertAuthorCodeToError(code)
					if code != codes.OK {
						return nil, e
					}

					return &library.RegisterAuthorResponse{
						Id: uuid.NewString(),
					}, e
				})
			}

			response, err := s.RegisterAuthor(ctx, req)
			require.Equal(t, status.Code(err), code)
			if err != nil {
				require.Nil(t, response)
				return
			}
			err = validation.ValidateStructWithContext(
				ctx,
				response,
				validation.Field(&response.Id, is.UUID))
			require.NoError(t, err)
		})
	}
}
