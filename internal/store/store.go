package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/oseayemenre/pagesy/internal/models"
)

type Store interface {
	UploadBook(ctx context.Context, book *models.Book) (string, error)
	GetBooksStats(ctx context.Context, id string, offset int) (*[]models.Book, error)
	GetBooksByGenre(ctx context.Context, genre []string) (*[]models.Book, error)
	GetBooksByLanguage(ctx context.Context, language []string) (*[]models.Book, error)
	GetBooksByGenreAndLanguage(ctx context.Context, genre []string, language []string) (*[]models.Book, error)
	GetAllBooks(ctx context.Context) (*[]models.Book, error)
	GetBook(ctx context.Context, id string) (*models.Book, error)
	DeleteBook(ctx context.Context, id string) error
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
