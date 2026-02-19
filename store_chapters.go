package main

import (
	"context"
	"fmt"
)

func (s *server) uploadChapter(ctx context.Context, userID string, ch *chapter) (string, error) {
	var id string
	var exists bool

	query :=
		`
			SELECT EXISTS(SELECT 1 FROM books WHERE id = $1 AND author_id = $2);
		`

	if err := s.store.QueryRowContext(ctx, query, ch.bookID, userID).Scan(&exists); err != nil {
		return "", fmt.Errorf("error checking if books exist, %v", err)
	}

	if !exists {
		return "", errBookNotFound
	}

	query =
		`
			INSERT INTO chapters (chapter_no, title, content, book_id)
			VALUES ($1, $2, $3, $4) RETURNING id;
		`

	if err := s.store.QueryRowContext(ctx, query, ch.chapterNo, ch.title, ch.content, ch.bookID).Scan(&id); err != nil {
		return "", fmt.Errorf("error uploading chapter, %v", err)
	}

	return id, nil
}
