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

func TestGetAuthorInfo(t *testing.T) {
	t.Parallel()

	const name = "Test Testovich"
	tests := []struct {
		name         string
		request      *library.GetAuthorInfoRequest
		codeResponse codes.Code
	}{
		{
			name: "Valid getting info",
			request: &library.GetAuthorInfoRequest{
				Id: uuid.NewString()},
			codeResponse: codes.OK},

		{name: "Invalid id",
			request: &library.GetAuthorInfoRequest{
				Id: "123"},
			codeResponse: codes.InvalidArgument},

		{name: "Unknown author",
			request: &library.GetAuthorInfoRequest{
				Id: uuid.NewString()},
			codeResponse: codes.NotFound},

		{name: "Internal error",
			request: &library.GetAuthorInfoRequest{
				Id: uuid.NewString()},
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
				mockAuthorUseCase.EXPECT().GetAuthorInfo(ctx, req.GetId()).DoAndReturn(func(ctx context.Context, id string) (*library.GetAuthorInfoResponse, error) {
					e := convertAuthorCodeToError(code)
					if code != codes.OK {
						return nil, e
					}
					return &library.GetAuthorInfoResponse{
						Id:   id,
						Name: name,
					}, e
				})
			}

			response, err := s.GetAuthorInfo(ctx, req)
			require.Equal(t, status.Code(err), code)
			if err != nil {
				require.Nil(t, response)
				return
			}
			require.Equal(t, response.GetId(), req.GetId())
		})
	}
}
