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
	errGenresNotFound               = errors.New("genres not found")
	errBookNameAlreadyTaken         = errors.New("book name already taken")
	errNoBooksUnderGenre            = errors.New("no books under genre")
	errNoBooksUnderLanguage         = errors.New("no books under language")
	errNoBooksUnderGenreAndLanguage = errors.New("no books under genre andn language")
	errUserHasNoBooks               = errors.New("user has no books")
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

type rowsFuncType func(rows *sql.Rows, bookIDs *[]string, booksMap map[string]book) error

func (s *server) helperGetBooks(ctx context.Context, query string, argErr error, rowsFunc rowsFuncType, items ...interface{}) ([]book, error) {
	var bookIDs []string
	booksMap := make(map[string]book)

	rows, err := s.store.QueryContext(ctx, query, items...)
	if err != nil {
		return nil, fmt.Errorf("error getting all books, %v", err)
	}
	defer rows.Close()

	if err := rowsFunc(rows, &bookIDs, booksMap); err != nil {
		return nil, err
	}

	if len(booksMap) < 1 {
		return nil, argErr
	}

	var books []book

	query =
		`
			SELECT 
				bg.book_id, 
				g.genre 
			FROM genres g
			JOIN books_genres bg ON (bg.genre_id = g.id)
			WHERE bg.book_id = ANY($1);
		`

	genreRows, err := s.store.QueryContext(ctx, query, pq.Array(bookIDs))
	if err != nil {
		return nil, fmt.Errorf("error getting genres, %v", err)
	}

	defer genreRows.Close()

	for genreRows.Next() {
		row := struct {
			bookID string
			genre  string
		}{}

		if err := genreRows.Scan(&row.bookID, &row.genre); err != nil {
			return nil, fmt.Errorf("error scanning book genres, %v", err)
		}

		if b, ok := booksMap[row.bookID]; ok {
			b.genres = append(b.genres, row.genre)
			booksMap[row.bookID] = b
		}
	}

	query =
		`
			SELECT 
				book_id, 
				day, no_of_chapters 
			FROM release_schedule 
			WHERE book_id = ANY($1)
		`

	releaseScheduleRows, err := s.store.QueryContext(ctx, query, pq.Array(bookIDs))

	if err != nil {
		return nil, fmt.Errorf("error getting release schedule, %v", err)
	}

	defer releaseScheduleRows.Close()

	for releaseScheduleRows.Next() {
		releaseSchedule := releaseSchedule{}

		if err := releaseScheduleRows.Scan(&releaseSchedule.BookID, &releaseSchedule.Day, &releaseSchedule.Chapters); err != nil {
			return nil, fmt.Errorf("error scanning release schedule, %v", err)
		}

		if b, ok := booksMap[releaseSchedule.BookID]; ok {
			b.releaseSchedule = append(b.releaseSchedule, releaseSchedule)
			booksMap[releaseSchedule.BookID] = b
		}
	}

	for _, b := range booksMap {
		books = append(books, b)
	}

	return books, nil
}

func helpersGetBooksRows(rows *sql.Rows, bookIDs *[]string, booksMap map[string]book) error {
	for rows.Next() {
		var book book
		if err := rows.Scan(&book.id, &book.name, &book.description, &book.image, &book.views, &book.rating, &book.chapterCount); err != nil {
			return fmt.Errorf("error scanning rows, %v", err)
		}
		*bookIDs = append(*bookIDs, book.id)
		booksMap[book.id] = book
	}
	return nil
}
func (s *server) getBooksByGenre(ctx context.Context, genre []string, offset, limit int, sort, order string) ([]book, error) {
	query :=
		fmt.Sprintf(`
			SELECT 
				b.name, 
				b.description, 
				b.image, 
				b.views, 
				b.rating, 
				COUNT(c.id)
			FROM books b
			JOIN chapters c ON (b.id = c.book_id)
			JOIN books_genres bg ON (bg.book_id = b.id)
			JOIN genres g ON (g.id = bg.genre_id)
			WHERE 
				g.genre = ANY($1) 
				AND b.approved = true
			GROUP BY b.id
			ORDER BY %s %s
			OFFSET $2 LIMIT $3;
		`, helperSortField(sort), order)

	books, err := s.helperGetBooks(ctx, query, errNoBooksUnderGenre, helpersGetBooksRows, pq.Array(genre), offset, limit)
	if err != nil {
		return nil, err
	}

	return books, nil
}

