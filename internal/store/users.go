package store

import (
	"context"
	"github.com/google/uuid"
)

func (s *PostgresStore) CreateUser(ctx context.Context) (*uuid.UUID, error) {
	var id uuid.UUID

	if err := s.DB.QueryRowContext(ctx, `
			INSERT INTO users(name) VALUES ('test user') RETURNING id;
		`).Scan(&id); err != nil {
	}

	return &id, nil
}
