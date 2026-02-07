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
	errNoBooksUnderGenre    = errors.New("no books under genre")
)

func (s *server) uploadBook(ctx context.Context, book *book) (string, error) {
	tx, err := s.store.Begin()

	if err != nil {
		return "", fmt.Errorf("error starting transaction, %v", err)
	}

	defer tx.Rollback()

	var id string
	query :=
		`
				SELECT id FROM books WHERE name = $1;
			`
	if err := tx.QueryRowContext(ctx, query, &book.name).Scan(&id); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("error checking book existence, %v", err)
	}
	if id != "" {
		return "", errBookNameAlreadyTaken
	}

	query =
		`
				INSERT INTO books (name, description, author_id, language) VALUES ($1, $2, $3, $4) RETURNING id;
			`

	if err := tx.QueryRowContext(ctx, query, &book.name, &book.description, &book.authorID, &book.language).Scan(&id); err != nil {
		return "", fmt.Errorf("error inserting book, %v", err)
	}

	valStrings := []string{}
	valArgs := []interface{}{}
	position := 1

	for _, sched := range book.releaseSchedule {
		valStrings = append(valStrings, fmt.Sprintf("($%d, $%d, $%d)", position, position+1, position+2))
		valArgs = append(valArgs, id, sched.Day, sched.Chapters)
		position += 3
	}

	query = fmt.Sprintf("INSERT INTO release_schedule(book_id, day, no_of_chapters) VALUES %s;", strings.Join(valStrings, ","))

	_, err = tx.ExecContext(ctx, query, valArgs...)

	if err != nil {
		return "", fmt.Errorf("error inserting release_schedule, %v", err)
	}

	var genreIDs []string

	var rows *sql.Rows

	query =
		`
			SELECT id FROM genres WHERE genre = ANY($1);
		`
	rows, err = tx.QueryContext(ctx, query, pq.Array(book.genres))

	if err != nil {
		if strings.Contains(err.Error(), "invalid input value for enum genre_type") {
			return "", errGenresNotFound
		}
		return "", fmt.Errorf("error retrieving genre ids, %v", err)
	}

	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", fmt.Errorf("error scanning genre ids: %v", err)
		}
		genreIDs = append(genreIDs, id)
	}

	valStrings = []string{}
	valArgs = []interface{}{}
	position = 1

	for _, genreId := range genreIDs {
		valStrings = append(valStrings, fmt.Sprintf("($%d, $%d)", position, position+1))
		valArgs = append(valArgs, id, genreId)
		position += 2
	}

	query = fmt.Sprintf("INSERT INTO books_genres(book_id, genre_id) VALUES %s ON CONFLICT DO NOTHING;", strings.Join(valStrings, ","))

	_, err = tx.ExecContext(ctx, query, valArgs...)

	if err != nil {
		return "", fmt.Errorf("error inserting into book_genres, %v", err)
	}

	query =
		`
				INSERT INTO chapters(chapter_no, title, content, book_id)
				VALUES (0, $1, $2, $3);
		`

	_, err = tx.ExecContext(ctx, query, book.draftChapter.Title, book.draftChapter.Content, id)

	if err != nil {
		return "", fmt.Errorf("error inserting draft chapter, %v", err)
	}

	query =
		`
			INSERT INTO recently_uploaded_books(book_id) VALUES ($1);
		`

	if _, err := tx.ExecContext(ctx, query, id); err != nil {
		return "", fmt.Errorf("error inserting in recently uploaded book, %v", err)
	}

	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("error commititng transaction, %v", err)
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

func helperSortField(sort string) string {
	if sort == "updated" {
		sort = "b.updated_at"
	} else {
		sort = "b.views"
	}

	return sort
}

func (s *server) getGenresAndReleaseSchedule(ctx context.Context, id string) {}

func (s *server) getBooksByGenre(ctx context.Context, genre []string, offset int, limit int, sort string, order string) ([]book, error) {
	var books []book
	var booksMap map[string]book
	query :=
		fmt.Sprintf(`
			SELECT b.name, b.description, b.image, b.views, b.rating, COUNT(c.id)
			FROM books b
			JOIN chapters c ON (b.id = c.book_id)
			JOIN books_genres bg ON (bg.book_id = b.id)
			JOIN genres g ON (g.id = bg.genre_id)
			WHERE g.genres = ANY($1) AND b.approved = true
			GROUP BY b.id
			ORDER BY %s %s
			OFFSET $2 LIMIT $3;
		`, helperSortField(sort), order)

	rows, err := s.store.QueryContext(ctx, query, pq.Array(genre), offset, limit)
	if err != nil {
		return nil, fmt.Errorf("error getting all books, %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var book book
		if err := rows.Scan(&book.name, &book.description, &book.image, &book.views, &book.rating, book.chapterCount); err != nil {
			return nil, fmt.Errorf("error scaaning rows, %v", err)
		}
		books = append(books, book)
	}

	if len(books) < 1 {
		return nil, errNoBooksUnderGenre
	}
	return nil, nil
}
