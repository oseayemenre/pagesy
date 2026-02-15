package main

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func connectTestDb(t *testing.T) *sql.DB {
	db, err := sql.Open("postgres", "postgres://pagesy_user:pagesy_password@localhost:5432/pagesy_db?sslmode=disable")
	if err != nil {
		t.Fatalf("error opening db connection, %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("error pinging db, %v", err)
	}
	return db
}

func createAndCleanUpUser(t *testing.T, db *sql.DB) string {
	var id string
	hash, _ := bcrypt.GenerateFromPassword([]byte("test_password"), bcrypt.DefaultCost)
	query :=
		`
			INSERT INTO users (display_name, email, password) VALUES ('test_display', 'test@test.com', $1) RETURNING id;
		`
	if err := db.QueryRowContext(context.Background(), query, hash).Scan(&id); err != nil {
		t.Errorf("error creating new user, %v", err)
	}

	t.Cleanup(func() {
		query :=
			`
				DELETE FROM users WHERE email = 'test@test.com' OR email = 'user@user.com';
			`
		if _, err := db.ExecContext(context.Background(), query); err != nil {
			t.Errorf("error deleting users, %v", err)
		}
	})
	return id
}

func createBook(t *testing.T, author_id string, db *sql.DB) string {
	var id string
	query :=
		`
			INSERT INTO books(name, description, author_id, approved) VALUES ('test book taken', 'test book description', $1, 'true') RETURNING id;
		`
	if err := db.QueryRowContext(context.Background(), query, author_id).Scan(&id); err != nil {
		t.Errorf("error creating new book, %v", err)
	}

	return id
}
