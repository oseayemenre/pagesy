package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/oseayemenre/pagesy/internal/models"
)

func (s *PostgresStore) UploadBook(ctx context.Context, book *models.Book) error {
	tx, err := s.DB.Begin()

	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var bookID string

	err = tx.QueryRowContext(ctx, `
			INSERT INTO books (name, description, image, author_id, language)
			VALUES ($1, $2, $3, $4, $5) RETURNING id;
		`, book.Name, book.Description, book.Image, book.Author_Id, book.Language).Scan(&bookID)

	if err != nil {
		return fmt.Errorf("error inserting into book table: %v", err)
	}

	valueStrings := []string{}
	valueArgs := []interface{}{}
	argPosition := 1

	for _, sched := range book.Release_schedule {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", argPosition, argPosition+1, argPosition+2))
		valueArgs = append(valueArgs, bookID, sched.Day, sched.Chapters)
		argPosition += 3
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO release_schedule(book_id, day, no_of_chapters)
			VALUES %s; 
		`, strings.Join(valueStrings, ",")), valueArgs...)

	if err != nil {
		return fmt.Errorf("error inserting release_schedule: %v", err)
	}

	var genreIDs []string

	rows, err := tx.QueryContext(ctx, `
			SELECT id FROM genres WHERE genres = ANY($1);
		`, pq.Array(book.Genres))

	if err != nil {
		return fmt.Errorf("error retrieving genre ids: %v", err)
	}

	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("error scanning genre ids: %v", err)
		}
		genreIDs = append(genreIDs, id)
	}

	if len(genreIDs) != len(book.Genres) {
		return fmt.Errorf("some genres were not found in the database")
	}

	valueStrings = []string{}
	valueArgs = []interface{}{}
	argPosition = 1

	for _, genreId := range genreIDs {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", argPosition, argPosition+1))
		valueArgs = append(valueArgs, bookID, genreId)
		argPosition += 2
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO books_genres(book_id, genre_id)
			VALUES %s
			ON CONFLICT DO NOTHING;
		`, strings.Join(valueStrings, ",")), valueArgs...)

	if err != nil {
		return fmt.Errorf("error inserting into book_genres: %v", err)
	}

	_, err = tx.ExecContext(ctx, `
			INSERT INTO chapters(title, content, book_id)
			VALUES ($1, $2, $3);
		`, book.Chapter_Draft.Title, book.Chapter_Draft.Content, bookID)

	if err != nil {
		return fmt.Errorf("error inserting draft chapter: %v", err)
	}

	err = tx.Commit()

	if err != nil {
		return err
	}

	return nil
}
