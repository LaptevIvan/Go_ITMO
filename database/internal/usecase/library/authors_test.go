package library

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/project/library/generated/api/library"
	"github.com/project/library/internal/usecase/library/mocks"

	"github.com/project/library/internal/entity"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var errInternalAuthor = errors.New("internal error")

func initAuthorTest(t *testing.T) (context.Context, *mocks.MockAuthorRepository, *libraryImpl) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockAuthorRepo := mocks.NewMockAuthorRepository(ctrl)
	ctx := context.Background()
	logger, e := zap.NewProduction()
	require.NoError(t, e)

	auc := New(logger, mockAuthorRepo, nil, nil, nil)
	return ctx, mockAuthorRepo, auc
}

func initAuthorTransactorTest(t *testing.T) (context.Context, *mocks.MockAuthorRepository, *mocks.MockOutboxRepository, *mocks.MockTransactor, *libraryImpl) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockAuthorRepo := mocks.NewMockAuthorRepository(ctrl)
	mockOutboxRepo := mocks.NewMockOutboxRepository(ctrl)
	mockTransactor := mocks.NewMockTransactor(ctrl)

	ctx := context.Background()
	logger, e := zap.NewProduction()
	require.NoError(t, e)

	auc := New(logger, mockAuthorRepo, nil, mockOutboxRepo, mockTransactor)
	return ctx, mockAuthorRepo, mockOutboxRepo, mockTransactor, auc
}

func TestRegisterAuthor(t *testing.T) {
	t.Parallel()

	const name = "testAuthor"

	tests := []struct {
		name                 string
		errDBRepoRequire     error
		errOutboxRepoRequire error
	}{
		{name: "valid registration",
			errDBRepoRequire:     nil,
			errOutboxRepoRequire: nil},
		{name: "register with internal error in data base repo",
			errDBRepoRequire: errInternalAuthor},
		{name: "register with internal error in outbox repo",
			errOutboxRepoRequire: errInternalAuthor},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, mockAuthorRepo, mockOutboxRepo, mockTransactor, s := initAuthorTransactorTest(t)
			tDBErr := test.errDBRepoRequire
			tOutboxErr := test.errOutboxRepoRequire

			mockTransactor.EXPECT().WithTx(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, f func(ctx context.Context) error) error {
				return f(ctx)
			})
			mockAuthorRepo.EXPECT().RegisterAuthor(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, input entity.Author) (entity.Author, error) {
				if tDBErr != nil {
					return entity.Author{}, tDBErr
				}
				return input, tDBErr
			})
			mockOutboxRepo.EXPECT().SendMessage(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(tOutboxErr).AnyTimes()
			response, err := s.RegisterAuthor(ctx, name)
			switch {
			case tDBErr != nil:
				require.Equal(t, tDBErr, err)
				require.Nil(t, response)
			case tOutboxErr != nil:
				require.Equal(t, tOutboxErr, err)
				require.Nil(t, response)
			default:
				require.NoError(t, err)
				err = validation.ValidateStructWithContext(
					ctx,
					response,
					validation.Field(&response.Id, is.UUID),
				)
				require.NoError(t, err)
			}
		})
	}
}

func TestChangeAuthorInfo(t *testing.T) {
	t.Parallel()

	const (
		id   = "123"
		name = "Test testovich"
	)

	tests := []struct {
		name       string
		errRequire error
	}{
		{name: "valid change author",
			errRequire: nil},
		{name: "change with internal error",
			errRequire: errInternalAuthor},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, mockAuthorRepo, s := initAuthorTest(t)
			tErr := test.errRequire
			mockAuthorRepo.EXPECT().ChangeAuthorInfo(ctx, entity.Author{
				ID:   id,
				Name: name,
			}).Return(tErr)
			err := s.ChangeAuthorInfo(ctx, id, name)
			require.Equal(t, tErr, err)
		})
	}
}

func TestGetAuthorInfo(t *testing.T) {
	t.Parallel()

	const (
		id   = "123"
		name = "testName"
	)

	tests := []struct {
		name            string
		requireResponse *library.GetAuthorInfoResponse
		requireErr      error
	}{
		{
			name: "valid getting info",
			requireResponse: &library.GetAuthorInfoResponse{
				Id:   id,
				Name: name,
			},
			requireErr: nil},

		{
			name:            "Get info with internal error",
			requireResponse: nil,
			requireErr:      errInternalAuthor},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, mockAuthorRepo, s := initAuthorTest(t)
			tResp := test.requireResponse
			tErr := test.requireErr

			mockAuthorRepo.EXPECT().GetAuthorInfo(ctx, id).DoAndReturn(func(ctx context.Context, idAuthor string) (entity.Author, error) {
				if tErr != nil {
					return entity.Author{}, tErr
				}
				return entity.Author{
					ID:   id,
					Name: name,
				}, nil
			})
			response, err := s.GetAuthorInfo(ctx, id)
			require.Equal(t, tErr, err)
			require.Equal(t, tResp, response)
		})
	}
}
