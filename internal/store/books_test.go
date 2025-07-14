package store

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
)

func TestUploadBook(t *testing.T) {
	db := setUpTestDb(t)

	author_id, _ := db.CreateUser(context.TODO(), &models.User{
		Username: "fake_username",
		Email:    "fake_email@email.com",
		Password: "fake_password",
	})

	defer func() {
		db.DB.Exec("DELETE FROM users WHERE id = $1", author_id)
	}()

	tests := []struct {
		name    string
		book    *models.Book
		wantErr bool
	}{
		{
			name:    "should return an error if fields are missing",
			book:    &models.Book{Name: "test book"},
			wantErr: true,
		},
		{
			name: "should return an error if genre doesn't exist",
			book: &models.Book{
				Name:        "test book",
				Description: "test book description",
				Image: sql.NullString{
					String: "test book image",
					Valid:  true,
				},
				Author_Id: *author_id,
				Language:  "English",
				Release_schedule: []models.Schedule{
					{
						Day:      "Monday",
						Chapters: 1,
					},
					{
						Day:      "Tuesday",
						Chapters: 2,
					},
				},
				Genres: []string{"Action", "Non-existent genre"},
				Chapter_Draft: models.Chapter{
					Title:   "test book chapter",
					Content: "test book content",
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if author id doesn't exist",
			book: &models.Book{
				Name:        "test book",
				Description: "test book description",
				Image: sql.NullString{
					String: "test book image",
					Valid:  true,
				},
				Author_Id: uuid.New(),
				Language:  "English",
				Release_schedule: []models.Schedule{
					{
						Day:      "Monday",
						Chapters: 1,
					},
					{
						Day:      "Tuesday",
						Chapters: 2,
					},
				},
				Genres: []string{"Action"},
				Chapter_Draft: models.Chapter{
					Title:   "test book chapter",
					Content: "test book content",
				},
			},
			wantErr: true,
		},
		{
			name: "should create the book",
			book: &models.Book{
				Name:        "test book",
				Description: "test book description",
				Image: sql.NullString{
					String: "test book image",
					Valid:  true,
				},
				Author_Id: *author_id,
				Language:  "English",
				Release_schedule: []models.Schedule{
					{
						Day:      "Monday",
						Chapters: 1,
					},
					{
						Day:      "Tuesday",
						Chapters: 2,
					},
				},
				Genres: []string{"Action"},
				Chapter_Draft: models.Chapter{
					Title:   "test book chapter",
					Content: "test book content",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := db.UploadBook(context.TODO(), tt.book)

			if (err != nil) != tt.wantErr {
				t.Fatalf("wanted: %v, got: %v", tt.wantErr, err != nil)
			}

			if id != nil {
				_, err = db.DB.Exec("DELETE FROM books WHERE id = $1", id)
				if err != nil {
					t.Fatalf("error deleting book: %v", err)
				}
			}
		})
	}
}

func TestGetBooksStats(t *testing.T) {
	db := setUpTestDb(t)

	author_id, _ := db.CreateUser(context.TODO(), &models.User{
		Username: "fake_username",
		Email:    "fake_email@email.com",
		Password: "fake_password",
	})

	defer func() {
		db.DB.Exec("DELETE FROM users WHERE id = $1", author_id)
	}()

	tests := []struct {
		name      string
		author_id string
		wantErr   bool
	}{
		{
			name:      "should return an error if author doesn't exist",
			author_id: "",
			wantErr:   true,
		},
		{
			name:      "should return book stats",
			author_id: author_id.String(),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.GetBooksStats(context.TODO(), tt.author_id, 0, 5)

			if (err != nil && err != ErrCreatorsBooksNotFound) != tt.wantErr {
				t.Fatalf("wanted: %v, got: %v", tt.wantErr, (err != nil && err != ErrCreatorsBooksNotFound))
			}
		})
	}
}

func TestGetBooksByGenre(t *testing.T) {
	tests := []struct {
		name    string
		genres  []string
		wantErr bool
	}{
		{
			name:    "should return an error if genre does not exist",
			genres:  []string{"non-existent genre"},
			wantErr: true,
		},
		{
			name:    "should return books by genre",
			genres:  []string{"Fantasy"},
			wantErr: false,
		},
	}

	db := setUpTestDb(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.GetBooksByGenre(context.TODO(), tt.genres, 0, 5)

			if (err != nil && err != ErrNoBooksUnderThisGenre) != tt.wantErr {
				t.Fatalf("wanted: %v, got: %v", tt.wantErr, (err != nil && err != ErrNoBooksUnderThisGenre))
			}
		})
	}
}

