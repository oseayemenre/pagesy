package store

import (
	"context"
	"fmt"
	"strings"

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

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User

	if err := s.DB.QueryRowContext(ctx, `
			SELECT u.id, r.name
			FROM users u
			JOIN users_roles ur ON (ur.user_id = u.id)
			JOIN roles r ON (ur.role_id = r.id)
			WHERE u.email = $1;
		`, email).Scan(&user.Id, &user.Role); err != nil {
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

	fields := []string{}
	clauses := []string{}
	arguments := []interface{}{user.Username, user.Display_name, user.Email}
	index := 3

	if user.Image != "" {
		index++
		fields = append(fields, "image")
		clauses = append(clauses, fmt.Sprintf("$%d", index))
		arguments = append(arguments, user.Image)
	}

	if user.About != "" {
		index++
		fields = append(fields, "about")
		clauses = append(clauses, fmt.Sprintf("$%d", index))
		arguments = append(arguments, user.About)
	}

	if user.Password != "" {
		index++
		fields = append(fields, "password")
		clauses = append(clauses, fmt.Sprintf("$%d", index))
		arguments = append(arguments, user.Password)
	}

	var fields_formatted string
	var clauses_formatted string

	if len(fields) > 0 {
		fields_formatted = fmt.Sprintf(", %s", strings.Join(fields, ","))
	}

	if len(clauses) > 0 {
		clauses_formatted = fmt.Sprintf(", %s", strings.Join(clauses, ","))
	}

	if err := s.DB.QueryRowContext(ctx, fmt.Sprintf(`
		INSERT INTO users(username, display_name, email%s)
		VALUES ($1, $2, $3%s) RETURNING id;`, fields_formatted, clauses_formatted),
		arguments...).Scan(&id); err != nil {
		return nil, fmt.Errorf("error inserting in users table: %v", err)
	}

	return &id, nil
}
