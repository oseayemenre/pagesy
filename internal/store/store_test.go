package store

import (
	"database/sql"
	_ "github.com/lib/pq"
	"testing"
)

func setUpTestDb(t *testing.T) *PostgresStore {
	db, err := sql.Open("postgres", "postgresql://postgres:jane@localhost/pagesy_db?sslmode=disable")

	if err != nil {
		t.Fatalf("error opening db connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("error pinging db: %v", err)
	}

	return &PostgresStore{
		DB: db,
	}
}
