package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	errUserExists = errors.New("user exists")
)

func (s *server) checkIfUserExists(ctx context.Context, email string) (string, error) {
	var id string
	query :=
		`
			SELECT id FROM users WHERE email = $1;
		`
	if err := s.store.QueryRowContext(ctx, query, email).Scan(&id); err != nil {
		return "", fmt.Errorf("error querying db, %w", err)
	}
	return id, nil
}

func (s *server) createUser(ctx context.Context, user *user) (string, error) {
	existingUser, err := s.checkIfUserExists(ctx, user.email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("error retrieving user, %v", err)
	}

	if existingUser != "" {
		return "", errUserExists
	}

	var id string
	query :=
		`
			INSERT INTO users (display_name, email, password, about, image) VALUES ($1, $2, $3, $4, $5) RETURNING id;
		`
	if err := s.store.QueryRowContext(ctx, query, user.displayName, user.email, user.password, user.about, user.image).Scan(&id); err != nil {
		return "", fmt.Errorf("error inserting into users table, %w", err)
	}
	return id, nil
}

func (s *server) getUserPassword(ctx context.Context, id string) (string, error) {
	var password string
	query :=
		`
			SELECT password FROM users WHERE id = $1;
		`
	if err := s.store.QueryRowContext(ctx, query, id).Scan(&password); err != nil {
		return "", fmt.Errorf("error retrieving password, %v", err)
	}

	return password, nil
}
