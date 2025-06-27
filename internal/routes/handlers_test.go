package routes

import (
	"context"
	"io"

	"github.com/oseayemenre/pagesy/internal/models"
)

type testLogger struct{}

func (l *testLogger) Info(msg string, args ...any)  {}
func (l *testLogger) Error(msg string, args ...any) {}
func (l *testLogger) Warn(msg string, args ...any)  {}

type testObjectStore struct{}

func (s *testObjectStore) UploadFile(ctx context.Context, file io.Reader, id string) (string, error) {
	return "http://mock-url.com", nil
}

type testStore struct{}

func (s *testStore) UploadBook(ctx context.Context, book *models.Book) (string, error) {
	return "", nil
}

func (s *testStore) UpdateBookImage(ctx context.Context, url string, id string) error {
	return nil
}

func (s *testStore) GetBooksStats(ctx context.Context, id string, offset int) (*[]models.Book, error) {
	return &[]models.Book{}, nil
}

func (s *testStore) GetBooksByGenre(ctx context.Context, genre []string) (*[]models.Book, error) {
	return &[]models.Book{}, nil
}

func (s *testStore) GetBooksByLanguage(ctx context.Context, language []string) (*[]models.Book, error) {
	return &[]models.Book{}, nil
}

func (s *testStore) GetBooksByGenreAndLanguage(ctx context.Context, genre []string, language []string) (*[]models.Book, error) {
	return &[]models.Book{}, nil
}

func (s *testStore) GetAllBooks(ctx context.Context) (*[]models.Book, error) {
	return &[]models.Book{}, nil
}

func (s *testStore) GetBook(ctx context.Context, id string) (*models.Book, error) {
	return &models.Book{}, nil
}

func (s *testStore) DeleteBook(ctx context.Context, id string) error {
	return nil
}

func (s *testStore) EditBook(ctx context.Context, book *models.HandleEditBookParam) error {
	return nil
}
