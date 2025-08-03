package store

import (
	"context"
	"fmt"
)

func (s *PostgresStore) UpdateUserCoinCount(ctx context.Context, userId string, amount int) error {
	query := `
			UPDATE users
			SET coins = coins + $1
			WHERE id = $2;
	`

	_, err := s.DB.ExecContext(ctx, query, amount, userId)

	if err != nil {
		return fmt.Errorf("error updating coin count: %v", err)
	}

	return nil
}
