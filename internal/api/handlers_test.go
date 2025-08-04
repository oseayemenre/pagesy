package api

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type testLogger struct{}

func (l *testLogger) Info(msg string, args ...any)  {}
func (l *testLogger) Error(msg string, args ...any) {}
func (l *testLogger) Warn(msg string, args ...any)  {}

type testObjectStore struct {
	uploadFileFunc func(ctx context.Context, file io.Reader, id string) (string, error)
}

func (s *testObjectStore) UploadFile(ctx context.Context, file io.Reader, id string) (string, error) {
	if s.uploadFileFunc != nil {
		return s.uploadFileFunc(ctx, file, id)
	}
	return "http://mock-url.com", nil
}

type testStore struct {
	uploadBookFunc                           func(ctx context.Context, book *models.Book) (*uuid.UUID, error)
	updateImageFunc                          func(ctx context.Context, url string, id string) error
	getBooksStatsFunc                        func(ctx context.Context, id string, offset int, limit int) ([]models.Book, error)
	getBooksByGenreFunc                      func(ctx context.Context, genre []string, offset int, limit int, sort string, order string) ([]models.Book, error)
	getBooksByLanguageFunc                   func(ctx context.Context, language []string, offset int, limit int, sort string, order string) ([]models.Book, error)
	getBooksByGenreAndLanguageFunc           func(ctx context.Context, genre []string, language []string, offset int, limit int, sort string, order string) ([]models.Book, error)
	getAllBooksFunc                          func(ctx context.Context, offset int, limit int, sort string, order string) ([]models.Book, error)
	getBookFunc                              func(ctx context.Context, id string) (*models.Book, error)
	deleteBookFunc                           func(ctx context.Context, bookId string, userId string) error
	editBookFunc                             func(ctx context.Context, book *models.Book) error
	approveBookFunc                          func(ctx context.Context, id string, approve bool) error
	completeBookFunc                         func(ctx context.Context, id string, complete bool) error
	getRecentReadsFunc                       func(ctx context.Context, id string, offset int, limit int) ([]models.Book, error)
	userExistsFunc                           func(ctx context.Context, email string, username string) (*uuid.UUID, error)
	getUserPasswordFunc                      func(ctx context.Context, id string) (string, error)
	getUserByIdFunc                          func(ctx context.Context, id string) (*models.User, error)
	checkIfBookIsEligibleForSubscriptionFunc func(ctx context.Context, bookId string) (bool, error)
	markBookForSubscriptionFunc              func(ctx context.Context, bookId string, userId string, eligible bool) error
	updateUserCoinCountFunc                  func(ctx context.Context, userId string, amount int) error
}

func (s *testStore) UploadBook(ctx context.Context, book *models.Book) (*uuid.UUID, error) {
	if s.uploadBookFunc != nil {
		return s.uploadBookFunc(ctx, book)
	}

	id := uuid.New()
	return &id, nil
}

func (s *testStore) UpdateBookImage(ctx context.Context, url string, id string) error {
	if s.updateImageFunc != nil {
		return s.updateImageFunc(ctx, url, id)
	}

	return nil
}

func (s *testStore) GetBooksStats(ctx context.Context, id string, offset int, limit int) ([]models.Book, error) {
	if s.getBooksStatsFunc != nil {
		return s.getBooksStatsFunc(ctx, id, offset, limit)
	}
	return []models.Book{}, nil
}

func (s *testStore) GetBooksByGenre(ctx context.Context, genre []string, offset int, limit int, sort string, order string) ([]models.Book, error) {
	if s.getBooksByGenreFunc != nil {
		return s.getBooksByGenreFunc(ctx, genre, offset, limit, sort, order)
	}
	return []models.Book{}, nil
}

func (s *testStore) GetBooksByLanguage(ctx context.Context, language []string, offset int, limit int, sort string, order string) ([]models.Book, error) {
	if s.getBooksByLanguageFunc != nil {
		return s.getBooksByLanguageFunc(ctx, language, offset, limit, sort, order)
	}
	return []models.Book{}, nil
}

