package library

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"github.com/project/library/internal/usecase/library/mocks"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
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
	logger, err := zap.NewProduction()
	if err != nil {
		t.Fatal("assertion error: " + err.Error())
	}
	auc := New(logger, mockAuthorRepo, nil)
	return ctx, mockAuthorRepo, auc
}

func TestRegisterAuthor(t *testing.T) {
	t.Parallel()
	ctx, mockAuthorRepo, s := initAuthorTest(t)

	const name = "testAuthor"

	tests := []struct {
		name       string
		errRequire error
	}{
		{name: "valid registration",
			errRequire: nil},
		{name: "register with internal error",
			errRequire: errInternalAuthor},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			tErr := test.errRequire
			mockAuthorRepo.EXPECT().RegisterAuthor(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, input entity.Author) (entity.Author, error) {
				if tErr != nil {
					return entity.Author{}, tErr
				}
				return input, tErr
			})
			author, err := s.RegisterAuthor(ctx, name)
			require.Equal(t, err, tErr)
			if err != nil {
				require.Empty(t, author)
				return
			}

			err = validation.ValidateStructWithContext(
				ctx,
				&author,
				validation.Field(&author.ID, is.UUID),
			)
			require.NoError(t, err)
			require.Equal(t, name, author.Name)
		})
	}
}

func TestChangeAuthorInfo(t *testing.T) {
	t.Parallel()
	ctx, mockAuthorRepo, s := initAuthorTest(t)

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

			tErr := test.errRequire
			mockAuthorRepo.EXPECT().ChangeAuthorInfo(ctx, gomock.Any()).Return(tErr)
			err := s.ChangeAuthorInfo(ctx, id, name)
			require.Equal(t, err, tErr)
		})
	}
}

func TestGetAuthorInfo(t *testing.T) {
	t.Parallel()
	ctx, mockAuthorRepo, s := initAuthorTest(t)

	const (
		id   = "123"
		name = "testName"
	)

	tests := []struct {
		name          string
		requireAuthor entity.Author
		requireErr    error
	}{
		{
			name: "valid getting info",
			requireAuthor: entity.Author{
				ID:   id,
				Name: name,
			},
			requireErr: nil},

		{
			name:          "Get info with internal error",
			requireAuthor: entity.Author{},
			requireErr:    errInternalAuthor},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			tAuthor := test.requireAuthor
			tErr := test.requireErr

			mockAuthorRepo.EXPECT().GetAuthorInfo(ctx, gomock.Any()).Return(tAuthor, tErr)
			author, err := s.GetAuthorInfo(ctx, id)
			require.Equal(t, err, tErr)
			require.Equal(t, author, tAuthor)
		})
	}
}
