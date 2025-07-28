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

var (
	ErrGenresNotFound                    = errors.New("genres not found")
	ErrCreatorsBooksNotFound             = errors.New("creator doesn't have any books")
	ErrNoBooksUnderThisGenre             = errors.New("no books under this genre yet")
	ErrNoBooksUnderThisLanguage          = errors.New("no books under this language yet")
	ErrNoBooksUnderThisGenreAndLanguage  = errors.New("no books under this genre and language yet")
	ErrBookNotFound                      = errors.New("book not found")
	ErrShouldAtLeasePassOneFieldToUpdate = errors.New("one field at least is required to update")
	ErrNoBooksInRecents                  = errors.New("no books in recents yet")
)

func (s *PostgresStore) GetGenresAndReleaseSchedules(ctx context.Context, bookIDs *[]uuid.UUID, booksMap map[uuid.UUID]*models.Book) ([]models.Book, error) {
	var books []models.Book

	query := `
			SELECT bg.book_id, g.genres 
			FROM genres g
			JOIN books_genres bg ON (bg.genre_id = g.id)
			WHERE bg.book_id = ANY($1);
	`

	genres, err := s.DB.QueryContext(ctx, query, pq.Array(*bookIDs))

	if err != nil {
		return nil, fmt.Errorf("error getting genres: %v", err)
	}

	defer genres.Close()

	for genres.Next() {
		genre := struct {
			bookId uuid.UUID
			genre  string
		}{}

		if err := genres.Scan(&genre.bookId, &genre.genre); err != nil {
			return nil, fmt.Errorf("error scanning book genres: %v", err)
		}

		if b, ok := booksMap[genre.bookId]; ok {
			b.Genres = append(b.Genres, genre.genre)
		}
	}

	query = `
			SELECT book_id, day, no_of_chapters FROM release_schedule WHERE book_id = ANY($1)
	`

	release_schedules, err := s.DB.QueryContext(ctx, query, pq.Array(*bookIDs))

	if err != nil {
		return nil, fmt.Errorf("error getting release schedule: %v", err)
	}

	defer release_schedules.Close()

	for release_schedules.Next() {
		release_schedule := models.Schedule{}

		if err := release_schedules.Scan(&release_schedule.BookId, &release_schedule.Day, &release_schedule.Chapters); err != nil {
			return nil, fmt.Errorf("error scanning release schedule: %v", err)
		}

		if b, ok := booksMap[release_schedule.BookId]; ok {
			b.Release_schedule = append(b.Release_schedule, release_schedule)
		}
	}

	for _, b := range booksMap {
		books = append(books, *b)
	}

	return books, nil
}

func (s *PostgresStore) UploadBook(ctx context.Context, book *models.Book) (*uuid.UUID, error) {
	tx, err := s.DB.Begin()

	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %v", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var bookID uuid.UUID

	query := `
			INSERT INTO books (name, description, author_id, language)
			VALUES ($1, $2, $3, $4) RETURNING id;
	`

	err = tx.QueryRowContext(ctx, query, book.Name, book.Description, book.Author_Id, book.Language).Scan(&bookID)

	if err != nil {
		return nil, fmt.Errorf("error inserting into book table: %v", err)
	}

	valueStrings := []string{}
	valueArgs := []interface{}{}
	argPosition := 1

	for _, sched := range book.Release_schedule {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", argPosition, argPosition+1, argPosition+2))
		valueArgs = append(valueArgs, bookID, sched.Day, sched.Chapters)
		argPosition += 3
	}

	query = fmt.Sprintf("INSERT INTO release_schedule(book_id, day, no_of_chapters) VALUES %s;", strings.Join(valueStrings, ","))

	_, err = tx.ExecContext(ctx, query, valueArgs...)

	if err != nil {
		return nil, fmt.Errorf("error inserting release_schedule: %v", err)
	}

	var genreIDs []string

	var rows *sql.Rows

	query = `
		SELECT id FROM genres WHERE genres = ANY($1);
	`
	rows, err = tx.QueryContext(ctx, query, pq.Array(book.Genres))

	if err != nil {
		return nil, fmt.Errorf("error retrieving genre ids: %v", err)
	}

	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("error scanning genre ids: %v", err)
		}
		genreIDs = append(genreIDs, id)
	}

	if len(genreIDs) != len(book.Genres) {
		err = ErrGenresNotFound
		return nil, err
	}

	valueStrings = []string{}
	valueArgs = []interface{}{}
	argPosition = 1

	for _, genreId := range genreIDs {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", argPosition, argPosition+1))
		valueArgs = append(valueArgs, bookID, genreId)
		argPosition += 2
	}

	query = fmt.Sprintf("INSERT INTO books_genres(book_id, genre_id) VALUES %s ON CONFLICT DO NOTHING;", strings.Join(valueStrings, ","))

	_, err = tx.ExecContext(ctx, query, valueArgs...)

	if err != nil {
		return nil, fmt.Errorf("error inserting into book_genres: %v", err)
	}

	query = `
			INSERT INTO chapters(chapter_no, title, content, book_id)
			VALUES (0, $1, $2, $3);
	`

	_, err = tx.ExecContext(ctx, query, book.Chapter_Draft.Title, book.Chapter_Draft.Content, bookID)

	if err != nil {
		return nil, fmt.Errorf("error inserting draft chapter: %v", err)
	}

	err = tx.Commit()

	if err != nil {
		return nil, err
	}

	return &bookID, nil
}