func TestGetBooksByLanguage(t *testing.T) {
	tests := []struct {
		name      string
		languages []string
		wantErr   bool
	}{
		{
			name:      "should return an error if language does not exist",
			languages: []string{"non-existent language"},
			wantErr:   true,
		},
		{
			name:      "should return books by language",
			languages: []string{"English"},
			wantErr:   false,
		},
	}

	db := setUpTestDb(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.GetBooksByLanguage(context.TODO(), tt.languages, 0, 5)

			if (err != nil && err != ErrNoBooksUnderThisLanguage) != tt.wantErr {
				t.Fatalf("wanted: %v, got: %v", tt.wantErr, (err != nil && err != ErrNoBooksUnderThisLanguage))
			}
		})
	}
}

func TestGetBooksByGenreAndLanguage(t *testing.T) {
	tests := []struct {
		name      string
		genres    []string
		languages []string
		wantErr   bool
	}{
		{
			name:      "should return an error if genre does not exist",
			genres:    []string{"non-existent genre"},
			languages: []string{"Spanish"},
			wantErr:   true,
		},
		{
			name:      "should return an error if language does not exist",
			genres:    []string{"Fantasy"},
			languages: []string{"non-existent language"},
			wantErr:   true,
		},
		{
			name:      "should return books by genre and language",
			genres:    []string{"Fantasy"},
			languages: []string{"English"},
			wantErr:   false,
		},
	}

	db := setUpTestDb(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.GetBooksByGenreAndLanguage(context.TODO(), tt.genres, tt.languages, 0, 5)

			if (err != nil && err != ErrNoBooksUnderThisGenreOrLanguage) != tt.wantErr {
				t.Fatalf("wanted: %v, got: %v", tt.wantErr, (err != nil && err != ErrNoBooksUnderThisGenreOrLanguage))
			}
		})
	}
}

func TestGetBook(t *testing.T) {
	db := setUpTestDb(t)

	author_id, _ := db.CreateUser(context.TODO(), &models.User{
		Username: "fake_username",
		Email:    "fake_email@email.com",
		Password: "fake_password",
	})

	defer func() {
		db.DB.Exec("DELETE FROM users WHERE id = $1", author_id)
	}()

	book_id, err := db.UploadBook(context.TODO(), &models.Book{
		Name:        "test book",
		Description: "test book description",
		Image: sql.NullString{
			String: "test book image",
			Valid:  true,
		},
		Author_Id: *author_id,
		Language:  "English",
		Release_schedule: []models.Schedule{
			{
				Day:      "Monday",
				Chapters: 1,
			},
			{
				Day:      "Tuesday",
				Chapters: 2,
			},
		},
		Genres: []string{"Action"},
		Chapter_Draft: models.Chapter{
			Title:   "test book chapter",
			Content: "test book content",
		},
	},
	)

	if err != nil {
		t.Fatalf("error: %v", err)
	}

	db.DB.Exec(`
			UPDATE books
			SET approved = true
			WHERE id = $1;
		`, book_id)

	tests := []struct {
		name    string
		id      *uuid.UUID
		wantErr bool
	}{
		{
			name:    "should return an error if book is not found",
			id:      nil,
			wantErr: true,
		},
		{
			name:    "should return book",
			id:      book_id,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var book *models.Book
			var err error

			if tt.id == nil {
				book, err = db.GetBook(context.TODO(), "")
			} else {
				book, err = db.GetBook(context.TODO(), tt.id.String())
			}

			if (err != nil) != tt.wantErr {
				t.Fatalf("expected %v, got %v", tt.wantErr, err != nil)
			}

			if book != nil && tt.wantErr == false && book.Id != *book_id {
				t.Fatalf("expected %v, got %v", *book_id, book.Id)
			}
		})
	}
}

func TestEditBook(t *testing.T) {
	db := setUpTestDb(t)

	t.Run("should return an error if at least one field isn't passed", func(t *testing.T) {
		err := db.EditBook(context.TODO(), &models.HandleEditBookParam{})

		if (err != nil) != true {
			t.Fatalf("expected %v, got %v", true, err != nil)
		}
	})
}

func TestGetRecentReads(t *testing.T) {
	db := setUpTestDb(t)

	id, _ := db.CreateUser(context.TODO(), &models.User{
		Username: "fake_username",
		Email:    "fake_email@email.com",
		Password: "fake_password",
	})

	defer func() {
		db.DB.Exec("DELETE FROM users WHERE id = $1", id)
	}()

	t.Run("should return an error if recent books is empty", func(t *testing.T) { //this isn't a bad error btw
		_, err := db.GetRecentReads(context.TODO(), id.String(), 0, 5)

		if (err != nil) != true {
			t.Fatalf("expected %v, got %v", true, err != nil)
		}
	})
}
