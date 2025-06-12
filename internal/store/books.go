package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/oseayemenre/pagesy/internal/models"
)

var ErrGenresNotFound = errors.New("some genres were not found in the database")
var ErrCreatorsBooksNotFound = errors.New("creator doesn't have any books")

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
		return ErrGenresNotFound
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

func (s *PostgresStore) GetBooksStats(ctx context.Context, id string, offset int) (*[]models.Book, error) {
	var books []models.Book
	booksMap := make(map[uuid.UUID]*models.Book)

	rows1, err := s.DB.QueryContext(ctx, `
			SELECT b.id, b.name, b.description, b.image, b.views, b.language, b.completed, b.approved, b.created_at, b.updated_at, 
						 COUNT(c.id) AS chapter_count
			FROM books b 
			JOIN chapters c ON (c.book_id = b.id)
			WHERE b.author_id = $1
			GROUP BY b.id
			OFFSET $2 LIMIT 10;
		`, id, offset)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCreatorsBooksNotFound
		}
		return nil, fmt.Errorf("error retrieving books: %v", err)
	}

	defer rows1.Close()

	var bookIds []uuid.UUID

	for rows1.Next() {
		var book models.Book

		if err := rows1.Scan(
			&book.Id,
			&book.Name,
			&book.Description,
			&book.Image,
			&book.Views,
			&book.Language,
			&book.Completed,
			&book.Approved,
			&book.Created_at,
			&book.Updated_at,
			&book.No_Of_Chapters,
		); err != nil {
			return nil, fmt.Errorf("error scanning book rows: %v", err)
		}

		bookIds = append(bookIds, book.Id)
		booksMap[book.Id] = &book
	}

	rows2, err := s.DB.QueryContext(ctx, `
			SELECT book_id, day, no_of_chapters FROM release_schedule WHERE book_id = ANY($1);
		`, pq.Array(bookIds))

	if err != nil {
		return nil, fmt.Errorf("error retrieving release_schedule: %v", err)
	}

	defer rows2.Close()

	for rows2.Next() {
		var release_schedule models.Schedule

		if err := rows2.Scan(
			&release_schedule.BookId,
			&release_schedule.Day,
			&release_schedule.Chapters,
		); err != nil {
			return nil, fmt.Errorf("error scanning release_schedule rows: %v", err)
		}

		if book, ok := booksMap[release_schedule.BookId]; ok {
			book.Release_schedule = append(book.Release_schedule, release_schedule)
		}
	}

	rows3, err := s.DB.QueryContext(ctx, `
			SELECT b.id, g.genres 
			FROM books_genres bg
			JOIN books b ON (b.id = bg.book_id)
			JOIN genres g ON (g.id = bg.genre_id)
			WHERE b.id = ANY($1);
		`, pq.Array(bookIds))

	if err != nil {
		return nil, fmt.Errorf("error retrieving book_genres: %v", err)
	}

	defer rows3.Close()

	for rows3.Next() {
		genre := struct {
			book_id uuid.UUID
			genres  string
		}{}

		if err := rows3.Scan(
			&genre.book_id,
			&genre.genres,
		); err != nil {
			return nil, fmt.Errorf("error scanning genre rows: %v", err)
		}

		if book, ok := booksMap[genre.book_id]; ok {
			book.Genres = append(book.Genres, genre.genres)
		}
	}

	for _, book := range booksMap {
		books = append(books, *book)
	}

	return &books, nil
}
