package main

import (
	"context"
	"fmt"
)

func (s *server) uploadBook(ctx context.Context, book *book) (string, error) {
	var id string
	query :=
		`
			INSERT INTO books (name, description, author_id, language) VALUES ($1, $2, $3, $4);
		`

	if err := s.store.QueryRowContext(ctx, query, &book.name, &book.description, &book.author_id, &book.language).Scan(&id); err != nil {
		return "", fmt.Errorf("error inserting book, %v", err)
	}

	return id, nil
}
