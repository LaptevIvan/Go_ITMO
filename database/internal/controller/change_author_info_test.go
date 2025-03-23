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

func TestChangeAuthorInfo(t *testing.T) {
	t.Parallel()

	const validName = "Test Testovich"

	tests := []struct {
		name         string
		request      *library.ChangeAuthorInfoRequest
		codeResponse codes.Code
	}{
		{name: "Valid change",
			request: &library.ChangeAuthorInfoRequest{
				Id:   uuid.NewString(),
				Name: validName},
			codeResponse: codes.OK},

		{name: "Empty author's name",
			request: &library.ChangeAuthorInfoRequest{
				Id:   uuid.NewString(),
				Name: ""},
			codeResponse: codes.InvalidArgument},

		{name: "Too long author's name",
			request: &library.ChangeAuthorInfoRequest{
				Id:   uuid.NewString(),
				Name: tooLongName},
			codeResponse: codes.InvalidArgument},

		{name: "Incorrect id",
			request: &library.ChangeAuthorInfoRequest{
				Id:   "123",
				Name: validName},
			codeResponse: codes.InvalidArgument},

		{name: "Unknown author",
			request: &library.ChangeAuthorInfoRequest{
				Id:   uuid.NewString(),
				Name: validName},
			codeResponse: codes.NotFound},

		{name: "Internal error",
			request: &library.ChangeAuthorInfoRequest{
				Id:   uuid.NewString(),
				Name: validName},
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
				mockAuthorUseCase.EXPECT().ChangeAuthorInfo(ctx, req.GetId(), req.GetName()).DoAndReturn(func(ctx context.Context, id, name string) error {
					return convertAuthorCodeToError(code)
				})
			}

			response, err := s.ChangeAuthorInfo(ctx, req)
			require.Equal(t, status.Code(err), code)
			if err != nil {
				require.Nil(t, response)
				return
			}
			require.NotNil(t, response)
		})
	}
}
