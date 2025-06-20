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
var ErrNoBooksUnderThisGenre = errors.New("no books under this genre yet")
var ErrNoBooksUnderThisLanguage = errors.New("no books under this language yet")
var ErrNoBooksUnderThisGenreOrLanguage = errors.New("no books under this genre or language yet")

func (s *PostgresStore) GetGenresAndReleaseSchedules(ctx context.Context, bookIDs *[]uuid.UUID, booksMap map[uuid.UUID]*models.Book) (*[]models.Book, error) {
	var books []models.Book

	rows1, err := s.DB.QueryContext(ctx, `
			SELECT bg.book_id, g.genres 
			FROM genres g
			JOIN books_genres bg ON (bg.genre_id = g.id)
			WHERE bg.book_id = ANY($1);
		`, pq.Array(*bookIDs))

	if err != nil {
		return nil, fmt.Errorf("error getting genres: %v", err)
	}

	defer rows1.Close()

	for rows1.Next() {
		genre := struct {
			bookId uuid.UUID
			genre  string
		}{}

		if err := rows1.Scan(&genre.bookId, &genre.genre); err != nil {
			return nil, fmt.Errorf("error scanning book genres: %v", err)
		}

		if b, ok := booksMap[genre.bookId]; ok {
			b.Genres = append(b.Genres, genre.genre)
		}
	}

	rows2, err := s.DB.QueryContext(ctx, `
			SELECT book_id, day, no_of_chapters FROM release_schedule WHERE book_id = ANY($1)
		`, pq.Array(*bookIDs))

	if err != nil {
		return nil, fmt.Errorf("error getting release schedule: %v", err)
	}

	defer rows2.Close()

	for rows2.Next() {
		release_schedule := models.Schedule{}

		if err := rows2.Scan(&release_schedule.BookId, &release_schedule.Day, &release_schedule.Chapters); err != nil {
			return nil, fmt.Errorf("error scanning release schedule: %v", err)
		}

		if b, ok := booksMap[release_schedule.BookId]; ok {
			b.Release_schedule = append(b.Release_schedule, release_schedule)
		}
	}

	for _, b := range booksMap {
		books = append(books, *b)
	}

	return &books, nil
}

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

	var rows *sql.Rows
	rows, err = tx.QueryContext(ctx, `
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
		err = ErrGenresNotFound
		return err
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
		return nil, fmt.Errorf("error retrieving books: %v", err)
	}

	defer rows1.Close()

	if !rows1.Next() {
		return nil, ErrCreatorsBooksNotFound
	}

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

	books, err := s.GetGenresAndReleaseSchedules(ctx, &bookIds, booksMap)

	if err != nil {
		return nil, err
	}

	return books, nil
}

func (s *PostgresStore) GetBooksByGenre(ctx context.Context, genre []string) (*[]models.Book, error) {
	rows1, err := s.DB.QueryContext(ctx, `
			SELECT b.id, b.name, b.description, b.image, b.views, b.rating,
			COUNT(c.id)
			FROM books b
			JOIN chapters c ON (b.id = c.book_id)
			JOIN books_genres bg ON (bg.book_id = b.id)
			JOIN genres g ON (g.id = bg.genre_id)
			WHERE g.genres = ANY($1)
			GROUP BY b.id;
		`, pq.Array(genre))

	if err != nil {
		return nil, fmt.Errorf("error getting books by genre: %v", err)
	}

	defer rows1.Close()

	if !rows1.Next() {
		return nil, ErrNoBooksUnderThisGenre
	}

	var bookIDs []uuid.UUID
	booksMap := map[uuid.UUID]*models.Book{}

	for rows1.Next() {
		var book models.Book
		if err := rows1.Scan(
			&book.Id,
			&book.Name,
			&book.Description,
			&book.Image,
			&book.Views,
			&book.Rating,
			&book.No_Of_Chapters,
		); err != nil {
			return nil, fmt.Errorf("error scanning books: %v", err)
		}

		bookIDs = append(bookIDs, book.Id)
		booksMap[book.Id] = &book
	}

	books, err := s.GetGenresAndReleaseSchedules(ctx, &bookIDs, booksMap)

	if err != nil {
		return nil, err
	}

	return books, nil
}

func (s *PostgresStore) GetBooksByLanguage(ctx context.Context, language []string) (*[]models.Book, error) {
	rows, err := s.DB.QueryContext(ctx, `
			SELECT b.id, b.name, b.description, b.image, b.views, b.rating,
			COUNT(c.id)
			FROM books b
			JOIN chapters c ON (b.id = c.book_id)
			WHERE b.language = ANY($1::languages[])
			GROUP BY b.id;
		`, pq.Array(language))

	if err != nil {
		return nil, fmt.Errorf("error getting books by language: %v", err)
	}

	defer rows.Close()

	if !rows.Next() {
		return nil, ErrNoBooksUnderThisLanguage
	}

	var bookIDs []uuid.UUID
	booksMap := make(map[uuid.UUID]*models.Book)

	for rows.Next() {
		var book models.Book
		if err := rows.Scan(
			&book.Id,
			&book.Name,
			&book.Description,
			&book.Image,
			&book.Views,
			&book.Rating,
			&book.No_Of_Chapters,
		); err != nil {
			return nil, fmt.Errorf("error scanning books: %v", err)
		}

		bookIDs = append(bookIDs, book.Id)
		booksMap[book.Id] = &book
	}

	books, err := s.GetGenresAndReleaseSchedules(ctx, &bookIDs, booksMap)

	if err != nil {
		return nil, err
	}

	return books, nil
}

func (s *PostgresStore) GetBooksByGenreAndLanguage(ctx context.Context, genre []string, language []string) (*[]models.Book, error) {
	rows, err := s.DB.QueryContext(ctx, `
			SELECT b.id, b.name, b.description, b.image, b.views, b.rating,
			COUNT(c.id)
			FROM books b
			JOIN chapters c ON (b.id = c.book_id)
			JOIN books_genres bg ON (bg.book_id = b.id)
			JOIN genres g ON (g.id = bg.genre_id)
			WHERE b.language = ANY($1) OR g.genres = ANY($2)
			GROUP BY b.id;
		`, pq.Array(language), pq.Array(genre))

	if err != nil {
		return nil, fmt.Errorf("error getting books by genre and language: %v", err)
	}

	defer rows.Close()

	if !rows.Next() {
		return nil, ErrNoBooksUnderThisGenreOrLanguage
	}

	var bookIDs []uuid.UUID
	booksMap := make(map[uuid.UUID]*models.Book)

	for rows.Next() {
		var book models.Book
		if err := rows.Scan(
			&book.Id,
			&book.Name,
			&book.Description,
			&book.Image,
			&book.Views,
			&book.Rating,
			&book.No_Of_Chapters,
		); err != nil {
			return nil, fmt.Errorf("error scanning books: %v", err)
		}

		bookIDs = append(bookIDs, book.Id)
		booksMap[book.Id] = &book
	}

	books, err := s.GetGenresAndReleaseSchedules(ctx, &bookIDs, booksMap)

	if err != nil {
		return nil, err
	}

	return books, nil
}

func (s *PostgresStore) GetAllBooks(ctx context.Context) (*[]models.Book, error) {
	rows, err := s.DB.QueryContext(ctx, `
			SELECT b.id, b.name, b.description, b.image, b.views, b.rating,
			COUNT(c.id)
			FROM books b
			JOIN chapters c ON (b.id = c.book_id)
			GROUP BY b.id;
		`)

	if err != nil {
		return nil, fmt.Errorf("error getting all books: %v", err)
	}

	defer rows.Close()

	var bookIDs []uuid.UUID
	booksMap := make(map[uuid.UUID]*models.Book)

	for rows.Next() {
		var book models.Book
		if err := rows.Scan(
			&book.Id,
			&book.Name,
			&book.Description,
			&book.Image,
			&book.Views,
			&book.Rating,
			&book.No_Of_Chapters,
		); err != nil {
			return nil, fmt.Errorf("error scanning books: %v", err)
		}

		bookIDs = append(bookIDs, book.Id)
		booksMap[book.Id] = &book
	}

	books, err := s.GetGenresAndReleaseSchedules(ctx, &bookIDs, booksMap)

	if err != nil {
		return nil, err
	}

	return books, nil
}
