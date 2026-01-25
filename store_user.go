package main

import (
	"context"
	"fmt"
)

func (s *server) getUser(ctx context.Context, id string) (*user, error) {
	var user user
	query :=
		`
			SELECT email, display_name, image, about, roles FROM users WHERE id = $1;
		`

	if err := s.store.QueryRowContext(ctx, query, id).Scan(&user.email, &user.display_name, &user.image, &user.about, &user.roles); err != nil {
		return nil, fmt.Errorf("error getting user. %v", err)
	}

	return &user, nil
}
