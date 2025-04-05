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

	const name = "TestBook"
	authors := []string{"1", "2", "3"}

	tests := []struct {
		name       string
		requireErr error
	}{
		{name: "valid add book",
			requireErr: nil},

		{name: "add with internal error",
			requireErr: errInternalBooks},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, mockBookRepo, s := initBookTest(t)
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
				return
			}

			err = validation.ValidateStructWithContext(
				ctx,
				&book,
				validation.Field(&book.ID, is.UUID),
			)
			require.NoError(t, err)
			require.Equal(t, name, book.Name)
			require.Equal(t, authors, book.AuthorIDs)
		})
	}
}

func TestUpdateBook(t *testing.T) {
	t.Parallel()

	const (
		id   = "123"
		name = "TestBook"
	)
	authors := []string{"1", "2", "3"}

	tests := []struct {
		name       string
		requireErr error
	}{
		{name: "valid update book",
			requireErr: nil},
		{name: "update book with internal error",
			requireErr: errInternalBooks},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, mockBookRepo, s := initBookTest(t)
			mockBookRepo.EXPECT().UpdateBook(ctx, gomock.Any()).Return(test.requireErr)
			err := s.UpdateBook(ctx, id, name, authors)
			require.Equal(t, err, test.requireErr)
		})
	}
}

func TestGetBookInfo(t *testing.T) {
	t.Parallel()

	const (
		id   = "123"
		name = "testName"
	)

	tests := []struct {
		name        string
		requireBook entity.Book
		requireErr  error
	}{
		{name: "valid get book info",
			requireBook: entity.Book{
				ID:        id,
				Name:      name,
				AuthorIDs: []string{"1", "2", "3"},
			},
			requireErr: nil},

		{name: "get book with internal error",
			requireBook: entity.Book{},
			requireErr:  errInternalBooks},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, mockBookRepo, s := initBookTest(t)
			tBook := test.requireBook
			tErr := test.requireErr

			mockBookRepo.EXPECT().GetBook(ctx, gomock.Any()).Return(tBook, tErr)
			book, err := s.GetBookInfo(ctx, id)
			require.Equal(t, book, tBook)
			require.Equal(t, err, tErr)
		})
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

func makeFilledChan(books []entity.Book) <-chan entity.Book {
	ans := make(chan entity.Book, len(books))
	defer close(ans)
	for _, b := range books {
		ans <- b
	}
	return ans
}

func readFilledChan(books <-chan entity.Book) []entity.Book {
	if books == nil {
		return nil
	}
	ans := make([]entity.Book, len(books))
	i := 0
	for b := range books {
		ans[i] = b
		i++
	}
	return ans
}

func TestGetAuthorBooks(t *testing.T) {
	t.Parallel()

	const idAuthor = "123"

	tests := []struct {
		name         string
		id           string
		requireBooks []entity.Book
		requireErr   error
	}{
		{name: "valid get author books",
			id:           idAuthor,
			requireBooks: generateBooks(3, idAuthor),
			requireErr:   nil},

		{name: "get author books with internal error",
			id:           idAuthor,
			requireBooks: nil,
			requireErr:   errInternalBooks},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, mockBookRepo, s := initBookTest(t)
			tBooks := test.requireBooks
			tErr := test.requireErr

			var returnChan <-chan entity.Book
			if tErr == nil {
				returnChan = makeFilledChan(tBooks)
			}

			mockBookRepo.EXPECT().GetAuthorBooks(ctx, gomock.Any()).Return(returnChan, tErr)
			bks, err := s.GetAuthorBooks(ctx, test.id)
			require.Equal(t, tBooks, readFilledChan(bks))
			require.Equal(t, tErr, err)
		})
	}
}
