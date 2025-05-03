package library

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"go.uber.org/zap"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/project/library/generated/api/library"

	"github.com/project/library/internal/usecase/library/mocks"

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
	logger, e := zap.NewProduction()
	require.NoError(t, e)

	auc := New(logger, nil, mockBooksRepo, nil, nil)
	return ctx, mockBooksRepo, auc
}

func initBookTransactorTest(t *testing.T) (context.Context, *mocks.MockBooksRepository, *mocks.MockOutboxRepository, *mocks.MockTransactor, *libraryImpl) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockBookRepo := mocks.NewMockBooksRepository(ctrl)
	mockOutboxRepo := mocks.NewMockOutboxRepository(ctrl)
	mockTransactor := mocks.NewMockTransactor(ctrl)

	ctx := context.Background()
	logger, e := zap.NewProduction()
	require.NoError(t, e)

	auc := New(logger, nil, mockBookRepo, mockOutboxRepo, mockTransactor)
	return ctx, mockBookRepo, mockOutboxRepo, mockTransactor, auc
}

func TestAddBook(t *testing.T) {
	t.Parallel()

	const name = "TestBook"
	authors := []string{"1", "2", "3"}

	tests := []struct {
		name                 string
		errDBRepoRequire     error
		errOutboxRepoRequire error
	}{
		{name: "valid add book",
			errDBRepoRequire: nil},

		{name: "add with internal error in data base repo",
			errDBRepoRequire: errInternalBooks},

		{name: "add with internal error in outbox repo",
			errOutboxRepoRequire: errInternalBooks},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, mockBookRepo, mockOutboxRepo, mockTransactor, s := initBookTransactorTest(t)
			tDBErr := test.errDBRepoRequire
			tOutboxErr := test.errOutboxRepoRequire

			mockTransactor.EXPECT().WithTx(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, f func(ctx context.Context) error) error {
				return f(ctx)
			})
			mockBookRepo.EXPECT().AddBook(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, input entity.Book) (entity.Book, error) {
				if tDBErr != nil {
					return entity.Book{}, tDBErr
				}
				return input, tDBErr
			})
			mockOutboxRepo.EXPECT().SendMessage(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(tOutboxErr).AnyTimes()
			response, err := s.AddBook(ctx, name, authors)
			switch {
			case tDBErr != nil:
				require.Equal(t, tDBErr, err)
				require.Nil(t, response)
			case tOutboxErr != nil:
				require.Equal(t, tOutboxErr, err)
				require.Nil(t, response)
			default:
				require.NoError(t, err)
				rBook := response.GetBook()
				err = validation.ValidateStructWithContext(
					ctx,
					rBook,
					validation.Field(&rBook.Id, is.UUID),
				)
				require.NoError(t, err)
				require.Equal(t, name, rBook.GetName())
				require.Equal(t, authors, rBook.GetAuthorId())
			}
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
			mockBookRepo.EXPECT().UpdateBook(ctx, entity.Book{
				ID:        id,
				Name:      name,
				AuthorIDs: authors,
			}).Return(test.requireErr)

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
	authorID := []string{uuid.NewString(), uuid.NewString()}

	tests := []struct {
		name            string
		requireResponse *library.GetBookInfoResponse
		requireErr      error
	}{
		{name: "valid get book info",
			requireResponse: &library.GetBookInfoResponse{
				Book: &library.Book{
					Id:       id,
					Name:     name,
					AuthorId: authorID,
				},
			},
			requireErr: nil},

		{name: "get book with internal error",
			requireResponse: nil,
			requireErr:      errInternalBooks,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, mockBookRepo, s := initBookTest(t)
			tResp := test.requireResponse
			tErr := test.requireErr

			mockBookRepo.EXPECT().GetBook(ctx, id).DoAndReturn(func(ctx context.Context, id string) (entity.Book, error) {
				if tErr != nil {
					return entity.Book{}, tErr
				}
				return entity.Book{
					ID:        id,
					Name:      name,
					AuthorIDs: authorID,
				}, nil
			})

			response, err := s.GetBookInfo(ctx, id)
			require.Equal(t, tErr, err)
			if tErr != nil {
				require.Nil(t, response)
				return
			}
			require.Equal(t, tResp.GetBook().GetId(), response.GetBook().GetId())
			require.Equal(t, tResp.GetBook().GetName(), response.GetBook().GetName())
			require.Equal(t, tResp.GetBook().GetAuthorId(), response.GetBook().GetAuthorId())
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

func readFilledChan(t *testing.T, books []entity.Book, bChan <-chan *library.Book) {
	t.Helper()
	if bChan == nil {
		return
	}
	i := 0
	for b := range bChan {
		bCheck := books[i]
		require.Equal(t, bCheck.ID, b.GetId())
		require.Equal(t, bCheck.Name, b.GetName())
		require.Equal(t, bCheck.AuthorIDs, b.GetAuthorId())
		i++
	}
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
			require.Equal(t, tErr, err)
			readFilledChan(t, tBooks, bks)
		})
	}
}