func (s *testStore) GetBooksByGenreAndLanguage(ctx context.Context, genre []string, language []string, offset int, limit int, sort string, order string) ([]models.Book, error) {
	if s.getBooksByGenreAndLanguageFunc != nil {
		return s.getBooksByGenreAndLanguageFunc(ctx, genre, language, offset, limit, sort, order)
	}
	return []models.Book{}, nil
}

func (s *testStore) GetAllBooks(ctx context.Context, offset int, limit int, sort string, order string) ([]models.Book, error) {
	if s.getAllBooksFunc != nil {
		return s.getAllBooksFunc(ctx, offset, limit, sort, order)
	}
	return []models.Book{}, nil
}

func (s *testStore) GetBook(ctx context.Context, id string) (*models.Book, error) {
	if s.getBookFunc != nil {
		return s.getBookFunc(ctx, id)
	}
	return &models.Book{}, nil
}

func (s *testStore) DeleteBook(ctx context.Context, bookId string, userId string) error {
	if s.deleteBookFunc != nil {
		return s.deleteBookFunc(ctx, bookId, userId)
	}
	return nil
}

func (s *testStore) EditBook(ctx context.Context, book *models.Book) error {
	if s.editBookFunc != nil {
		return s.editBookFunc(ctx, book)
	}
	return nil
}

func (s *testStore) ApproveBook(ctx context.Context, id string, approve bool) error {
	if s.approveBookFunc != nil {
		return s.approveBookFunc(ctx, id, approve)
	}
	return nil
}

func (s *testStore) MarkBookAsComplete(ctx context.Context, id string, complete bool) error {
	if s.completeBookFunc != nil {
		return s.completeBookFunc(ctx, id, complete)
	}
	return nil
}

func (s *testStore) GetRecentReads(ctx context.Context, id string, offset int, limit int) ([]models.Book, error) {
	if s.getRecentReadsFunc != nil {
		return s.getRecentReadsFunc(ctx, id, offset, limit)
	}
	return []models.Book{}, nil
}

func (s *testStore) CreateUser(ctx context.Context, user *models.User) (*uuid.UUID, error) {
	return nil, nil
}

func (s *testStore) CheckIfUserExists(ctx context.Context, email string, username string) (*uuid.UUID, error) {
	if s.userExistsFunc != nil {
		return s.userExistsFunc(ctx, email, username)
	}

	return nil, nil
}

func (s *testStore) GetUserById(ctx context.Context, id string) (*models.User, error) {
	if s.getUserByIdFunc != nil {
		return s.getUserByIdFunc(ctx, id)
	}
	return nil, nil
}

func (s *testStore) GetUserPassword(ctx context.Context, id string) (string, error) {
	if s.getUserPasswordFunc != nil {
		return s.getUserPasswordFunc(ctx, id)
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	return string(hash), nil
}

func (s *testStore) UploadChapter(ctx context.Context, userId string, chapter *models.Chapter) (*uuid.UUID, error) {
	id := uuid.New()
	return &id, nil
}

func (s *testStore) CheckIfBookIsEligibleForSubscription(ctx context.Context, bookId string) (bool, error) {
	if s.checkIfBookIsEligibleForSubscriptionFunc != nil {
		return s.checkIfBookIsEligibleForSubscriptionFunc(ctx, bookId)
	}
	return true, nil
}

func (s *testStore) MarkBookForSubscription(ctx context.Context, bookId string, userId string, eligible bool) error {
	if s.markBookForSubscriptionFunc != nil {
		return s.markBookForSubscriptionFunc(ctx, bookId, userId, eligible)
	}
	return nil
}

func (s *testStore) UpdateUserCoinCount(ctx context.Context, userId string, amount int) error {
	if s.updateUserCoinCountFunc != nil {
		return s.updateUserCoinCountFunc(ctx, userId, amount)
	}
	return nil
}
