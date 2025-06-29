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

var ErrGenresNotFound = errors.New("genres not found")
var ErrCreatorsBooksNotFound = errors.New("creator doesn't have any books")
var ErrNoBooksUnderThisGenre = errors.New("no books under this genre yet")
var ErrNoBooksUnderThisLanguage = errors.New("no books under this language yet")
var ErrNoBooksUnderThisGenreOrLanguage = errors.New("no books under this genre or language yet")
var ErrBookNotFound = errors.New("book not found")
var ErrShouldAtLeasePassOneFieldToUpdate = errors.New("one field at least is required to update")

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

func (s *PostgresStore) UploadBook(ctx context.Context, book *models.Book) (string, error) {
	tx, err := s.DB.Begin()

	if err != nil {
		return "", fmt.Errorf("error starting transaction: %v", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var bookID string

	err = tx.QueryRowContext(ctx, `
			INSERT INTO books (name, description, author_id, language)
			VALUES ($1, $2, $3, $4) RETURNING id;
		`, book.Name, book.Description, book.Author_Id, book.Language).Scan(&bookID)

	if err != nil {
		return "", fmt.Errorf("error inserting into book table: %v", err)
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
		return "", fmt.Errorf("error inserting release_schedule: %v", err)
	}

	var genreIDs []string

	var rows *sql.Rows
	rows, err = tx.QueryContext(ctx, `
			SELECT id FROM genres WHERE genres = ANY($1);
		`, pq.Array(book.Genres))

	if err != nil {
		return "", fmt.Errorf("error retrieving genre ids: %v", err)
	}

	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", fmt.Errorf("error scanning genre ids: %v", err)
		}
		genreIDs = append(genreIDs, id)
	}

	if len(genreIDs) != len(book.Genres) {
		err = ErrGenresNotFound
		return "", err
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
		return "", fmt.Errorf("error inserting into book_genres: %v", err)
	}

	_, err = tx.ExecContext(ctx, `
			INSERT INTO chapters(title, content, book_id)
			VALUES ($1, $2, $3);
		`, book.Chapter_Draft.Title, book.Chapter_Draft.Content, bookID)

	if err != nil {
		return "", fmt.Errorf("error inserting draft chapter: %v", err)
	}

	err = tx.Commit()

	if err != nil {
		return "", err
	}

	return bookID, nil
}

