package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	errChapterNotFound = errors.New("chapter not found")
)

func (s *server) checkIfBookBelongsToUser(ctx context.Context, bookID, userID string) error {
	var exists bool

	query :=
		`
			SELECT EXISTS(SELECT 1 FROM books WHERE id = $1 AND author_id = $2);
		`

	if err := s.store.QueryRowContext(ctx, query, bookID, userID).Scan(&exists); err != nil {
		return fmt.Errorf("error checking if books exist, %v", err)
	}

	if !exists {
		return errBookNotFound
	}

	return nil
}

func (s *server) uploadChapter(ctx context.Context, userID string, ch *chapter) (string, error) {
	var id string

	if err := s.checkIfBookBelongsToUser(ctx, ch.bookID, userID); err != nil {
		return "", err
	}

	query :=
		`
			INSERT INTO chapters (chapter_no, title, content, book_id)
			VALUES ($1, $2, $3, $4) RETURNING id;
		`

	if err := s.store.QueryRowContext(ctx, query, ch.chapterNo, ch.title, ch.content, ch.bookID).Scan(&id); err != nil {
		return "", fmt.Errorf("error uploading chapter, %v", err)
	}

	return id, nil
}

func (s *server) getChapter(ctx context.Context, userID, bookID string) (*chapter, error) {
	var ch chapter

	query :=
		`
			SELECT
				book_id,
				chapter_no, 
				title, 
				content 
			FROM chapters c
			JOIN books b ON (c.book_id = b.id)
			WHERE c.id = $1 AND b.approved = true;
		`

	if err := s.store.QueryRowContext(ctx, query, bookID).Scan(&ch.bookID, &ch.chapterNo, &ch.title, &ch.content); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errChapterNotFound
		}
		return nil, fmt.Errorf("error scanning chapter, %v", err)
	}

	query =
		`
			INSERT INTO recent_books(user_id, book_id, chapter)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, book_id)
			DO UPDATE SET 
				chapter = EXCLUDED.chapter, 
				updated_at = NOW();
		`

	if _, err := s.store.ExecContext(ctx, query, userID, ch.bookID, ch.chapterNo); err != nil {
		return nil, fmt.Errorf("error inserting into recent books, %v", err)
	}

	return &ch, nil
}

func (s *server) deleteChapter(ctx context.Context, userID, bookID, chapterID string) error {
	if err := s.checkIfBookBelongsToUser(ctx, bookID, userID); err != nil {
		return err
	}

	query :=
		`
			DELETE FROM chapters WHERE id = $1;
		`

	results, err := s.store.ExecContext(ctx, query, chapterID)
	if err != nil {
		return fmt.Errorf("error deleting chapter, %v", err)
	}

	rows, err := results.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking number of rows affected, %v", err)
	}
	if rows == 0 {
		return errChapterNotFound
	}

	return nil
}

func (s *server) editChapter(ctx context.Context, userID string, ch *chapter) error {
	if err := s.checkIfBookBelongsToUser(ctx, ch.bookID, userID); err != nil {
		return err
	}

	index := 1
	var values []string
	var args []any

	if ch.content != "" {
		values = append(values, fmt.Sprintf("content=$%v", index))
		args = append(args, ch.content)
		index++
	}
	if ch.title != "" {
		values = append(values, fmt.Sprintf("title=$%v", index))
		args = append(args, ch.title)
		index++
	}

	query := fmt.Sprintf("UPDATE chapters SET %v WHERE id = $%v;", strings.Join(values, ","), index)
	args = append(args, ch.id)

	results, err := s.store.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("error updating chapter chapter, %v", err)
	}

	rows, err := results.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking number of rows affected, %v", err)
	}
	if rows == 0 {
		return errChapterNotFound
	}

	return nil
}