func (s *server) getBooksByLanguage(ctx context.Context, language []string, offset, limit int, sort, order string) ([]book, error) {
	query :=
		fmt.Sprintf(`
			SELECT 
				b.id, 
				b.name, 
				b.description, 
				b.image, 
				b.views, 
				b.rating,
				COUNT(c.id)
			FROM books b
			JOIN chapters c ON (b.id = c.book_id)
			WHERE 
				b.language = ANY($1::language_type[]) 
				AND b.approved = true
			GROUP BY b.id
			ORDER BY %s %s 
			OFFSET $2 LIMIT $3;
		`, helperSortField(sort), order)

	books, err := s.helperGetBooks(ctx, query, errNoBooksUnderLanguage, helpersGetBooksRows, pq.Array(language), offset, limit)
	if err != nil {
		return nil, err
	}

	return books, nil
}

func (s *server) getBooksByGenreAndLanguage(ctx context.Context, genre []string, language []string, offset, limit int, sort, order string) ([]book, error) {
	query :=
		fmt.Sprintf(`
			SELECT 
				b.id, 
				b.name, 
				b.description, 
				b.image, 
				b.views, 
				b.rating,
				COUNT(c.id)
			FROM books b
			JOIN chapters c ON (b.id = c.book_id)
			JOIN books_genres bg ON (bg.book_id = b.id)
			JOIN genres g ON (g.id = bg.genre_id)
			WHERE 
				b.language = ANY($1::language_type[]) 
				AND g.genre = ANY($2) 
				AND b.approved = true
			GROUP BY b.id
			ORDER BY %s %s
			OFFSET $3 LIMIT $4;
		`, helperSortField(sort), order)

	books, err := s.helperGetBooks(ctx, query, errNoBooksUnderGenreAndLanguage, helpersGetBooksRows, pq.Array(language), pq.Array(genre), offset, limit)
	if err != nil {
		return nil, err
	}

	return books, nil
}

func (s *server) getAllBooks(ctx context.Context, offset, limit int, sort, order string) ([]book, error) {
	query :=
		fmt.Sprintf(`
			SELECT 
				b.id, 
				b.name, 
				b.description, 
				b.image, 
				b.views, 
				b.rating,
				COUNT(c.id)
			FROM books b
			JOIN chapters c ON (b.id = c.book_id)
			WHERE b.approved = true
			GROUP BY b.id
			ORDER BY %s %s
			OFFSET $1 LIMIT $2;
		`, helperSortField(sort), order)

	books, err := s.helperGetBooks(ctx, query, nil, helpersGetBooksRows, offset, limit)
	if err != nil {
		return nil, err
	}

	return books, nil
}

func helperGetBooksStatsRows(rows *sql.Rows, bookIDs *[]string, booksMap map[string]book) error {
	for rows.Next() {
		var book book
		if err := rows.Scan(&book.id, &book.name, &book.description, &book.image, &book.views, &book.rating, &book.language, &book.completed, &book.approved, &book.createdAt, &book.updatedAt, &book.chapterCount); err != nil {
			return fmt.Errorf("error scanning rows, %v", err)
		}
		*bookIDs = append(*bookIDs, book.id)
		booksMap[book.id] = book
	}
	return nil
}
func (s *server) getBooksStats(ctx context.Context, id string, offset, limit int) ([]book, error) {
	query :=
		`
			SELECT 
				b.id, 
				b.name, 
				b.description, 
				b.image, 
				b.views,
				b.rating, 
				b.language,
				b.completed, 
				b.approved, 
				b.created_at, 
				b.updated_at, 
				COUNT(c.id) AS chapter_count
			FROM books b 
			JOIN chapters c ON (c.book_id = b.id)
			WHERE b.author_id = $1
			GROUP BY b.id
			ORDER BY b.created_at DESC
			OFFSET $2 LIMIT $3;
		`

	books, err := s.helperGetBooks(ctx, query, errUserHasNoBooks, helperGetBooksStatsRows, id, offset, limit)
	if err != nil {
		return nil, err
	}

	return books, nil
}

func (s *server) getRecentlyReadBooks(ctx context.Context, userID string, offset, limit int) ([]recentBook, error) {
	var books []recentBook

	query :=
		`
			SELECT
				b.name,
				b.image,
				rb.chapter,
				rb.updated_at
			FROM recent_books rb
			JOIN books b ON (b.id = rb.book_id)
			JOIN users u ON (u.id = rb.user_id)
			WHERE u.id = $1
			ORDER BY rb.updated_at DESC
			OFFSET $2 LIMIT $3;
		`

	rows, err := s.store.QueryContext(ctx, query, userID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("error getting recently read books, %v", err)
	}

	for rows.Next() {
		var book recentBook
		if err := rows.Scan(&book.name, &book.image, &book.lastReadChapter, &book.updatedAt); err != nil {
			return nil, fmt.Errorf("error scanning recently read books, %v", err)
		}
		books = append(books, book)
	}

	return books, nil
}
