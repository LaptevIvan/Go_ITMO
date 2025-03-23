package library

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"go.uber.org/zap"

	"github.com/project/library/internal/usecase/library/mocks"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/project/library/internal/entity"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var errInternalBooks = errors.New("internal error")

func initBookTest(t *testing.T) (context.Context, *mocks.MockBooksRepository, *libraryImpl) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockBooksRepo := mocks.NewMockBooksRepository(ctrl)
	ctx := context.Background()
	logger, err := zap.NewProduction()
	if err != nil {
		t.Fatal("assertion error: " + err.Error())
	}
	auc := New(logger, nil, mockBooksRepo)
	return ctx, mockBooksRepo, auc
}

func TestAddBook(t *testing.T) {
	t.Parallel()
	ctx, mockBookRepo, s := initBookTest(t)

	const name = "TestBook"
	authors := []string{"1", "2", "3"}

	tests := []struct {
		requireErr error
	}{
		{nil},
		{errInternalBooks},
	}

	for _, test := range tests {
		mockBookRepo.EXPECT().AddBook(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, input entity.Book) (entity.Book, error) {
			e := test.requireErr
			if e != nil {
				return entity.Book{}, e
			}
			return input, e
		})
		book, err := s.AddBook(ctx, name, authors)
		require.Equal(t, err, test.requireErr)
		if err != nil {
			require.Empty(t, book)
			continue
		}

		err = validation.ValidateStructWithContext(
			ctx,
			&book,
			validation.Field(&book.ID, is.UUID),
		)
		require.NoError(t, err)
		require.Equal(t, name, book.Name)
		require.Equal(t, authors, book.AuthorIDs)
	}
}

func TestUpdateBook(t *testing.T) {
	t.Parallel()
	ctx, mockBookRepo, s := initBookTest(t)

	const (
		id   = "123"
		name = "TestBook"
	)
	authors := []string{"1", "2", "3"}

	tests := []struct {
		requireErr error
	}{
		{nil},
		{errInternalBooks},
	}

	for _, test := range tests {
		mockBookRepo.EXPECT().UpdateBook(ctx, gomock.Any()).Return(test.requireErr)
		err := s.UpdateBook(ctx, id, name, authors)
		require.Equal(t, err, test.requireErr)
	}
}

func TestGetBookInfo(t *testing.T) {
	t.Parallel()
	ctx, mockBookRepo, s := initBookTest(t)

	const (
		id   = "123"
		name = "testName"
	)

	tests := []struct {
		requireBook entity.Book
		requireErr  error
	}{
		{requireBook: entity.Book{
			ID:        id,
			Name:      name,
			AuthorIDs: []string{"1", "2", "3"},
		},
			requireErr: nil},

		{requireBook: entity.Book{},
			requireErr: errInternalBooks},
	}

	for _, test := range tests {
		tBook := test.requireBook
		tErr := test.requireErr

		mockBookRepo.EXPECT().GetBook(ctx, gomock.Any()).Return(tBook, tErr)
		book, err := s.GetBookInfo(ctx, id)
		require.Equal(t, book, tBook)
		require.Equal(t, err, tErr)
	}
}

func generateBooks(n int, authorID string) []entity.Book {
	ans := make([]entity.Book, n)
	const name = "nameTest"
	for i := 0; i < n; i++ {
		ans[i] = entity.Book{
			ID:        strconv.Itoa(i),
			Name:      name,
			AuthorIDs: []string{authorID},
		}
	}
	return ans
}

func TestGetAuthorBooks(t *testing.T) {
	t.Parallel()
	ctx, mockBookRepo, s := initBookTest(t)

	const idAuthor = "123"

	tests := []struct {
		id           string
		requireBooks []entity.Book
		requireErr   error
	}{
		{idAuthor, generateBooks(3, idAuthor), nil},
		{idAuthor, nil, errInternalBooks},
	}

	for _, test := range tests {
		tBooks := test.requireBooks
		tErr := test.requireErr

		mockBookRepo.EXPECT().GetAuthorBooks(ctx, gomock.Any()).Return(tBooks, tErr)
		bks, err := s.GetAuthorBooks(ctx, test.id)
		require.Equal(t, bks, tBooks)
		require.Equal(t, err, tErr)
	}
}
