package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

var (
	errGenresNotFound       = errors.New("genres not found")
	errBookNameAlreadyTaken = errors.New("book name already taken")
)

func (s *server) uploadBook(ctx context.Context, book *book) (string, error) {
	tx, err := s.store.Begin()

	if err != nil {
		return "", fmt.Errorf("error starting transaction, %v", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var id string
	query :=
		`
				SELECT id FROM books WHERE name = $1;
			`
	if err = s.store.QueryRowContext(ctx, query, &book.name).Scan(&id); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("error checking book existence, %v", err)
	}
	if id != "" {
		return "", errBookNameAlreadyTaken
	}

	query =
		`
				INSERT INTO books (name, description, author_id, language) VALUES ($1, $2, $3, $4) RETURNING id;
			`

	if err = s.store.QueryRowContext(ctx, query, &book.name, &book.description, &book.author_id, &book.language).Scan(&id); err != nil {
		return "", fmt.Errorf("error inserting book, %v", err)
	}

	val_strings := []string{}
	val_args := []interface{}{}
	position := 1

	for _, sched := range book.release_schedule {
		val_strings = append(val_strings, fmt.Sprintf("($%d, $%d, $%d)", position, position+1, position+2))
		val_args = append(val_args, id, sched.Day, sched.Chapters)
		position += 3
	}

	query = fmt.Sprintf("INSERT INTO release_schedule(book_id, day, no_of_chapters) VALUES %s;", strings.Join(val_strings, ","))

	_, err = tx.ExecContext(ctx, query, val_args...)

	if err != nil {
		return "", fmt.Errorf("error inserting release_schedule, %v", err)
	}

	var genre_ids []string

	var rows *sql.Rows

	query =
		`
			SELECT id FROM genres WHERE genre = ANY($1);
		`
	rows, err = tx.QueryContext(ctx, query, pq.Array(book.genres))

	if err != nil {
		return "", fmt.Errorf("error retrieving genre ids, %v", err)
	}

	defer rows.Close()

	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return "", fmt.Errorf("error scanning genre ids: %v", err)
		}
		genre_ids = append(genre_ids, id)
	}

	if len(genre_ids) != len(book.genres) {
		err = errGenresNotFound
		return "", err
	}

	val_strings = []string{}
	val_args = []interface{}{}
	position = 1

	for _, genreId := range genre_ids {
		val_strings = append(val_strings, fmt.Sprintf("($%d, $%d)", position, position+1))
		val_args = append(val_args, id, genreId)
		position += 2
	}

	query = fmt.Sprintf("INSERT INTO books_genres(book_id, genre_id) VALUES %s ON CONFLICT DO NOTHING;", strings.Join(val_strings, ","))

	_, err = tx.ExecContext(ctx, query, val_args...)

	if err != nil {
		return "", fmt.Errorf("error inserting into book_genres, %v", err)
	}

	query =
		`
				INSERT INTO chapters(chapter_no, title, content, book_id)
				VALUES (0, $1, $2, $3);
		`

	_, err = tx.ExecContext(ctx, query, book.draft_chapter.Title, book.draft_chapter.Content, id)

	if err != nil {
		return "", fmt.Errorf("error inserting draft chapter, %v", err)
	}

	query =
		`
			INSERT INTO recently_uploaded_books(book_id) VALUES ($1);
		`

	if _, err = tx.ExecContext(ctx, query, id); err != nil {
		return "", fmt.Errorf("error inserting in recently uploaded book, %v", err)
	}

	err = tx.Commit()

	if err != nil {
		return "", err
	}

	return id, nil
}

func (s *server) updateBookImage(ctx context.Context, url string, id string) error {
	query :=
		`
				UPDATE books SET image = $1 WHERE id = $2;
			`

	if _, err := s.store.ExecContext(ctx, query, url, id); err != nil {
		return fmt.Errorf("error setting book image, %v", err)
	}
	return nil
}
