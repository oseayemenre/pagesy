package main

import (
	"context"
	"fmt"

	"github.com/lib/pq"
)

func (s *server) getUser(ctx context.Context, id string) (*user, error) {
	var user user
	query :=
		`
			SELECT 
				email, 
				display_name, 
				image, 
				about, 
				roles 
			FROM users 
			WHERE id = $1;
		`

	if err := s.store.QueryRowContext(ctx, query, id).Scan(&user.email, &user.displayName, &user.image, &user.about, pq.Array(&user.roles)); err != nil {
		return nil, fmt.Errorf("error getting user. %v", err)
	}

	return &user, nil
}