func (s *PostgresStore) UpdateBookImage(ctx context.Context, url string, id string) error {
	query := `
			UPDATE books
			SET image = $1
			WHERE id = $2;
	`
	_, err := s.DB.ExecContext(ctx, query, url, id)

	if err != nil {
		return fmt.Errorf("error updating book image: %v", err)
	}

	return nil
}

func (s *PostgresStore) GetBooksStats(ctx context.Context, id string, offset int, limit int) ([]models.Book, error) {
	booksMap := make(map[uuid.UUID]*models.Book)

	query := `
			SELECT b.id, b.name, b.description, b.image, b.views, b.language, b.completed, b.approved, b.created_at, b.updated_at, 
						 COUNT(c.id) AS chapter_count
			FROM books b 
			JOIN chapters c ON (c.book_id = b.id)
			WHERE b.author_id = $1
			GROUP BY b.id
			ORDER BY b.created_at DESC
			OFFSET $2 LIMIT $3;
	`

	rows, err := s.DB.QueryContext(ctx, query, id, offset, limit)

	if err != nil {
		return nil, fmt.Errorf("error retrieving books: %v", err)
	}

	defer rows.Close()

	var bookIds []uuid.UUID

	for rows.Next() {
		var book models.Book

		if err := rows.Scan(
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

func sortByFieldHelper(sort string) string {
	if sort == "updated" {
		sort = "b.updated_at"
	} else {
		sort = "b.views"
	}

	return sort
}

func (s *PostgresStore) GetBooksByGenre(ctx context.Context, genre []string, offset int, limit int, sort string, order string) ([]models.Book, error) {
	field := sortByFieldHelper(sort)

	query := fmt.Sprintf(`
			SELECT b.id, b.name, b.description, b.image, b.views, b.rating,
			COUNT(c.id)
			FROM books b
			JOIN chapters c ON (b.id = c.book_id)
			JOIN books_genres bg ON (bg.book_id = b.id)
			JOIN genres g ON (g.id = bg.genre_id)
			WHERE g.genres = ANY($1) AND b.approved = true
			GROUP BY b.id
			ORDER BY %s %s
			OFFSET $2 LIMIT $3;
		`, field, order)

	rows, err := s.DB.QueryContext(ctx, query, pq.Array(genre), offset, limit)

	if err != nil {
		return nil, fmt.Errorf("error getting books by genre: %v", err)
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
		return nil, ErrNoBooksUnderThisGenre
	}

	books, err := s.GetGenresAndReleaseSchedules(ctx, &bookIDs, booksMap)

	if err != nil {
		return nil, err
	}

	return books, nil
}

func (s *PostgresStore) GetBooksByLanguage(ctx context.Context, language []string, offset int, limit int, sort string, order string) ([]models.Book, error) {
	field := sortByFieldHelper(sort)

	query := fmt.Sprintf(`
			SELECT b.id, b.name, b.description, b.image, b.views, b.rating,
			COUNT(c.id)
			FROM books b
			JOIN chapters c ON (b.id = c.book_id)
			WHERE b.language = ANY($1::languages[]) AND b.approved = true
			GROUP BY b.id
			ORDER BY %s %s 
			OFFSET $2 LIMIT $3;
		`, field, order)

	rows, err := s.DB.QueryContext(ctx, query, pq.Array(language), offset, limit)

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

func (s *PostgresStore) GetBooksByGenreAndLanguage(ctx context.Context, genre []string, language []string, offset int, limit int, sort string, order string) ([]models.Book, error) {
	field := sortByFieldHelper(sort)

	query := fmt.Sprintf(`
			SELECT b.id, b.name, b.description, b.image, b.views, b.rating,
			COUNT(c.id)
			FROM books b
			JOIN chapters c ON (b.id = c.book_id)
			JOIN books_genres bg ON (bg.book_id = b.id)
			JOIN genres g ON (g.id = bg.genre_id)
			WHERE b.language = ANY($1::languages[]) AND g.genres = ANY($2) AND b.approved = true
			GROUP BY b.id
			ORDER BY %s %s
			OFFSET $3 LIMIT $4;
		`, field, order)

	rows, err := s.DB.QueryContext(ctx, query, pq.Array(language), pq.Array(genre), offset, limit)

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
		return nil, ErrNoBooksUnderThisGenreAndLanguage
	}

	books, err := s.GetGenresAndReleaseSchedules(ctx, &bookIDs, booksMap)

	if err != nil {
		return nil, err
	}

	return books, nil
}

func (s *PostgresStore) GetAllBooks(ctx context.Context, offset int, limit int, sort string, order string) ([]models.Book, error) {
	field := sortByFieldHelper(sort)

	query := fmt.Sprintf(`
			SELECT b.id, b.name, b.description, b.image, b.views, b.rating,
			COUNT(c.id)
			FROM books b
			JOIN chapters c ON (b.id = c.book_id)
			WHERE b.approved = true
			GROUP BY b.id
			ORDER BY %s %s
			OFFSET $1 LIMIT $2;
		`, field, order)

	rows, err := s.DB.QueryContext(ctx, query, offset, limit)

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

	query := `
			SELECT b.id, b.name, b.description, b.image, b.views, b.rating, b.language, b.completed, b.created_at,
			u.username,
			COUNT (c.id)
			FROM books b
			JOIN users u ON (u.id = b.author_id)
			JOIN chapters c ON (c.book_id = b.id)
			WHERE b.id = $1 AND b.approved = true
			GROUP BY b.id, u.username;
	`

	err := s.DB.QueryRowContext(ctx, query, id).Scan(
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

	query = `
			SELECT g.genres 
			FROM genres g
			JOIN books_genres bg ON (bg.genre_id = g.id)
			WHERE bg.book_id = $1;
	`

	genres, err := s.DB.QueryContext(ctx, query, book.Id)

	if err != nil {
		return nil, fmt.Errorf("error getting genres: %v", err)
	}

	defer genres.Close()

	for genres.Next() {
		var genre string

		if err := genres.Scan(&genre); err != nil {
			return nil, fmt.Errorf("error scanning genre: %v", err)
		}

		book.Genres = append(book.Genres, genre)
	}

	query = `
			SELECT day, no_of_chapters FROM release_schedule WHERE book_id = $1;
	`

	release_schedules, err := s.DB.QueryContext(ctx, query, book.Id)

	if err != nil {
		return nil, fmt.Errorf("error getting release schedule: %v", err)
	}

	defer release_schedules.Close()

	for release_schedules.Next() {
		var schedule models.Schedule

		if err := release_schedules.Scan(
			&schedule.Day,
			&schedule.Chapters,
		); err != nil {
			return nil, fmt.Errorf("error scanning release schedule: %v", err)
		}

		book.Release_schedule = append(book.Release_schedule, schedule)
	}

	query = `
			SELECT title, chapter_no, created_at FROM chapters WHERE book_id = $1;
	`

	chapters, err := s.DB.QueryContext(ctx, query, book.Id)

	if err != nil {
		return nil, fmt.Errorf("error getting chapters: %v", err)
	}

	defer chapters.Close()

	for chapters.Next() {
		var chapter models.Chapter

		if err := chapters.Scan(
			&chapter.Title,
			&chapter.Chapter_no,
			&chapter.Created_at,
		); err != nil {
			return nil, fmt.Errorf("error scanning chapters: %v", err)
		}

		book.Chapters = append(book.Chapters, chapter)
	}

	return book, nil
}

func (s *PostgresStore) DeleteBook(ctx context.Context, bookId string, userId string) error {
	query := `
			DELETE FROM books WHERE id = $1 
			AND 
			(author_id = $2 OR EXISTS(
				SELECT 1 
				FROM users_roles ur 
				JOIN roles r ON (ur.role_id = r.id)
				WHERE ur.user_id = $2 AND r.name = 'admin'
			));
	`

	_, err := s.DB.ExecContext(ctx, query, bookId, userId)
	if err != nil {
		return fmt.Errorf("error deleting book: %v", err)
	}
	return nil
}

func (s *PostgresStore) EditBook(ctx context.Context, book *models.HandleEditBookParam, userId string) error {
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

	if len(clauses) < 1 && len(book.Release_schedule) < 1 && len(book.Genres) < 1 {
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

	query := fmt.Sprintf(`UPDATE books SET %v WHERE id = $%d AND author_id = $%d;`, strings.Join(clauses, ","), index+1, index+2)

	if len(clauses) > 0 {
		_, err = tx.ExecContext(ctx, query, arguments...)

		if err != nil {
			return fmt.Errorf("error updating book: %v", err)
		}
	}

	if len(book.Release_schedule) > 0 {
		query = `
				DELETE FROM release_schedule WHERE book_id = $1;
		`

		_, err = tx.ExecContext(ctx, query, book.Id)

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
		query = `
				DELETE FROM books_genres WHERE book_id = $1;
		`

		_, err = tx.ExecContext(ctx, query, book.Id)

		var genreIDs []string

		var rows *sql.Rows

		query := `
				SELECT id FROM genres WHERE genres = ANY($1);
		`

		rows, err = tx.QueryContext(ctx, query, pq.Array(book.Genres))

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

		query = fmt.Sprintf(`
				INSERT INTO books_genres(book_id, genre_id)
				VALUES %s
				ON CONFLICT DO NOTHING;
				`, strings.Join(clauses, ","))

		_, err = tx.ExecContext(ctx, query, arguments...)

		if err != nil {
			return fmt.Errorf("error inserting into book_genres: %v", err)
		}
	}

	return tx.Commit()
}

func (s *PostgresStore) ApproveBook(ctx context.Context, id string, approve bool) error {
	query := `
			UPDATE books
			SET approved = $1
			WHERE id = $2;
	`

	if _, err := s.DB.ExecContext(ctx, query, approve, id); err != nil {
		return fmt.Errorf("error approving book: %v", err)
	}
	return nil
}

func (s *PostgresStore) MarkBookAsComplete(ctx context.Context, id string, complete bool) error {
	query := `
			UPDATE books
			SET completed = $1
			WHERE id = $2;
	`

	if _, err := s.DB.ExecContext(ctx, query, complete, id); err != nil {
		return fmt.Errorf("error marking book as complete: %v", err)
	}

	return nil
}

func (s *PostgresStore) GetRecentReads(ctx context.Context, id string, offset int, limit int) ([]models.Book, error) {
	var books []models.Book

	query := `
			SELECT b.name, b.image,
			r.chapter_no, r.updated_at
			FROM recents r
			JOIN books b ON (r.book_id = b.id)
			WHERE r.user_id = $1
			OFFSET $2 LIMIT $3;
	`

	rows, err := s.DB.QueryContext(ctx, query, id, offset, limit)

	if err != nil {
		return nil, fmt.Errorf("error getting recent reads: %v", err)
	}

	for rows.Next() {
		var book models.Book

		if err := rows.Scan(&book.Name, &book.Image, &book.ChapterLastRead, &book.TimeLastOpened); err != nil {
			return nil, fmt.Errorf("error scanning books")
		}

		books = append(books, book)
	}

	if len(books) < 1 {
		return nil, ErrNoBooksInRecents
	}

	return books, nil
}
