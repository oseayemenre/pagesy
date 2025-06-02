package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type Store interface {
	UploadBook(ctx context.Context, book *Book) error
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
