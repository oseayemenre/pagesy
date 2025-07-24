package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/oseayemenre/pagesy/internal/models"
)

type Store interface {
	UploadBook(ctx context.Context, book *models.Book) (*uuid.UUID, error)
	UpdateBookImage(ctx context.Context, url string, id string) error
	GetBooksStats(ctx context.Context, id string, offset int, limit int) ([]models.Book, error)
	GetBooksByGenre(ctx context.Context, genre []string, offset int, limit int, sort string, order string) ([]models.Book, error)
	GetBooksByLanguage(ctx context.Context, language []string, offset int, limit int, sort string, order string) ([]models.Book, error)
	GetBooksByGenreAndLanguage(ctx context.Context, genre []string, language []string, offset int, limit int, sort string, order string) ([]models.Book, error)
	GetAllBooks(ctx context.Context, offset int, limit int, sort string, order string) ([]models.Book, error)
	GetBook(ctx context.Context, id string) (*models.Book, error)
	DeleteBook(ctx context.Context, id string) error
	EditBook(ctx context.Context, book *models.HandleEditBookParam) error
	ApproveBook(ctx context.Context, id string, approve bool) error
	MarkBookAsComplete(ctx context.Context, id string, complete bool) error
	GetRecentReads(ctx context.Context, id string, offset int, limit int) ([]models.Book, error)
	CheckIfUserExists(ctx context.Context, email string, username string) (*uuid.UUID, error)
	GetUserById(ctx context.Context, id string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) (*uuid.UUID, error)
	GetUserPassword(ctx context.Context, id string) (string, error)
	UploadChapter(ctx context.Context, userId string, chapter *models.Chapter) (*uuid.UUID, error)
}

type PostgresStore struct {
	*sql.DB
}

func NewPostgresStore(conn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", conn)

	if err != nil {
		return nil, fmt.Errorf("error connecting to db: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging db: %v", err)
	}

	return &PostgresStore{
		DB: db,
	}, nil
}
