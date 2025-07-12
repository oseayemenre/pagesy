package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
)

func (s *PostgresStore) CheckIfUserExists(ctx context.Context, email string) (*uuid.UUID, error) {
	var id uuid.UUID

	if err := s.DB.QueryRowContext(ctx, `SELECT id from users WHERE email = $1;`, email).Scan(&id); err != nil {
		return nil, fmt.Errorf("error retrieving user id: %w", err)
	}

	return &id, nil
}

func (s *PostgresStore) CreateUserOauth(ctx context.Context, user *models.User) (*uuid.UUID, error) {
	var id uuid.UUID

	if err := s.DB.QueryRowContext(ctx, `
			INSERT INTO users(email, image) VALUES ($1, $2)
			RETURNING id;
		`, user.Email, user.Image).Scan(&id); err != nil {
		return nil, fmt.Errorf("error inserting into users table: %v", err)
	}

	return &id, nil
}

func (s *PostgresStore) GetUserById(ctx context.Context, id string) (*models.User, error) {
	var user models.User

	if err := s.DB.QueryRowContext(ctx, `
			SELECT u.id, r.name
			FROM users u
			JOIN users_roles ur ON (ur.user_id = u.id)
			JOIN roles r ON (ur.role_id = r.id)
			WHERE u.id = $1;
		`, id).Scan(&user.Id, &user.Role); err != nil {
		return nil, fmt.Errorf("error querying users table: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, `
			SELECT p.name
			FROM privileges p
			JOIN roles_privileges rp ON (rp.privilege_id = p.id)
			JOIN roles r ON (rp.role_id = r.id)
			WHERE r.name = $1;
		`, user.Role)

	if err != nil {
		return nil, fmt.Errorf("error querying privileges table: %v", err)
	}

	defer rows.Close()

	var privileges []string

	for rows.Next() {
		var privilege string

		if err := rows.Scan(&privilege); err != nil {
			return nil, fmt.Errorf("error scanning privileges: %v", err)
		}

		privileges = append(privileges, privilege)
	}

	user.Privileges = privileges

	return &user, nil
}

func (s *PostgresStore) CreateUser(ctx context.Context, user *models.User) (*uuid.UUID, error) {
	var id uuid.UUID

	if err := s.DB.QueryRowContext(ctx, `
		INSERT INTO users(username, email, password) 
		VALUES ($1, $2, $3) RETURNING id;`,
		user.Username, user.Email, user.Password).Scan(&id); err != nil {
		return nil, fmt.Errorf("error inserting in users table: %v", err)
	}
	return &id, nil
}