func (s *PostgresStore) UpdateBookImage(ctx context.Context, url string, id string) error {
	_, err := s.DB.ExecContext(ctx, `
			UPDATE books
			SET image = $1
			WHERE id = $2;
		`, url, id)

	if err != nil {
		return fmt.Errorf("error updating book image: %v", err)
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

	if len(booksMap) < 1 {
		return nil, ErrCreatorsBooksNotFound
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
			GROUP BY b.id
			ORDER BY b.views DESC;
		`, pq.Array(genre))

	if err != nil {
		return nil, fmt.Errorf("error getting books by genre: %v", err)
	}

	defer rows1.Close()

	var bookIDs []uuid.UUID
	booksMap := map[uuid.UUID]*models.Book{}

	for rows1.Next() {
		book := models.Book{}
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

	if len(booksMap) < 1 {
		return nil, ErrNoBooksUnderThisGenre
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
			GROUP BY b.id
			ORDER BY b.views DESC;
		`, pq.Array(language))

	if err != nil {
		return nil, fmt.Errorf("error getting books by language: %v", err)
	}

	defer rows.Close()

	var bookIDs []uuid.UUID
	booksMap := map[uuid.UUID]*models.Book{}

	for rows.Next() {
		book := models.Book{}
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

	if len(booksMap) < 1 {
		return nil, ErrNoBooksUnderThisLanguage
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
			WHERE b.language = ANY($1::languages[]) AND g.genres = ANY($2)
			GROUP BY b.id;
		`, pq.Array(language), pq.Array(genre))

	if err != nil {
		return nil, fmt.Errorf("error getting books by genre and language: %v", err)
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

	if len(booksMap) < 1 {
		return nil, ErrNoBooksUnderThisGenreOrLanguage
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
			GROUP BY b.id
			ORDER BY b.views DESC;
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

func (s *PostgresStore) GetBook(ctx context.Context, id string) (*models.Book, error) {
	book := &models.Book{}

	err := s.DB.QueryRowContext(ctx, `
			SELECT b.id, b.name, b.description, b.image, b.views, b.rating, b.language, b.completed, b.created_at,
			u.name,
			COUNT (c.id)
			FROM books b
			JOIN users u ON (u.id = b.author_id)
			JOIN chapters c ON (c.book_id = b.id)	
			WHERE b.id = $1
			GROUP BY b.id, u.name;
		`, id).Scan(
		&book.Id,
		&book.Name,
		&book.Description,
		&book.Image,
		&book.Views,
		&book.Rating,
		&book.Language,
		&book.Completed,
		&book.Created_at,
		&book.Author_name,
		&book.No_Of_Chapters,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrBookNotFound
		}

		return nil, fmt.Errorf("error scanning book: %v", err)
	}

	rows1, err := s.DB.QueryContext(ctx, `
			SELECT g.genres 
			FROM genres g
			JOIN books_genres bg ON (bg.genre_id = g.id)
			WHERE bg.book_id = $1;
		`, book.Id)

	if err != nil {
		return nil, fmt.Errorf("error getting genres: %v", err)
	}

	defer rows1.Close()

	for rows1.Next() {
		var genre string

		if err := rows1.Scan(&genre); err != nil {
			return nil, fmt.Errorf("error scanning genre: %v", err)
		}

		book.Genres = append(book.Genres, genre)
	}

	rows2, err := s.DB.QueryContext(ctx, `
			SELECT day, no_of_chapters FROM release_schedule WHERE book_id = $1;
		`, book.Id)

	if err != nil {
		return nil, fmt.Errorf("error getting release schedule: %v", err)
	}

	defer rows2.Close()

	for rows2.Next() {
		var schedule models.Schedule

		if err := rows2.Scan(
			&schedule.Day,
			&schedule.Chapters,
		); err != nil {
			return nil, fmt.Errorf("error scanning release schedule: %v", err)
		}

		book.Release_schedule = append(book.Release_schedule, schedule)
	}

	rows3, err := s.DB.QueryContext(ctx, `
			SELECT title, created_at FROM chapters WHERE book_id = $1;
		`, book.Id)

	if err != nil {
		return nil, fmt.Errorf("error getting chapters: %v", err)
	}

	defer rows3.Close()

	for rows3.Next() {
		var chapter models.Chapter

		if err := rows3.Scan(
			&chapter.Title,
			&chapter.Created_at,
		); err != nil {
			return nil, fmt.Errorf("error scanning chapters: %v", err)
		}

		book.Chapters = append(book.Chapters, chapter)
	}

	return book, nil
}

func (s *PostgresStore) DeleteBook(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM books WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("error deleting book: %v", err)
	}
	return nil
}

func (s *PostgresStore) EditBook(ctx context.Context, book *models.HandleEditBookParam) error {
	index := 0
	clauses := []string{}
	arguments := []interface{}{}

	if book.Name != "" {
		index++
		clauses = append(clauses, fmt.Sprintf("name=$%d", index))
		arguments = append(arguments, book.Name)
	}

	if book.Description != "" {
		index++
		clauses = append(clauses, fmt.Sprintf("description=$%d", index))
		arguments = append(arguments, book.Description)
	}

	if book.Image != "" {
		index++
		clauses = append(clauses, fmt.Sprintf("image=$%d", index))
		arguments = append(arguments, book.Image)
	}

	arguments = append(arguments, book.Id)

	if len(clauses) < 1 || len(book.Release_schedule) < 1 || len(book.Genres) < 1 {
		return ErrShouldAtLeasePassOneFieldToUpdate
	}

	tx, err := s.DB.Begin()

	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if len(clauses) > 0 {
		_, err = tx.ExecContext(ctx, fmt.Sprintf(`
			UPDATE books
			SET %v
			WHERE id = $%d;
			`, strings.Join(clauses, ","), index+1), arguments...)

		if err != nil {
			return fmt.Errorf("error updating book: %v", err)
		}
	}

	if len(book.Release_schedule) > 0 {
		_, err = tx.ExecContext(ctx, "DELETE FROM release_schedule WHERE book_id = $1", book.Id)

		clauses = []string{}
		arguments = []interface{}{}
		index = 1

		for _, sched := range book.Release_schedule {
			clauses = append(clauses, fmt.Sprintf("($%d, $%d, $%d)", index, index+1, index+2))
			arguments = append(arguments, book.Id, sched.Day, sched.Chapters)
			index += 3
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO release_schedule(book_id, day, no_of_chapters)
			VALUES %s; 
			`, strings.Join(clauses, ",")), arguments...)

		if err != nil {
			return fmt.Errorf("error inserting release_schedule: %v", err)
		}
	}

	if len(book.Genres) > 0 {

		_, err = tx.ExecContext(ctx, "DELETE FROM books_genres WHERE book_id = $1", book.Id)

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

		clauses = []string{}
		arguments = []interface{}{}
		index = 1

		for _, genreId := range genreIDs {
			clauses = append(clauses, fmt.Sprintf("($%d, $%d)", index, index+1))
			arguments = append(arguments, book.Id, genreId)
			index += 2
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO books_genres(book_id, genre_id)
			VALUES %s
			ON CONFLICT DO NOTHING;
			`, strings.Join(clauses, ",")), arguments...)

		if err != nil {
			return fmt.Errorf("error inserting into book_genres: %v", err)
		}
	}

	return tx.Commit()
}

func (s *PostgresStore) ApproveBook(ctx context.Context, id string, approve bool) error {
	if _, err := s.DB.ExecContext(ctx, `
			UPDATE books
			SET approved = $1
			WHERE id = $2;
		`, approve, id); err != nil {
		return fmt.Errorf("error approving book: %v", err)
	}
	return nil
}

func (s *PostgresStore) MarkBookAsComplete(ctx context.Context, id string, complete bool) error {
	if _, err := s.DB.ExecContext(ctx, `
			UPDATE books
			SET completed = $1
			WHERE id = $2;
		`, complete, id); err != nil {
		return fmt.Errorf("error marking book as complete: %v", err)
	}

	return nil
}
