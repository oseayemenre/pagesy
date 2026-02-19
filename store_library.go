package main

import (
	"context"
	"fmt"
)

func (s *server) getUserLibrary(ctx context.Context, userID string) ([]string, error) {
	var bookIDs []string

	query :=
		`
			SELECT book_id FROM library WHERE user_id = $1;
		`

	rows, err := s.store.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting books in user library, %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bookID string

		if err := rows.Scan(&bookID); err != nil {
			return nil, fmt.Errorf("error scanning book ids, %v", err)
		}

		bookIDs = append(bookIDs, bookID)
	}

	return bookIDs, nil
}
