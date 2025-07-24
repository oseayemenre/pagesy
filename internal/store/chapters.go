package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
)

func (s *PostgresStore) UploadChapter(ctx context.Context, userId string, chapter *models.Chapter) (*uuid.UUID, error) {
	var id uuid.UUID
	var exists bool

	err := s.DB.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM books WHERE id = $1 AND author_id = $2);
		`, chapter.Book_Id, userId).Scan(&exists)

	if err != nil {
		return nil, fmt.Errorf("error while checking if books exist: %v", err)
	}

	if !exists {
		return nil, ErrBookNotFound
	}

	if err := s.DB.QueryRowContext(ctx, `
			INSERT INTO chapters (chapter_no, title, content, book_id)
			VALUES ($1, $2, $3, $4) RETURNING id;
		`, chapter.Chapter_no, chapter.Title, chapter.Content, chapter.Book_Id).Scan(&id); err != nil {
		return nil, fmt.Errorf("error uploading chapter: %v", err)
	}

	return &id, nil
}
