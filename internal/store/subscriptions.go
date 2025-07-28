package store

import (
	"context"
	"fmt"
)

func (s *PostgresStore) CheckIfBookIsEligibleForSubscription(ctx context.Context, bookId string) (bool, error) {
	var count int

	query := `
		SELECT COUNT(*) FROM chapters WHERE book_id = $1; 
	`

	if err := s.DB.QueryRowContext(ctx, query, bookId).Scan(&count); err != nil {
		return false, fmt.Errorf("error quering chapter count: %v", err)
	}

	return count >= 15, nil
}

func (s *PostgresStore) MarkBookForSubscription(ctx context.Context, bookId string, userId string, eligible bool) error {
	query := `
		UPDATE books
		SET subscription = $1
		WHERE id = $2 AND author_id = $3;
	`

	_, err := s.DB.ExecContext(ctx, query, eligible, bookId, userId)

	if err != nil {
		return fmt.Errorf("error marking book for subscription: %v", err)
	}

	return nil
}
